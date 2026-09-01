package mediaadminv1

// Minimal proto-shaped snapshot for reconciler tests (no gRPC deps).

type GetMediaTypeInfoRequest struct{}
type GetMediaTypeInfoResponse struct{}
type ListItemsRequest struct{}
type ListItemsResponse struct{}

// MediaAdminServiceServer is the server API for MediaAdminService.
type MediaAdminServiceServer interface {
	GetMediaTypeInfo(*GetMediaTypeInfoRequest) (*GetMediaTypeInfoResponse, error)
	ListItems(*ListItemsRequest) (*ListItemsResponse, error)
}
