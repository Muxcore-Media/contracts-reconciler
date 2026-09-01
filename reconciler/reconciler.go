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

	// AllowedHosts restricts git clone targets. Defaults to github.com.
	AllowedHosts []string
}

// Resolve checks whether a module's contract declaration is compatible with
// the canonical contract repo for the same interface name.
//
// It returns (nil, nil) when:
//   - The declaration already uses the canonical repo (no work needed)
//   - No canonical equivalent exists for this interface (not a MuxCore contract)
//   - The interface is reserved (unpublished / events-only surface)
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
	if err := r.validateRepoImportPath(decl.Repo); err != nil {
		return nil, fmt.Errorf("third-party repo: %w", err)
	}

	canon := Canonical(decl.Interface)
	if canon == nil {
		return nil, nil
	}
	if canon.Reserved {
		return nil, nil
	}

	if decl.Repo == canon.ImportPath {
		return nil, nil
	}

	canonVersion := resolveCompareVersion(decl, *canon)

	canonSpec, err := r.fetchAndParse(canon.ImportPath, canonVersion, decl.Interface)
	if err != nil {
		return nil, fmt.Errorf("canonical %s: %w", decl.Interface, err)
	}

	thirdVersion := decl.Version
	if thirdVersion == "" {
		thirdVersion = canonVersion
	}
	thirdSpec, err := r.fetchAndParse(decl.Repo, thirdVersion, decl.Interface)
	if err != nil {
		return nil, fmt.Errorf("third-party %s from %s: %w", decl.Interface, decl.Repo, err)
	}

	if !canonSpec.Equal(*thirdSpec) {
		diff := diffSpecs(*canonSpec, *thirdSpec)
		return nil, fmt.Errorf("interface mismatch for %s:\n%s\nDeclared by %s does not match canonical %s\nUse type aliases (type X = canonical.X) or match the method signatures exactly", decl.Interface, diff, decl.Repo, canon.ImportPath)
	}

	pinVersion := canonVersion
	if pinVersion == "" {
		pinVersion = decl.Version
	}

	return &ReplaceDirective{
		OldPath: decl.Repo,
		NewPath: canon.ImportPath,
		Version: pinVersion,
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

func resolveCompareVersion(decl Declaration, canon CanonicalRepo) string {
	if decl.CanonicalVersion != "" {
		return decl.CanonicalVersion
	}
	if decl.Version != "" {
		return decl.Version
	}
	return canon.Version
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

	spec.PackagePath = importPath
	return spec, nil
}

func (r *Resolver) validateRepoImportPath(importPath string) error {
	return validateImportPath(importPath, r.allowedHosts())
}

func (r *Resolver) allowedHosts() []string {
	if len(r.AllowedHosts) > 0 {
		return r.AllowedHosts
	}
	return defaultAllowedHosts
}

func (r *Resolver) cloneRepo(importPath, version string) (string, error) {
	if err := r.validateRepoImportPath(importPath); err != nil {
		return "", err
	}

	cacheDir := r.CacheDir
	if cacheDir == "" {
		cacheDir = os.TempDir()
	}

	safePath := strings.NewReplacer("/", "-", ".", "-").Replace(importPath)
	cacheKey := filepath.Join(cacheDir, "contracts-reconciler-cache", safePath+"-"+version)

	if info, err := os.Stat(cacheKey); err == nil && info.IsDir() {
		return cacheKey, nil
	}

	repoURL, err := importPathToRepoURL(importPath)
	if err != nil {
		return "", err
	}

	tmpDir, err := os.MkdirTemp(cacheDir, "reconciler-clone-*")
	if err != nil {
		return "", fmt.Errorf("temp dir: %w", err)
	}

	args := []string{"clone", "--depth", "1"}
	if version != "" {
		args = append(args, "--branch", version)
	}
	args = append(args, repoURL, tmpDir)

	cmd := exec.CommandContext(context.Background(), "git", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		_ = os.RemoveAll(tmpDir)
		return "", fmt.Errorf("git clone %s: %s (%w)", repoURL, string(out), err)
	}

	_ = os.MkdirAll(filepath.Dir(cacheKey), 0o750)
	if err := os.Rename(tmpDir, cacheKey); err != nil {
		return tmpDir, nil //nolint:nilerr // intentional fallback when cache rename is unavailable
	}

	return cacheKey, nil
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
