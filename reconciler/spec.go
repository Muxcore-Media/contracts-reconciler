// Package reconciler provides structural contract matching and go.mod replace
// directive generation for MuxCore third-party module imports.
//
// When a module declares it implements "MediaAdminService" from
// "github.com/some-dev/contracts-media-admin", the reconciler fetches both that repo
// and the canonical "github.com/Muxcore-Media/contracts-media-admin", extracts the
// Go interface definitions via AST parsing, compares method signatures, and
// — if structurally compatible — generates a go.mod replace directive that
// normalizes the import to the canonical path.
//
// This preserves Go's nominal type safety while honoring the MuxCore philosophy:
// contracts are patterns, not org-bound dependencies.
package reconciler

import (
	"fmt"
	"strings"
)

// Declaration describes a contract a module claims to implement.
// This is what appears in a module's muxcore.json or ContractDeclaration list.
type Declaration struct {
	Repo             string // Go module path (e.g. "github.com/some-dev/contracts-media-admin")
	Version          string // semantic version tag (e.g. "v1.2.0")
	CanonicalVersion string // optional canonical tag override for comparison/pinning
	Interface        string // Go interface name (e.g. "MediaAdminServiceServer")
}

// CanonicalRepo maps an interface name to its canonical Go module path.
// The reconciler uses this to know which Muxcore-Media contract repo is the
// authority for a given interface.
type CanonicalRepo struct {
	ImportPath string // Go module import path (e.g. "github.com/Muxcore-Media/contracts-media-admin")
	Version    string // default clone tag when Declaration.Version is empty
	Reserved   bool   // unpublished or events-only — Resolve skips without cloning
}

// ReplaceDirective represents a single go.mod replace directive.
type ReplaceDirective struct {
	OldPath string // the module path to replace
	NewPath string // the canonical module path
	Version string // version to pin (e.g. "v1.0.0")
}

// String formats the directive as a human-readable go.mod replace line.
func (d ReplaceDirective) String() string {
	if d.Version != "" {
		return fmt.Sprintf("%s => %s %s", d.OldPath, d.NewPath, d.Version)
	}
	return fmt.Sprintf("%s => %s", d.OldPath, d.NewPath)
}

// ModEditArg formats the directive for `go mod edit -replace=...`.
func (d ReplaceDirective) ModEditArg() string {
	if d.Version != "" {
		return fmt.Sprintf("%s=%s@%s", d.OldPath, d.NewPath, d.Version)
	}
	return fmt.Sprintf("%s=%s", d.OldPath, d.NewPath)
}

// ParseReplaceDirective parses a "go mod edit -replace" output line.
func ParseReplaceDirective(line string) (ReplaceDirective, error) {
	line = strings.TrimSpace(line)
	if !strings.HasPrefix(line, "-replace ") {
		line = strings.TrimSpace(strings.TrimPrefix(line, "replace "))
	}
	line = strings.TrimPrefix(line, "-replace ")
	line = strings.TrimPrefix(line, "replace ")

	parts := strings.SplitN(line, "=>", 2)
	if len(parts) != 2 {
		return ReplaceDirective{}, fmt.Errorf("invalid replace directive: %s", line)
	}

	oldP := strings.TrimSpace(parts[0])
	right := strings.TrimSpace(parts[1])

	// right is either "new/path" or "new/path version"
	rightParts := strings.Fields(right)
	if len(rightParts) == 2 {
		return ReplaceDirective{OldPath: oldP, NewPath: rightParts[0], Version: rightParts[1]}, nil
	}
	return ReplaceDirective{OldPath: oldP, NewPath: right}, nil
}

// MethodSpec describes a single method in a Go interface.
type MethodSpec struct {
	Name    string     // method name
	Params  []TypeSpec // parameter types (receiver excluded)
	Results []TypeSpec // return types
}

// TypeSpec describes a Go type for structural comparison.
// Only carries enough information to match signatures — two types with
// different package paths but identical structure are considered compatible.
type TypeSpec struct {
	Kind       string     // "ident", "selector", "star", "array", "slice", "map", "interface", "chan", "func", "struct"
	Name       string     // package-local name (e.g. "MediaObject", "MediaFilter")
	ImportPath string     // for selectors: the package import path (e.g. "github.com/Muxcore-Media/contracts-media-admin")
	Elem       *TypeSpec  // element type for pointers, slices, arrays
	Key        *TypeSpec  // key type for maps
	Params     []TypeSpec // for func types
	Results    []TypeSpec // for func types
	Fields     []TypeSpec // for struct types
}

