package reconciler

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// seedFixtureRepo writes a minimal Go package with one exported interface under
// the reconciler cache key layout so Resolve can run offline (no git clone).
func seedFixtureRepo(t *testing.T, cacheDir, importPath, version, ifaceSrc string) {
	t.Helper()
	safePath := strings.NewReplacer("/", "-", ".", "-").Replace(importPath)
	dir := filepath.Join(cacheDir, "contracts-reconciler-cache", safePath+"-"+version)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "contract.go"), []byte(ifaceSrc), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
}

const matchingDownloaderIface = `package contract

type DownloaderServiceServer interface {
	Add(id string) error
	Get(id string) (string, error)
}
`

const mismatchedDownloaderIface = `package contract

type DownloaderServiceServer interface {
	Add(id string) error
	Get(id string) (int, error)
}
`

func TestResolve_FixtureMatchReplace(t *testing.T) {
	cache := t.TempDir()
	canonPath := "github.com/Muxcore-Media/contracts-downloader"
	thirdPath := "github.com/example/contracts-downloader"
	version := "v0.1.0"

	seedFixtureRepo(t, cache, canonPath, version, matchingDownloaderIface)
	seedFixtureRepo(t, cache, thirdPath, version, matchingDownloaderIface)

	r := &Resolver{CacheDir: cache}
	directive, err := r.Resolve(Declaration{
		Repo:      thirdPath,
		Version:   version,
		Interface: "DownloaderServiceServer",
	})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if directive == nil {
		t.Fatal("expected replace directive for matching third-party contract")
	}
	if directive.OldPath != thirdPath || directive.NewPath != canonPath || directive.Version != version {
		t.Fatalf("unexpected directive: %+v", directive)
	}
}

func TestResolve_FixtureMismatch(t *testing.T) {
	cache := t.TempDir()
	canonPath := "github.com/Muxcore-Media/contracts-downloader"
	thirdPath := "github.com/example/contracts-downloader"
	version := "v0.1.0"

	seedFixtureRepo(t, cache, canonPath, version, matchingDownloaderIface)
	seedFixtureRepo(t, cache, thirdPath, version, mismatchedDownloaderIface)

	r := &Resolver{CacheDir: cache}
	_, err := r.Resolve(Declaration{
		Repo:      thirdPath,
		Version:   version,
		Interface: "DownloaderServiceServer",
	})
	if err == nil {
		t.Fatal("expected interface mismatch error")
	}
	if !strings.Contains(err.Error(), "interface mismatch") {
		t.Fatalf("expected mismatch error, got: %v", err)
	}
}

func TestResolve_ReservedPlayback(t *testing.T) {
	r := &Resolver{}
	directive, err := r.Resolve(Declaration{
		Repo:      "github.com/example/contracts-playback",
		Version:   "v1.0.0",
		Interface: "Playback",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if directive != nil {
		t.Fatalf("expected nil for reserved Playback, got %+v", directive)
	}
}

func TestResolve_VersionPinUsesDeclarationNotRegistry(t *testing.T) {
	cache := t.TempDir()
	canonPath := "github.com/Muxcore-Media/contracts-downloader"
	thirdPath := "github.com/example/contracts-downloader"
	declVersion := "v1.2.0"

	seedFixtureRepo(t, cache, canonPath, declVersion, matchingDownloaderIface)
	seedFixtureRepo(t, cache, thirdPath, declVersion, matchingDownloaderIface)

	r := &Resolver{CacheDir: cache}
	directive, err := r.Resolve(Declaration{
		Repo:      thirdPath,
		Version:   declVersion,
		Interface: "DownloaderServiceServer",
	})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if directive == nil || directive.Version != declVersion {
		t.Fatalf("expected pin at %s, got %+v", declVersion, directive)
	}
}

func TestParseDir_RecursiveFixture(t *testing.T) {
	root := t.TempDir()
	nested := filepath.Join(root, "muxcore", "downloader", "v1")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	src := `package downloaderv1

type DownloaderServiceServer interface {
	Add(id string) error
}
`
	if err := os.WriteFile(filepath.Join(nested, "svc.go"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}

	specs, err := ParseDir(root)
	if err != nil {
		t.Fatalf("ParseDir: %v", err)
	}
	if _, err := FindInterface(specs, "DownloaderServiceServer"); err != nil {
		t.Fatalf("FindInterface: %v (specs=%+v)", err, specs)
	}
}

func TestParseDir_TestdataMediaAdmin(t *testing.T) {
	repoDir := filepath.Join("testdata", "media-admin")
	specs, err := ParseDir(repoDir)
	if err != nil {
		t.Fatalf("ParseDir(%s): %v", repoDir, err)
	}
	spec, err := FindInterface(specs, "MediaAdminServiceServer")
	if err != nil {
		t.Fatalf("MediaAdminServiceServer not found: %v", err)
	}
	if len(spec.Methods) < 2 {
		t.Errorf("expected at least 2 methods, got %d", len(spec.Methods))
	}
}

func TestParseDir_TestdataDownloader(t *testing.T) {
	repoDir := filepath.Join("testdata", "downloader")
	specs, err := ParseDir(repoDir)
	if err != nil {
		t.Fatalf("ParseDir(%s): %v", repoDir, err)
	}
	spec, err := FindInterface(specs, "DownloaderServiceServer")
	if err != nil {
		t.Fatalf("DownloaderServiceServer not found: %v", err)
	}
	if len(spec.Methods) < 2 {
		t.Errorf("expected at least 2 methods, got %d", len(spec.Methods))
	}
}

func TestDryRun_WithCache(t *testing.T) {
	cache := t.TempDir()
	canonPath := "github.com/Muxcore-Media/contracts-downloader"
	thirdPath := "github.com/example/contracts-downloader"
	version := "v0.1.0"

	seedFixtureRepo(t, cache, canonPath, version, matchingDownloaderIface)
	seedFixtureRepo(t, cache, thirdPath, version, matchingDownloaderIface)

	report, err := DryRun(&Resolver{CacheDir: cache}, []Declaration{
		{Repo: thirdPath, Version: version, Interface: "DownloaderServiceServer"},
	})
	if err != nil {
		t.Fatalf("DryRun: %v", err)
	}
	if !strings.Contains(report, thirdPath) {
		t.Errorf("expected replace report for third party, got: %s", report)
	}
}
