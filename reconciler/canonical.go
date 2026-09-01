package reconciler

// canonicalRegistry maps MuxCore interface names to their canonical Go module
// import paths. Published entries are reconcilable; reserved entries are known
// contract surfaces that are not yet published or have no Server interface.
var canonicalRegistry = map[string]CanonicalRepo{
	// contracts-media-admin (published)
	"MediaAdminServiceServer": {
		ImportPath: "github.com/Muxcore-Media/contracts-media-admin",
		Version:    "v0.1.0",
	},

	// contracts-downloader (published)
	"DownloaderServiceServer": {
		ImportPath: "github.com/Muxcore-Media/contracts-downloader",
		Version:    "v0.1.0",
	},

	// contracts-indexer (published)
	"IndexerServiceServer": {
		ImportPath: "github.com/Muxcore-Media/contracts-indexer",
		Version:    "v0.1.0",
	},

	// contracts-notification (published)
	"NotificationServiceServer": {
		ImportPath: "github.com/Muxcore-Media/contracts-notification",
		Version:    "v0.1.0",
	},

	// contracts-metadata (published)
	"MetadataServiceServer": {
		ImportPath: "github.com/Muxcore-Media/contracts-metadata",
		Version:    "v0.1.0",
	},

	// contracts-scanner (published)
	"ScannerServiceServer": {
		ImportPath: "github.com/Muxcore-Media/contracts-scanner",
		Version:    "v0.1.0",
	},

	// contracts-automation (published)
	"AutomationServiceServer": {
		ImportPath: "github.com/Muxcore-Media/contracts-automation",
		Version:    "v0.1.0",
	},

	// Reserved — repos not published or events-only (Resolve returns nil)

	"Playback": {
		ImportPath: "github.com/Muxcore-Media/contracts-playback",
		Version:    "v1.0.0",
		Reserved:   true,
	},

	"Transcoder": {
		ImportPath: "github.com/Muxcore-Media/contracts-transcoder",
		Version:    "v1.0.0",
		Reserved:   true,
	},

	"ArtworkProvider": {
		ImportPath: "github.com/Muxcore-Media/contracts-artwork",
		Version:    "v1.0.0",
		Reserved:   true,
	},

	"SupplementaryContentProvider": {
		ImportPath: "github.com/Muxcore-Media/contracts-content",
		Version:    "v1.0.0",
		Reserved:   true,
	},

	"MediaDiscovery": {
		ImportPath: "github.com/Muxcore-Media/contracts-discovery",
		Version:    "v1.0.0",
		Reserved:   true,
	},

	"ReleaseDecider": {
		ImportPath: "github.com/Muxcore-Media/contracts-quality",
		Version:    "v1.0.0",
		Reserved:   true,
	},
	"FormatMatcher": {
		ImportPath: "github.com/Muxcore-Media/contracts-quality",
		Version:    "v1.0.0",
		Reserved:   true,
	},

	"MediaInfoProvider": {
		ImportPath: "github.com/Muxcore-Media/contracts-mediainfo",
		Version:    "v1.0.0",
		Reserved:   true,
	},

	"MediaResolver": {
		ImportPath: "github.com/Muxcore-Media/contracts-resolver",
		Version:    "v1.0.0",
		Reserved:   true,
	},

	"ImportListProvider": {
		ImportPath: "github.com/Muxcore-Media/contracts-importlist",
		Version:    "v1.0.0",
		Reserved:   true,
	},

	"TagProvider": {
		ImportPath: "github.com/Muxcore-Media/contracts-tag",
		Version:    "v1.0.0",
		Reserved:   true,
	},

	"WorkflowEngine": {
		ImportPath: "github.com/Muxcore-Media/contracts-workflow",
		Version:    "v1.0.0",
		Reserved:   true,
	},

	"FileWatcher": {
		ImportPath: "github.com/Muxcore-Media/contracts-filewatcher",
		Version:    "v1.0.0",
		Reserved:   true,
	},
}

// Canonical returns the canonical repo for a given interface name, or nil
// if the interface is not recognized as a MuxCore contract.
func Canonical(interfaceName string) *CanonicalRepo {
	if cr, ok := canonicalRegistry[interfaceName]; ok {
		return &cr
	}
	return nil
}

// RegisteredCanonicals returns all known canonical contract entries.
func RegisteredCanonicals() map[string]CanonicalRepo {
	out := make(map[string]CanonicalRepo, len(canonicalRegistry))
	for k, v := range canonicalRegistry {
		out[k] = v
	}
	return out
}
