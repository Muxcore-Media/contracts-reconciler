package reconciler

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Resolver checks third-party contract declarations against canonical
// equivalents and generates go.mod replace directives when interfaces match.
type Resolver struct {
	// CacheDir is where cloned contract repos are stored. Defaults to a
	// temp directory. Set this if you want to cache across invocations.
	CacheDir string
}

// Resolve checks whether a module's contract declaration is compatible with
// the canonical contract repo for the same interface name.
//
// It returns (nil, nil) when:
//   - The declaration already uses the canonical repo (no work needed)
//   - No canonical equivalent exists for this interface (not a MuxCore contract)
//
// It returns a ReplaceDirective when:
//   - The declaration uses a non-canonical repo
//   - The interface in both repos is structurally identical
//
// It returns an error when:
//   - The interface name isn't found in either repo
//   - The interfaces are structurally different (mismatch report)
//   - Either repo can't be fetched or parsed
func (r *Resolver) Resolve(decl Declaration) (*ReplaceDirective, error) {
	// Check if this is already a canonical declaration
	canon := Canonical(decl.Interface)
	if canon == nil {
		// Unknown interface — not a MuxCore contract, nothing to reconcile
		return nil, nil
	}

	if decl.Repo == canon.ImportPath {
		// Already using the canonical repo, no replace needed
		return nil, nil
	}

	// Fetch and parse the canonical interface
	canonSpec, err := r.fetchAndParse(canon.ImportPath, canon.Version, decl.Interface)
	if err != nil {
		return nil, fmt.Errorf("canonical %s: %w", decl.Interface, err)
	}

	// Fetch and parse the third-party interface
	thirdSpec, err := r.fetchAndParse(decl.Repo, decl.Version, decl.Interface)
	if err != nil {
		return nil, fmt.Errorf("third-party %s from %s: %w", decl.Interface, decl.Repo, err)
	}

	// Structural comparison
	if !canonSpec.Equal(*thirdSpec) {
		diff := diffSpecs(*canonSpec, *thirdSpec)
		return nil, fmt.Errorf("interface mismatch for %s:\n%s\nDeclared by %s does not match canonical %s\nUse type aliases (type X = canonical.X) or match the method signatures exactly", decl.Interface, diff, decl.Repo, canon.ImportPath)
	}

	return &ReplaceDirective{
		OldPath: decl.Repo,
		NewPath: canon.ImportPath,
		Version: canon.Version,
	}, nil
}

// ResolveAll processes a list of declarations and returns all applicable
// replace directives. Mismatches are collected as errors in the returned
// slice alongside successful directives.
func (r *Resolver) ResolveAll(decls []Declaration) ([]ReplaceDirective, []error) {
	var directives []ReplaceDirective
	var errs []error

	for _, decl := range decls {
		directive, err := r.Resolve(decl)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		if directive != nil {
			directives = append(directives, *directive)
		}
	}
	return directives, errs
}

func (r *Resolver) fetchAndParse(importPath, version, interfaceName string) (*InterfaceSpec, error) {
	dir, err := r.cloneRepo(importPath, version)
	if err != nil {
		return nil, err
	}

	spec, err := findInterfaceByName(dir, interfaceName)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", importPath, err)
	}

	// Override the package path with the actual import path
	spec.PackagePath = importPath
	return spec, nil
}

func (r *Resolver) cloneRepo(importPath, version string) (string, error) {
	cacheDir := r.CacheDir
	if cacheDir == "" {
		cacheDir = os.TempDir()
	}

	// Build a cache key from import path + version
	safePath := strings.NewReplacer("/", "-", ".", "-").Replace(importPath)
	cacheKey := filepath.Join(cacheDir, "contracts-reconciler-cache", safePath+"-"+version)

	// Check cache
	if info, err := os.Stat(cacheKey); err == nil && info.IsDir() {
		return cacheKey, nil
	}

	// Convert import path to repo URL
	repoURL := importPathToRepoURL(importPath)

	// Clone
	tmpDir, err := os.MkdirTemp(cacheDir, "reconciler-clone-*")
	if err != nil {
		return "", fmt.Errorf("temp dir: %w", err)
	}

	args := []string{"clone", "--depth", "1"}
	if version != "" {
		args = append(args, "--branch", version)
	}
	args = append(args, repoURL, tmpDir)

	cmd := exec.CommandContext(context.Background(), "git", args...) //nolint:noctx // offline git clone without request cancellation
	out, err := cmd.CombinedOutput()
	if err != nil {
		_ = os.RemoveAll(tmpDir)
		return "", fmt.Errorf("git clone %s: %s (%w)", repoURL, string(out), err)
	}

	// Move to cache on success
	_ = os.MkdirAll(filepath.Dir(cacheKey), 0o750)
	if err := os.Rename(tmpDir, cacheKey); err != nil {
		// Rename failed (cross-device?), use tmpDir directly
		return tmpDir, nil //nolint:nilerr // intentional fallback when cache rename is unavailable
	}

	return cacheKey, nil
}

func importPathToRepoURL(importPath string) string {
	parts := strings.Split(importPath, "/")
	if len(parts) < 3 {
		return "https://" + importPath
	}
	// github.com/owner/repo → https://github.com/owner/repo
	return "https://" + strings.Join(parts[:3], "/")
}

func diffSpecs(canon, third InterfaceSpec) string {
	var lines []string

	canonMethods := make(map[string]MethodSpec, len(canon.Methods))
	for _, m := range canon.Methods {
		canonMethods[m.Name] = m
	}

	for _, m := range third.Methods {
		cm, ok := canonMethods[m.Name]
		if !ok {
			lines = append(lines, fmt.Sprintf("  + %s (missing from canonical)", methodSig(m)))
			continue
		}
		if !cm.equalTypes(m) {
			lines = append(lines,
				fmt.Sprintf("  ~ %s", m.Name),
				fmt.Sprintf("      canonical: %s", methodSig(cm)),
				fmt.Sprintf("      declared:  %s", methodSig(m)),
			)
		}
		delete(canonMethods, m.Name)
	}

	for _, m := range canonMethods {
		lines = append(lines, fmt.Sprintf("  - %s (missing from declaration)", methodSig(m)))
	}

	return strings.Join(lines, "\n")
}

func methodSig(m MethodSpec) string {
	params := make([]string, len(m.Params))
	for i, p := range m.Params {
		params[i] = p.String()
	}
	results := make([]string, len(m.Results))
	for i, r := range m.Results {
		results[i] = r.String()
	}

	sig := m.Name + "(" + strings.Join(params, ", ") + ")"
	switch len(results) {
	case 0:
		return sig
	case 1:
		return sig + " " + results[0]
	default:
		return sig + " (" + strings.Join(results, ", ") + ")"
	}
}
