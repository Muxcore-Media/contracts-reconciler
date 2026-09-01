package downloaderv1

// Minimal proto-shaped snapshot for reconciler tests (no gRPC deps).

type AddTorrentRequest struct{}
type AddTorrentResponse struct{}
type RemoveTorrentRequest struct{}
type RemoveTorrentResponse struct{}

// DownloaderServiceServer is the server API for DownloaderService.
type DownloaderServiceServer interface {
	AddTorrent(*AddTorrentRequest) (*AddTorrentResponse, error)
	RemoveTorrent(*RemoveTorrentRequest) (*RemoveTorrentResponse, error)
}
