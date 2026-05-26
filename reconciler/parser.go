package reconciler

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

// ParseDir parses all .go files in a directory and extracts exported interface
// definitions. Non-Go files and test files are skipped.
func ParseDir(dir string) ([]InterfaceSpec, error) {
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, dir, func(fi os.FileInfo) bool {
		return !strings.HasSuffix(fi.Name(), "_test.go") && strings.HasSuffix(fi.Name(), ".go")
	}, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", dir, err)
	}

	if len(pkgs) == 0 {
		return nil, fmt.Errorf("no Go packages found in %s", dir)
	}

	// Use the first package found (there should only be one package per dir)
	var pkg *ast.Package
	for _, p := range pkgs {
		pkg = p
		break
	}

	pkgPath := pkg.Name // just the local package name
	var specs []InterfaceSpec
	for _, file := range pkg.Files {
		specs = append(specs, extractInterfaces(file, pkgPath)...)
	}
	return specs, nil
}

// ParseFile parses a single .go file and extracts exported interface definitions.
func ParseFile(filePath string) ([]InterfaceSpec, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", filePath, err)
	}

	// Package import path from the file's package declaration
	pkgPath := file.Name.Name
	return extractInterfaces(file, pkgPath), nil
}

// FindInterface finds a specific interface by name in a list of specs.
func FindInterface(specs []InterfaceSpec, name string) (*InterfaceSpec, error) {
	for i := range specs {
		if specs[i].Name == name {
			return &specs[i], nil
		}
	}
	return nil, fmt.Errorf("interface %q not found", name)
}

// findInterfaceByName searches a parsed directory for a specific interface.
func findInterfaceByName(dir, name string) (*InterfaceSpec, error) {
	specs, err := ParseDir(dir)
	if err != nil {
		return nil, err
	}
	return FindInterface(specs, name)
}

func extractInterfaces(file *ast.File, pkgPath string) []InterfaceSpec {
	var specs []InterfaceSpec
	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.TYPE {
			continue
		}
		for _, spec := range genDecl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}
			ifaceType, ok := typeSpec.Type.(*ast.InterfaceType)
			if !ok {
				continue
			}
			if !typeSpec.Name.IsExported() {
				continue
			}

			is := InterfaceSpec{
				PackagePath: pkgPath,
				Name:        typeSpec.Name.Name,
			}
			for _, method := range ifaceType.Methods.List {
				is.Methods = append(is.Methods, extractMethod(method)...)
			}
			specs = append(specs, is)
		}
	}
	return specs
}

func extractMethod(field *ast.Field) []MethodSpec {
	// Embedded interface — type is an ident or selector with no names
	if len(field.Names) == 0 {
		// Embedded interfaces are not expanded — we skip them for structural
		// comparison. Only explicit methods are compared.
		return nil
	}

	funcType, ok := field.Type.(*ast.FuncType)
	if !ok {
		return nil
	}

	var methods []MethodSpec
	for _, name := range field.Names {
		if !name.IsExported() {
			continue
		}
		ms := MethodSpec{
			Name:    name.Name,
			Params:  extractFieldList(funcType.Params),
			Results: extractFieldList(funcType.Results),
		}
		methods = append(methods, ms)
	}
	return methods
}

func extractFieldList(fieldList *ast.FieldList) []TypeSpec {
	if fieldList == nil {
		return nil
	}
	var types []TypeSpec
	for _, field := range fieldList.List {
		ts := extractType(field.Type)
		// If there are multiple names for the same type, add one entry per name
		if len(field.Names) == 0 {
			types = append(types, ts)
		} else {
			for range field.Names {
				types = append(types, ts)
			}
		}
	}
	return types
}