// InterfaceSpec describes a Go interface for structural comparison.
type InterfaceSpec struct {
	PackagePath string       // the Go import path of the package
	Name        string       // interface name
	Methods     []MethodSpec // methods (excluding embedded interfaces)
}

// Equal checks whether two interface specs are structurally identical.
// Package paths are NOT compared — only method names, parameter types, and
// return types matter. This is what makes third-party contracts compatible
// with canonical ones: same pattern, different origin.
func (s InterfaceSpec) Equal(other InterfaceSpec) bool {
	if s.Name != other.Name {
		return false
	}
	if len(s.Methods) != len(other.Methods) {
		return false
	}

	methodMap := make(map[string]MethodSpec, len(s.Methods))
	for _, m := range s.Methods {
		methodMap[m.Name] = m
	}

	for _, m := range other.Methods {
		sm, ok := methodMap[m.Name]
		if !ok {
			return false
		}
		if !sm.equalTypes(m) {
			return false
		}
	}
	return true
}

func (m MethodSpec) equalTypes(other MethodSpec) bool {
	if len(m.Params) != len(other.Params) {
		return false
	}
	if len(m.Results) != len(other.Results) {
		return false
	}
	for i := range m.Params {
		if !m.Params[i].Equal(other.Params[i]) {
			return false
		}
	}
	for i := range m.Results {
		if !m.Results[i].Equal(other.Results[i]) {
			return false
		}
	}
	return true
}

// Equal checks structural type equality, ignoring package paths.
// Two types from different packages that resolve to the same shape are equal.
func (t TypeSpec) Equal(other TypeSpec) bool { //nolint:gocyclo // structural type equality mirrors Go type grammar
	if t.Kind != other.Kind {
		return false
	}

	switch t.Kind {
	case "ident":
		return t.Name == other.Name
	case "selector":
		// Compare only the type name, not the import path
		return t.Name == other.Name
	case "star", "slice", "array":
		if t.Elem == nil || other.Elem == nil {
			return t.Elem == nil && other.Elem == nil
		}
		return t.Elem.Equal(*other.Elem)
	case "map":
		if (t.Key == nil) != (other.Key == nil) {
			return false
		}
		if (t.Elem == nil) != (other.Elem == nil) {
			return false
		}
		if t.Key != nil && !t.Key.Equal(*other.Key) {
			return false
		}
		if t.Elem != nil && !t.Elem.Equal(*other.Elem) {
			return false
		}
		return true
	case "interface":
		return true // empty interface{}, any method set mismatch is covered above
	case "func":
		if len(t.Params) != len(other.Params) {
			return false
		}
		if len(t.Results) != len(other.Results) {
			return false
		}
		for i := range t.Params {
			if !t.Params[i].Equal(other.Params[i]) {
				return false
			}
		}
		for i := range t.Results {
			if !t.Results[i].Equal(other.Results[i]) {
				return false
			}
		}
		return true
	case "chan":
		if t.Elem == nil || other.Elem == nil {
			return t.Elem == nil && other.Elem == nil
		}
		return t.Elem.Equal(*other.Elem)
	case "struct":
		if len(t.Fields) != len(other.Fields) {
			return false
		}
		for i := range t.Fields {
			if !t.Fields[i].Equal(other.Fields[i]) {
				return false
			}
		}
		return true
	default:
		return false
	}
}

// String returns a human-readable representation of the type.
func (t TypeSpec) String() string {
	switch t.Kind {
	case "ident":
		return t.Name
	case "selector":
		return t.Name
	case "star":
		return "*" + t.Elem.String()
	case "slice":
		return "[]" + t.Elem.String()
	case "array":
		return "[...]" + t.Elem.String()
	case "map":
		return "map[" + t.Key.String() + "]" + t.Elem.String()
	case "interface":
		return "interface{}"
	case "func":
		return t.funcString()
	case "chan":
		return "chan " + t.Elem.String()
	case "struct":
		return t.structString()
	default:
		return t.Kind
	}
}

func (t TypeSpec) funcString() string {
	params := make([]string, len(t.Params))
	for i, p := range t.Params {
		params[i] = p.String()
	}
	results := make([]string, len(t.Results))
	for i, r := range t.Results {
		results[i] = r.String()
	}
	switch len(results) {
	case 0:
		return "func(" + strings.Join(params, ", ") + ")"
	case 1:
		return "func(" + strings.Join(params, ", ") + ") " + results[0]
	default:
		return "func(" + strings.Join(params, ", ") + ") (" + strings.Join(results, ", ") + ")"
	}
}

func (t TypeSpec) structString() string {
	fields := make([]string, len(t.Fields))
	for i, f := range t.Fields {
		fields[i] = f.String()
	}
	return "struct{" + strings.Join(fields, "; ") + "}"
}
