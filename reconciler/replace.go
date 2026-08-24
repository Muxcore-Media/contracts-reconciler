package reconciler

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ApplyReplaceDirectives runs "go mod edit -replace" for each directive in a
// go.mod file. The workdir must contain the go.mod to modify.
//
// Existing replace directives for the same old path are overwritten.
func ApplyReplaceDirectives(workdir string, directives []ReplaceDirective) error {
	if len(directives) == 0 {
		return nil
	}

	modFile := filepath.Join(workdir, "go.mod")
	if _, err := os.Stat(modFile); os.IsNotExist(err) {
		return fmt.Errorf("go.mod not found at %s", modFile)
	}

	for _, d := range directives {
		replace := d.String()
		args := []string{"mod", "edit", "-replace", replace}

		cmd := exec.CommandContext(context.Background(), "go", args...) //nolint:gosec // controlled go mod edit args
		cmd.Dir = workdir
		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("go mod edit -replace %s: %s (%w)", replace, string(out), err)
		}
	}

	return nil
}

// GenerateReplaceBlock returns a multi-line string containing all replace
// directives in go.mod format. Useful for previewing or embedding in
// documentation.
func GenerateReplaceBlock(directives []ReplaceDirective) string {
	if len(directives) == 0 {
		return ""
	}

	var lines []string
	lines = append(lines, "replace (")
	for _, d := range directives {
		lines = append(lines, "\t"+d.String())
	}
	lines = append(lines, ")")
	return strings.Join(lines, "\n")
}

// DryRun returns a human-readable report of what would be changed without
// modifying any files.
func DryRun(declarations []Declaration) (string, error) {
	r := &Resolver{}
	directives, errs := r.ResolveAll(declarations)

	var out strings.Builder
	out.WriteString("=== Contract Reconciliation Dry Run ===\n\n")

	if len(errs) > 0 {
		out.WriteString("ERRORS:\n")
		for _, err := range errs {
			fmt.Fprintf(&out, "  %s\n", err)
		}
		out.WriteString("\n")
	}

	if len(directives) == 0 {
		out.WriteString("No contract normalization needed. All declarations use canonical repos.\n")
		return out.String(), nil
	}

	out.WriteString("REPLACE DIRECTIVES TO APPLY:\n")
	out.WriteString(GenerateReplaceBlock(directives))
	out.WriteString("\n\n")

	out.WriteString("SUMMARY:\n")
	for _, d := range directives {
		fmt.Fprintf(&out, "  %s → %s %s\n", d.OldPath, d.NewPath, d.Version)
	}

	return out.String(), nil
}