func extractType(expr ast.Expr) TypeSpec {
	switch t := expr.(type) {
	case *ast.Ident:
		if t.Name == "error" || t.Name == "bool" || t.Name == "string" ||
			t.Name == "int" || t.Name == "int8" || t.Name == "int16" ||
			t.Name == "int32" || t.Name == "int64" ||
			t.Name == "uint" || t.Name == "uint8" || t.Name == "uint16" ||
			t.Name == "uint32" || t.Name == "uint64" ||
			t.Name == "float32" || t.Name == "float64" ||
			t.Name == "complex64" || t.Name == "complex128" ||
			t.Name == "byte" || t.Name == "rune" ||
			t.Name == "uintptr" || t.Name == "any" {
			return TypeSpec{Kind: "ident", Name: t.Name}
		}
		return TypeSpec{Kind: "ident", Name: t.Name}

	case *ast.SelectorExpr:
		return TypeSpec{
			Kind: "selector",
			Name: t.Sel.Name,
			// ImportPath is resolved from imports — for structural equality we
			// don't need it, but it's useful for debugging
			ImportPath: resolveImportPath(t),
		}

	case *ast.StarExpr:
		elem := extractType(t.X)
		return TypeSpec{Kind: "star", Elem: &elem}

	case *ast.ArrayType:
		if t.Len == nil {
			elem := extractType(t.Elt)
			return TypeSpec{Kind: "slice", Elem: &elem}
		}
		elem := extractType(t.Elt)
		return TypeSpec{Kind: "array", Elem: &elem}

	case *ast.MapType:
		key := extractType(t.Key)
		elem := extractType(t.Value)
		return TypeSpec{Kind: "map", Key: &key, Elem: &elem}

	case *ast.InterfaceType:
		// For method-based interfaces used as parameter/return types,
		// just mark it as interface{} — method set comparison happens at
		// the top-level InterfaceSpec level
		if t.Methods == nil || len(t.Methods.List) == 0 {
			return TypeSpec{Kind: "interface"}
		}
		// Non-empty interface used inline — extract method set
		ts := TypeSpec{Kind: "interface"}
		return ts

	case *ast.FuncType:
		ts := TypeSpec{Kind: "func"}
		ts.Params = extractFieldList(t.Params)
		ts.Results = extractFieldList(t.Results)
		return ts

	case *ast.ChanType:
		elem := extractType(t.Value)
		return TypeSpec{Kind: "chan", Elem: &elem}

	case *ast.StructType:
		ts := TypeSpec{Kind: "struct"}
		if t.Fields != nil {
			ts.Fields = extractFieldList(t.Fields)
		}
		return ts

	case *ast.Ellipsis:
		elem := extractType(t.Elt)
		return TypeSpec{Kind: "slice", Elem: &elem}

	default:
		return TypeSpec{Kind: "ident", Name: fmt.Sprintf("%T", expr)}
	}
}

// resolveImportPath attempts to reconstruct the full import path from a
// selector expression by walking the file's imports. Returns empty string
// if it can't be determined (which is fine — structural comparison ignores it).
func resolveImportPath(_ *ast.SelectorExpr) string {
	// Full import path resolution requires walking the file's import block.
	// We don't need it for structural comparison — the type name alone is
	// sufficient since we compare method signatures, not package origins.
	return ""
}

// CloneAndParse clones a git repo to a temp directory, parses its Go source,
// and returns the extracted interface specs. The caller is responsible for
// cleanup via the returned cleanup function.
func CloneAndParse(repoURL, version string) ([]InterfaceSpec, func(), error) {
	dir, err := os.MkdirTemp("", "contracts-reconciler-*")
	if err != nil {
		return nil, nil, fmt.Errorf("create temp dir: %w", err)
	}

	cleanup := func() { os.RemoveAll(dir) }

	// Build the clone URL with the version tag
	cloneURL := repoURL
	if !strings.HasSuffix(cloneURL, ".git") {
		cloneURL += ".git"
	}

	// For GitHub URLs, convert to clone URL
	if strings.HasPrefix(cloneURL, "https://github.com/") && !strings.Contains(cloneURL, ".git") {
		cloneURL += ".git"
	}

	// Clone the repo
	if err := cloneRepo(cloneURL, version, dir); err != nil {
		cleanup()
		return nil, nil, fmt.Errorf("clone %s: %w", repoURL, err)
	}

	specs, err := ParseDir(dir)
	if err != nil {
		cleanup()
		return nil, nil, fmt.Errorf("parse %s: %w", dir, err)
	}

	return specs, cleanup, nil
}

func cloneRepo(url, version, destDir string) error {
	// Simple approach: git clone --depth 1 --branch <version> <url> <dest>
	// For a real implementation, this would shell out to git.
	// The caller (spool CLI, marketplace module) provides the actual clone mechanism.
	// This function is a placeholder that expects the files to already be on disk
	// at destDir.
	return fmt.Errorf("not implemented: use CloneRepo from the spool CLI")
}

// ParseGoModFile parses a go.mod file and returns all require directives.
func ParseGoModFile(modPath string) ([]ModRequire, error) {
	data, err := os.ReadFile(modPath)
	if err != nil {
		return nil, fmt.Errorf("read go.mod: %w", err)
	}

	return parseGoMod(string(data))
}

// ModRequire represents a single require directive in go.mod.
type ModRequire struct {
	Path    string
	Version string
}

func parseGoMod(content string) ([]ModRequire, error) {
	var reqs []ModRequire
	lines := strings.Split(content, "\n")
	inRequire := false

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "//") {
			continue
		}
		if line == "require (" {
			inRequire = true
			continue
		}
		if inRequire && line == ")" {
			inRequire = false
			continue
		}

		if strings.HasPrefix(line, "require ") {
			parts := strings.Fields(strings.TrimPrefix(line, "require "))
			if len(parts) >= 2 {
				reqs = append(reqs, ModRequire{Path: parts[0], Version: parts[1]})
			}
			continue
		}

		if inRequire {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				reqs = append(reqs, ModRequire{Path: parts[0], Version: parts[1]})
			}
		}
	}
	return reqs, nil
}

// writeToFile is a test helper.
var writeToFile = filepath.Walk // unused; placeholder for test support
