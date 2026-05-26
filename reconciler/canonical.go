package reconciler

// canonicalRegistry maps MuxCore interface names to their canonical Go module
// import paths. This is the authoritative list maintained alongside the
// contracts-* repos in github.com/Muxcore-Media.
//
// When a third-party module declares a contract from a non-canonical repo,
// the reconciler looks up the interface name here to find the canonical
// equivalent for structural comparison.
//
// Interface names must be unique across all contract repos. If two contract
// repos define different interfaces with the same name, one must be renamed.
var canonicalRegistry = map[string]CanonicalRepo{
	// contracts-media
	"MediaLibrary": {
		ImportPath: "github.com/Muxcore-Media/contracts-media",
		Version:    "v1.0.0",
	},

	// contracts-downloader
	"Downloader": {
		ImportPath: "github.com/Muxcore-Media/contracts-downloader",
		Version:    "v1.0.0",
	},
	"DownloadRouter": {
		ImportPath: "github.com/Muxcore-Media/contracts-downloader",
		Version:    "v1.0.0",
	},

	// contracts-indexer
	"Indexer": {
		ImportPath: "github.com/Muxcore-Media/contracts-indexer",
		Version:    "v1.0.0",
	},

	// contracts-playback
	"Playback": {
		ImportPath: "github.com/Muxcore-Media/contracts-playback",
		Version:    "v1.0.0",
	},

	// contracts-transcoder
	"Transcoder": {
		ImportPath: "github.com/Muxcore-Media/contracts-transcoder",
		Version:    "v1.0.0",
	},

	// contracts-metadata
	"MetadataProvider": {
		ImportPath: "github.com/Muxcore-Media/contracts-metadata",
		Version:    "v1.0.0",
	},

	// contracts-artwork
	"ArtworkProvider": {
		ImportPath: "github.com/Muxcore-Media/contracts-artwork",
		Version:    "v1.0.0",
	},

	// contracts-content
	"SupplementaryContentProvider": {
		ImportPath: "github.com/Muxcore-Media/contracts-content",
		Version:    "v1.0.0",
	},

	// contracts-discovery
	"MediaDiscovery": {
		ImportPath: "github.com/Muxcore-Media/contracts-discovery",
		Version:    "v1.0.0",
	},

	// contracts-quality
	"ReleaseDecider": {
		ImportPath: "github.com/Muxcore-Media/contracts-quality",
		Version:    "v1.0.0",
	},
	"FormatMatcher": {
		ImportPath: "github.com/Muxcore-Media/contracts-quality",
		Version:    "v1.0.0",
	},

	// contracts-mediainfo
	"MediaInfoProvider": {
		ImportPath: "github.com/Muxcore-Media/contracts-mediainfo",
		Version:    "v1.0.0",
	},

	// contracts-resolver
	"MediaResolver": {
		ImportPath: "github.com/Muxcore-Media/contracts-resolver",
		Version:    "v1.0.0",
	},

	// contracts-notification
	"NotificationProvider": {
		ImportPath: "github.com/Muxcore-Media/contracts-notification",
		Version:    "v1.0.0",
	},

	// contracts-importlist
	"ImportListProvider": {
		ImportPath: "github.com/Muxcore-Media/contracts-importlist",
		Version:    "v1.0.0",
	},

	// contracts-tag
	"TagProvider": {
		ImportPath: "github.com/Muxcore-Media/contracts-tag",
		Version:    "v1.0.0",
	},

	// contracts-workflow
	"WorkflowEngine": {
		ImportPath: "github.com/Muxcore-Media/contracts-workflow",
		Version:    "v1.0.0",
	},

	// contracts-filewatcher
	"FileWatcher": {
		ImportPath: "github.com/Muxcore-Media/contracts-filewatcher",
		Version:    "v1.0.0",
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
