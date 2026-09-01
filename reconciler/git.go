package reconciler

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

var defaultAllowedHosts = []string{"github.com"}

// validateImportPath rejects unsafe or unsupported module paths before git clone.
func validateImportPath(importPath string, allowedHosts []string) error {
	if importPath == "" {
		return fmt.Errorf("empty import path")
	}
	if strings.Contains(importPath, "://") {
		return fmt.Errorf("scheme URLs are not allowed: %s", importPath)
	}
	if strings.HasPrefix(importPath, "/") || strings.HasPrefix(importPath, ".") {
		return fmt.Errorf("relative import paths are not allowed: %s", importPath)
	}
	parts := strings.Split(importPath, "/")
	if len(parts) < 3 {
		return fmt.Errorf("import path must be host/owner/repo: %s", importPath)
	}
	host := parts[0]
	if allowedHosts == nil {
		allowedHosts = defaultAllowedHosts
	}
	for _, h := range allowedHosts {
		if host == h {
			return nil
		}
	}
	return fmt.Errorf("host %q is not in the clone allow-list", host)
}

func importPathToRepoURL(importPath string) (string, error) {
	if err := validateImportPath(importPath, defaultAllowedHosts); err != nil {
		return "", err
	}
	parts := strings.Split(importPath, "/")
	return "https://" + strings.Join(parts[:3], "/"), nil
}

func normalizeCloneURL(repoURL string) string {
	cloneURL := strings.TrimSpace(repoURL)
	if strings.HasPrefix(cloneURL, "https://") || strings.HasPrefix(cloneURL, "git@") {
		if !strings.HasSuffix(cloneURL, ".git") {
			cloneURL += ".git"
		}
		return cloneURL
	}
	// Treat as module import path.
	url, err := importPathToRepoURL(cloneURL)
	if err != nil {
		return cloneURL
	}
	if !strings.HasSuffix(url, ".git") {
		url += ".git"
	}
	return url
}

func gitCloneBranch(repoURL, version, destDir string) error {
	cloneURL := normalizeCloneURL(repoURL)
	args := []string{"clone", "--depth", "1"}
	if version != "" {
		args = append(args, "--branch", version)
	}
	args = append(args, cloneURL, destDir)

	cmd := exec.CommandContext(context.Background(), "git", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git clone %s: %s (%w)", cloneURL, string(out), err)
	}
	return nil
}
