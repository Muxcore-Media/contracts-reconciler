package reconciler

import (
	"os"

	"strings"
	"testing"
)

func TestInterfaceSpecEqual(t *testing.T) {
	tests := []struct {
		name string
		a    InterfaceSpec
		b    InterfaceSpec
		want bool
	}{
		{
			name: "identical",
			a: InterfaceSpec{
				Name: "Foo",
				Methods: []MethodSpec{
					{Name: "Bar", Params: []TypeSpec{{Kind: "ident", Name: "string"}}, Results: []TypeSpec{{Kind: "ident", Name: "error"}}},
				},
			},
			b: InterfaceSpec{
				Name: "Foo",
				Methods: []MethodSpec{
					{Name: "Bar", Params: []TypeSpec{{Kind: "ident", Name: "string"}}, Results: []TypeSpec{{Kind: "ident", Name: "error"}}},
				},
			},
			want: true,
		},
		{
			name: "different name",
			a:    InterfaceSpec{Name: "Foo"},
			b:    InterfaceSpec{Name: "Bar"},
			want: false,
		},
		{
			name: "different method count",
			a: InterfaceSpec{
				Name: "Foo",
				Methods: []MethodSpec{
					{Name: "Bar"},
					{Name: "Baz"},
				},
			},
			b: InterfaceSpec{
				Name: "Foo",
				Methods: []MethodSpec{
					{Name: "Bar"},
				},
			},
			want: false,
		},
		{
			name: "different method signatures",
			a: InterfaceSpec{
				Name: "Foo",
				Methods: []MethodSpec{
					{Name: "Bar", Params: []TypeSpec{{Kind: "ident", Name: "string"}}},
				},
			},
			b: InterfaceSpec{
				Name: "Foo",
				Methods: []MethodSpec{
					{Name: "Bar", Params: []TypeSpec{{Kind: "ident", Name: "int"}}},
				},
			},
			want: false,
		},
		{
			name: "same methods different order",
			a: InterfaceSpec{
				Name: "Foo",
				Methods: []MethodSpec{
					{Name: "Bar"},
					{Name: "Baz"},
				},
			},
			b: InterfaceSpec{
				Name: "Foo",
				Methods: []MethodSpec{
					{Name: "Baz"},
					{Name: "Bar"},
				},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.a.Equal(tt.b)
			if got != tt.want {
				t.Errorf("Equal() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTypeSpecEqual(t *testing.T) {
	tests := []struct {
		name string
		a    TypeSpec
		b    TypeSpec
		want bool
	}{
		{
			name: "same selector different package",
			a:    TypeSpec{Kind: "selector", Name: "Reader"},
			b:    TypeSpec{Kind: "selector", Name: "Reader"},
			want: true,
		},
		{
			name: "different selector names",
			a:    TypeSpec{Kind: "selector", Name: "Reader"},
			b:    TypeSpec{Kind: "selector", Name: "Writer"},
			want: false,
		},
		{
			name: "same pointer type",
			a:    TypeSpec{Kind: "star", Elem: &TypeSpec{Kind: "ident", Name: "int"}},
			b:    TypeSpec{Kind: "star", Elem: &TypeSpec{Kind: "ident", Name: "int"}},
			want: true,
		},
		{
			name: "slice vs pointer",
			a:    TypeSpec{Kind: "slice", Elem: &TypeSpec{Kind: "ident", Name: "string"}},
			b:    TypeSpec{Kind: "star", Elem: &TypeSpec{Kind: "ident", Name: "string"}},
			want: false,
		},
		{
			name: "same func type",
			a: TypeSpec{
				Kind:    "func",
				Params:  []TypeSpec{{Kind: "ident", Name: "string"}},
				Results: []TypeSpec{{Kind: "ident", Name: "error"}},
			},
			b: TypeSpec{
				Kind:    "func",
				Params:  []TypeSpec{{Kind: "ident", Name: "string"}},
				Results: []TypeSpec{{Kind: "ident", Name: "error"}},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.a.Equal(tt.b)
			if got != tt.want {
				t.Errorf("Equal() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseReplaceDirective(t *testing.T) {
	tests := []struct {
		input   string
		want    ReplaceDirective
		wantErr bool
	}{
		{
			input: "github.com/foo/bar => github.com/Muxcore-Media/bar v1.0.0",
			want:  ReplaceDirective{OldPath: "github.com/foo/bar", NewPath: "github.com/Muxcore-Media/bar", Version: "v1.0.0"},
		},
		{
			input: "github.com/foo/bar => github.com/Muxcore-Media/bar",
			want:  ReplaceDirective{OldPath: "github.com/foo/bar", NewPath: "github.com/Muxcore-Media/bar"},
		},
		{
			input:   "garbage",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ParseReplaceDirective(tt.input)
			if tt.wantErr && err == nil {
				t.Errorf("expected error, got nil")
				return
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if err != nil {
				return
			}
			if got.OldPath != tt.want.OldPath || got.NewPath != tt.want.NewPath || got.Version != tt.want.Version {
				t.Errorf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestParseGoMod(t *testing.T) {
	content := `module example.com/test

go 1.23

require (
    github.com/Muxcore-Media/contracts-media-admin v0.1.0
    github.com/some-dev/contracts-media-admin v1.2.0
)

require golang.org/x/mod v0.22.0
`
	reqs, err := parseGoMod(content)
	if err != nil {
		t.Fatalf("parseGoMod: %v", err)
	}
	if len(reqs) != 3 {
		t.Fatalf("expected 3 requires, got %d", len(reqs))
	}
	found := false
	for _, r := range reqs {
		if r.Path == "github.com/some-dev/contracts-media-admin" && r.Version == "v1.2.0" {
			found = true
		}
	}
	if !found {
		t.Errorf("third-party require not found: %+v", reqs)
	}
}

func TestCanonicalRegistry(t *testing.T) {
	// Every canonical entry must have a valid import path
	for name, cr := range canonicalRegistry {
		if cr.ImportPath == "" {
			t.Errorf("%s: empty import path", name)
		}
		if !strings.HasPrefix(cr.ImportPath, "github.com/Muxcore-Media/contracts-") {
			t.Errorf("%s: import path should start with github.com/Muxcore-Media/contracts-: %s", name, cr.ImportPath)
		}
	}

	// Common interfaces must exist
	for _, name := range []string{"MediaAdminService", "Downloader", "Indexer", "NotificationProvider"} {
		if cr := Canonical(name); cr == nil {
			t.Errorf("Canonical(%q) returned nil — interface must be registered", name)
		}
	}
}

func TestParseDir_MediaAdminContracts(t *testing.T) {
	// Test parsing the actual contracts-media-admin repo
	repoDir := "/opt/repos/contracts-media-admin"
	if _, err := os.Stat(repoDir); os.IsNotExist(err) {
		t.Skip("contracts-media-admin not cloned — skipping integration test")
	}

	specs, err := ParseDir(repoDir)
	if err != nil {
		t.Fatalf("ParseDir(%s): %v", repoDir, err)
	}

	// contracts-media-admin should have MediaAdminServiceServer
	found := false
	for _, s := range specs {
		if s.Name == "MediaAdminServiceServer" {
			found = true
			if len(s.Methods) == 0 {
				t.Error("MediaAdminServiceServer has no methods — check parser")
			}
			t.Logf("MediaAdminServiceServer methods: %d", len(s.Methods))
			for _, m := range s.Methods {
				t.Logf("  %s", methodSig(m))
			}
		}
	}
	if !found {
		t.Error("MediaAdminServiceServer interface not found in contracts-media-admin")
	}
}

func TestParseDir_ContentContracts(t *testing.T) {
	repoDir := "/opt/repos/contracts-content"
	if _, err := os.Stat(repoDir); os.IsNotExist(err) {
		t.Skip("contracts-content not cloned — skipping integration test")
	}

	specs, err := ParseDir(repoDir)
	if err != nil {
		t.Fatalf("ParseDir(%s): %v", repoDir, err)
	}

	found := false
	for _, s := range specs {
		if s.Name == "SupplementaryContentProvider" {
			found = true
			t.Logf("SupplementaryContentProvider methods: %d", len(s.Methods))
		}
	}
	if !found {
		t.Error("SupplementaryContentProvider interface not found in contracts-content")
	}
}

func TestDryRun(t *testing.T) {
	report, err := DryRun([]Declaration{
		{Repo: "github.com/Muxcore-Media/contracts-media-admin", Version: "v0.1.0", Interface: "MediaAdminService"},
	})
	if err != nil {
		t.Fatalf("DryRun: %v", err)
	}
	if !strings.Contains(report, "No contract normalization needed") {
		t.Errorf("expected no-op report, got: %s", report)
	}
}

func TestReplaceDirectiveString(t *testing.T) {
	d := ReplaceDirective{
		OldPath: "github.com/thirdparty/contracts-media-admin",
		NewPath: "github.com/Muxcore-Media/contracts-media-admin",
		Version: "v0.1.0",
	}
	got := d.String()
	want := "github.com/thirdparty/contracts-media-admin => github.com/Muxcore-Media/contracts-media-admin v0.1.0"
	if got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
}

func TestGenerateReplaceBlock(t *testing.T) {
	directives := []ReplaceDirective{
		{OldPath: "github.com/foo/A", NewPath: "github.com/Muxcore-Media/A", Version: "v1.0.0"},
		{OldPath: "github.com/bar/B", NewPath: "github.com/Muxcore-Media/B", Version: "v1.0.0"},
	}
	block := GenerateReplaceBlock(directives)
	if !strings.Contains(block, "replace (") {
		t.Error("missing replace block header")
	}
	if !strings.Contains(block, "github.com/foo/A => github.com/Muxcore-Media/A v1.0.0") {
		t.Error("missing first replace directive")
	}
	if !strings.Contains(block, "github.com/bar/B => github.com/Muxcore-Media/B v1.0.0") {
		t.Error("missing second replace directive")
	}
	t.Logf("Block:\n%s", block)
}

func TestGenerateReplaceBlockEmpty(t *testing.T) {
	block := GenerateReplaceBlock(nil)
	if block != "" {
		t.Errorf("expected empty block, got: %s", block)
	}
}

func TestResolve_AlreadyCanonical(t *testing.T) {
	r := &Resolver{}
	directive, err := r.Resolve(Declaration{
		Repo:      "github.com/Muxcore-Media/contracts-media-admin",
		Version:   "v0.1.0",
		Interface: "MediaAdminService",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if directive != nil {
		t.Errorf("expected nil directive for already-canonical, got %+v", directive)
	}
}

func TestResolve_UnknownInterface(t *testing.T) {
	r := &Resolver{}
	directive, err := r.Resolve(Declaration{
		Repo:      "github.com/some-dev/noncanonical",
		Version:   "v1.0.0",
		Interface: "SomeUnknownInterface",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if directive != nil {
		t.Errorf("expected nil directive for unknown interface, got %+v", directive)
	}
}

func TestTypeSpecString(t *testing.T) {
	tests := []struct {
		ts   TypeSpec
		want string
	}{
		{TypeSpec{Kind: "ident", Name: "string"}, "string"},
		{TypeSpec{Kind: "selector", Name: "MediaObject"}, "MediaObject"},
		{TypeSpec{Kind: "star", Elem: &TypeSpec{Kind: "ident", Name: "int"}}, "*int"},
		{TypeSpec{Kind: "slice", Elem: &TypeSpec{Kind: "ident", Name: "string"}}, "[]string"},
		{TypeSpec{Kind: "interface"}, "interface{}"},
	}

	for _, tt := range tests {
		got := tt.ts.String()
		if got != tt.want {
			t.Errorf("String() = %q, want %q", got, tt.want)
		}
	}
}

func TestApplyReplaceDirectives_NoGoMod(t *testing.T) {
	dir := t.TempDir()
	err := ApplyReplaceDirectives(dir, []ReplaceDirective{
		{OldPath: "github.com/foo", NewPath: "github.com/bar", Version: "v1.0.0"},
	})
	if err == nil {
		t.Error("expected error for missing go.mod")
	}
}

func TestMethodSig(t *testing.T) {
	m := MethodSpec{
		Name:    "Get",
		Params:  []TypeSpec{{Kind: "ident", Name: "context"}, {Kind: "ident", Name: "string"}},
		Results: []TypeSpec{{Kind: "selector", Name: "MediaObject"}, {Kind: "ident", Name: "error"}},
	}
	got := methodSig(m)
	want := "Get(context, string) (MediaObject, error)"
	if got != want {
		t.Errorf("methodSig = %q, want %q", got, want)
	}
}

func TestRegisteredCanonicals(t *testing.T) {
	reg := RegisteredCanonicals()
	if len(reg) == 0 {
		t.Error("canonical registry is empty")
	}
	// Should have at least the 17 contract repos worth of interfaces
	if len(reg) < 15 {
		t.Errorf("expected at least 15 canonical entries, got %d", len(reg))
	}
	t.Logf("Registered %d canonical contracts", len(reg))
}
