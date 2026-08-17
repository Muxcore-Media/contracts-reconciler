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

type DownloaderService interface {
	Add(id string) error
	Get(id string) (string, error)
}
`

const mismatchedDownloaderIface = `package contract

type DownloaderService interface {
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
		Interface: "DownloaderService",
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
		Interface: "DownloaderService",
	})
	if err == nil {
		t.Fatal("expected interface mismatch error")
	}
	if !strings.Contains(err.Error(), "interface mismatch") {
		t.Fatalf("expected mismatch error, got: %v", err)
	}
}

func TestParseDir_RecursiveFixture(t *testing.T) {
	root := t.TempDir()
	nested := filepath.Join(root, "muxcore", "downloader", "v1")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	src := `package downloaderv1

type DownloaderService interface {
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
	if _, err := FindInterface(specs, "DownloaderService"); err != nil {
		t.Fatalf("FindInterface: %v (specs=%+v)", err, specs)
	}
}

func TestParseDir_SiblingContractsMediaAdmin(t *testing.T) {
	// Prefer workspace sibling checkout when present (laptop / self-hosted).
	candidates := []string{
		filepath.Join("..", "..", "contracts-media-admin"),
		"/home/user/Projects/MuxCore/contracts-media-admin",
		"/opt/repos/contracts-media-admin",
	}
	var repoDir string
	for _, c := range candidates {
		if st, err := os.Stat(c); err == nil && st.IsDir() {
			repoDir = c
			break
		}
	}
	if repoDir == "" {
		t.Skip("contracts-media-admin not available — skipping sibling parse test")
	}

	specs, err := ParseDir(repoDir)
	if err != nil {
		t.Fatalf("ParseDir(%s): %v", repoDir, err)
	}
	if _, err := FindInterface(specs, "MediaAdminServiceServer"); err != nil {
		t.Fatalf("MediaAdminServiceServer not found: %v", err)
	}
}
