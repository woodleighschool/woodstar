package santa

import (
	"context"
	"errors"

	syncv1 "buf.build/gen/go/northpolesec/protos/protocolbuffers/go/sync"
)

// ErrNotImplemented indicates a sync stage route exists before its service
// behavior has landed.
var ErrNotImplemented = errors.New("santa sync stage not implemented")

// Service coordinates Santa sync protocol stages.
type Service struct {
	store *Store
}

func NewService(store *Store) *Service {
	return &Service{store: store}
}

func (s *Service) HandlePreflight(
	context.Context,
	string,
	*syncv1.PreflightRequest,
) (*syncv1.PreflightResponse, error) {
	return nil, ErrNotImplemented
}

func (s *Service) HandleEventUpload(
	context.Context,
	string,
	*syncv1.EventUploadRequest,
) (*syncv1.EventUploadResponse, error) {
	return nil, ErrNotImplemented
}

func (s *Service) HandleRuleDownload(
	context.Context,
	string,
	*syncv1.RuleDownloadRequest,
) (*syncv1.RuleDownloadResponse, error) {
	return nil, ErrNotImplemented
}

func (s *Service) HandlePostflight(
	context.Context,
	string,
	*syncv1.PostflightRequest,
) (*syncv1.PostflightResponse, error) {
	return nil, ErrNotImplemented
}
