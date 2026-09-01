package reconciler

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// ParseDir walks dir and all subdirectories, parsing every Go package found,
// and returns exported interface definitions from the entire tree.
func ParseDir(dir string) ([]InterfaceSpec, error) {
	var specs []InterfaceSpec
	found := false

	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if !d.IsDir() {
			return nil
		}
		if d.Name() == ".git" || d.Name() == "vendor" {
			return filepath.SkipDir
		}

		pkgSpecs, err := parsePackageDir(path)
		if err != nil {
			if strings.Contains(err.Error(), "no Go packages") {
				return nil
			}
			return err
		}
		if len(pkgSpecs) > 0 {
			found = true
			specs = append(specs, pkgSpecs...)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk %s: %w", dir, err)
	}
	if !found {
		return nil, fmt.Errorf("no Go packages found in %s", dir)
	}
	return specs, nil
}

// parsePackageDir parses a single directory as one Go package.
func parsePackageDir(dir string) ([]InterfaceSpec, error) {
	fset := token.NewFileSet()
	//nolint:staticcheck // ParseDir is enough for structural interface extraction without build tags.
	pkgs, err := parser.ParseDir(fset, dir, func(fi os.FileInfo) bool {
		return !strings.HasSuffix(fi.Name(), "_test.go") && strings.HasSuffix(fi.Name(), ".go")
	}, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", dir, err)
	}
	if len(pkgs) == 0 {
		return nil, fmt.Errorf("no Go packages in %s", dir)
	}

	var specs []InterfaceSpec
	for _, pkg := range pkgs {
		pkgPath := pkg.Name
		for _, file := range pkg.Files {
			specs = append(specs, extractInterfaces(file, pkgPath)...)
		}
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

// findInterfaceByName searches a parsed directory tree for a specific interface.
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
	if len(field.Names) == 0 {
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
		return TypeSpec{Kind: "ident", Name: t.Name}

	case *ast.SelectorExpr:
		return TypeSpec{
			Kind:       "selector",
			Name:       t.Sel.Name,
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
		if t.Methods == nil || len(t.Methods.List) == 0 {
			return TypeSpec{Kind: "interface"}
		}
		return TypeSpec{Kind: "interface"}

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

func resolveImportPath(_ *ast.SelectorExpr) string {
	return ""
}

// CloneAndParse clones a git repo to a temp directory, parses its Go source tree,
// and returns the extracted interface specs. The caller is responsible for cleanup
// via the returned cleanup function.
func CloneAndParse(repoURL, version string) ([]InterfaceSpec, func(), error) {
	dir, err := os.MkdirTemp("", "contracts-reconciler-*")
	if err != nil {
		return nil, nil, fmt.Errorf("create temp dir: %w", err)
	}

	cleanup := func() { _ = os.RemoveAll(dir) }

	if cloneErr := gitCloneBranch(repoURL, version, dir); cloneErr != nil {
		cleanup()
		return nil, nil, fmt.Errorf("clone %s: %w", repoURL, cloneErr)
	}

	specs, err := ParseDir(dir)
	if err != nil {
		cleanup()
		return nil, nil, fmt.Errorf("parse %s: %w", dir, err)
	}

	return specs, cleanup, nil
}

// ParseGoModFile parses a go.mod file and returns all require directives.
func ParseGoModFile(modPath string) ([]ModRequire, error) {
	data, err := os.ReadFile(modPath) //nolint:gosec // modPath supplied by caller for explicit go.mod parsing
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
