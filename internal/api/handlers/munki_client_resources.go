package handlers

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/munki/clientresources"
	munkiupload "github.com/woodleighschool/woodstar/internal/munki/objectupload"
	"github.com/woodleighschool/woodstar/internal/storage"
)

const (
	clientResourcesPath  = "/api/munki/client-resources"
	clientResourcesLabel = "Munki client resources"
)

type ClientResourcesBannerUploadRequest struct {
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
	SizeBytes   int64  `json:"size_bytes"   minimum:"1" maximum:"5242880"`
}

type clientResourcesUploadInput struct {
	Body ClientResourcesBannerUploadRequest
}

type clientResourcesPutInput struct {
	Body clientresources.Mutation
}

type clientResourcesOutput struct {
	Body MunkiClientResources
}

type MunkiClientResources struct {
	Banner          MunkiObjectView                 `json:"banner"`
	BannerAlignment clientresources.BannerAlignment `json:"banner_alignment"`
	Links           []clientresources.Link          `json:"links"`
	FooterText      string                          `json:"footer_text"`
	FooterLinks     []clientresources.Link          `json:"footer_links"`
	CreatedAt       time.Time                       `json:"created_at"`
	UpdatedAt       time.Time                       `json:"updated_at"`
}

func registerMunkiClientResources(
	api huma.API,
	service *clientresources.Service,
	objects *storage.ObjectStore,
	presigner storage.Presigner,
	logger *slog.Logger,
) {
	registerGetMunkiClientResources(api, service, objects, presigner, logger)
	registerSaveMunkiClientResources(api, service, objects, presigner, logger)
	registerDeleteMunkiClientResources(api, service, logger)
	registerCreateClientResourcesBannerUpload(api, objects, presigner, logger)
}

func registerGetMunkiClientResources(
	api huma.API,
	service *clientresources.Service,
	objects *storage.ObjectStore,
	presigner storage.Presigner,
	logger *slog.Logger,
) {
	huma.Register(api, huma.Operation{
		OperationID: "get-munki-client-resources",
		Method:      http.MethodGet,
		Path:        clientResourcesPath,
		Tags:        []string{munkiTag},
		Summary:     "Get configured Munki client resources",
		Errors:      []int{http.StatusNotFound},
	}, func(ctx context.Context, _ *struct{}) (*clientResourcesOutput, error) {
		resource, err := service.Get(ctx)
		if err != nil {
			return nil, resourceError(ctx, logger, "get-munki-client-resources", clientResourcesLabel, err)
		}
		output, err := clientResourcesResponse(ctx, objects, presigner, *resource)
		if err != nil {
			return nil, resourceError(ctx, logger, "get-munki-client-resources", clientResourcesLabel, err)
		}
		return output, nil
	})
}

func registerSaveMunkiClientResources(
	api huma.API,
	service *clientresources.Service,
	objects *storage.ObjectStore,
	presigner storage.Presigner,
	logger *slog.Logger,
) {
	huma.Register(api, huma.Operation{
		OperationID: "save-munki-client-resources",
		Method:      http.MethodPut,
		Path:        clientResourcesPath,
		Tags:        []string{munkiTag},
		Summary:     "Build and publish Munki client resources",
		Errors: []int{
			http.StatusBadRequest,
			http.StatusNotFound,
		},
	}, func(ctx context.Context, input *clientResourcesPutInput) (*clientResourcesOutput, error) {
		resource, err := service.Save(ctx, input.Body)
		if err != nil {
			return nil, resourceError(ctx, logger, "save-munki-client-resources", clientResourcesLabel, err)
		}
		output, err := clientResourcesResponse(ctx, objects, presigner, *resource)
		if err != nil {
			return nil, resourceError(ctx, logger, "save-munki-client-resources", clientResourcesLabel, err)
		}
		return output, nil
	})
}

func registerDeleteMunkiClientResources(
	api huma.API,
	service *clientresources.Service,
	logger *slog.Logger,
) {
	huma.Register(api, huma.Operation{
		OperationID:   "delete-munki-client-resources",
		Method:        http.MethodDelete,
		Path:          clientResourcesPath,
		Tags:          []string{munkiTag},
		Summary:       "Remove Munki client resources and use Munki defaults",
		DefaultStatus: http.StatusNoContent,
		Errors:        []int{http.StatusNotFound},
	}, func(ctx context.Context, _ *struct{}) (*struct{}, error) {
		if err := service.Delete(ctx); err != nil {
			return nil, resourceError(ctx, logger, "delete-munki-client-resources", clientResourcesLabel, err)
		}
		return &struct{}{}, nil
	})
}

func registerCreateClientResourcesBannerUpload(
	api huma.API,
	objects *storage.ObjectStore,
	presigner storage.Presigner,
	logger *slog.Logger,
) {
	huma.Register(api, huma.Operation{
		OperationID:   "create-munki-client-resources-banner-upload",
		Method:        http.MethodPost,
		Path:          clientResourcesPath + "/banner",
		Tags:          []string{munkiTag},
		Summary:       "Create a banner upload for Munki client resources",
		DefaultStatus: http.StatusCreated,
		Errors:        []int{http.StatusBadRequest},
	}, func(ctx context.Context, input *clientResourcesUploadInput) (*munkiUploadOutput, error) {
		if err := clientresources.ValidateBannerUpload(input.Body.ContentType, input.Body.SizeBytes); err != nil {
			return nil, resourceError(
				ctx,
				logger,
				"create-munki-client-resources-banner-upload",
				clientResourcesLabel,
				err,
			)
		}
		object, target, err := munkiupload.Create(
			ctx,
			objects,
			presigner,
			clientresources.BannerObjectPrefix,
			input.Body.Filename,
			input.Body.ContentType,
		)
		if err != nil {
			return nil, resourceError(
				ctx,
				logger,
				"create-munki-client-resources-banner-upload",
				clientResourcesLabel,
				err,
			)
		}
		return munkiUploadOutputFromTarget(object, target), nil
	})
}

func clientResourcesResponse(
	ctx context.Context,
	objects *storage.ObjectStore,
	presigner storage.Presigner,
	resource clientresources.ClientResources,
) (*clientResourcesOutput, error) {
	bannerObject, err := objects.GetByID(ctx, resource.BannerObjectID)
	if err != nil {
		return nil, err
	}
	if bannerObject.Prefix != clientresources.BannerObjectPrefix || !bannerObject.Available() {
		return nil, errors.New("configured client resources reference an invalid banner object")
	}
	banner, err := munkiObjectViewWithContentURL(ctx, presigner, *bannerObject)
	if err != nil {
		return nil, err
	}
	return &clientResourcesOutput{Body: MunkiClientResources{
		Banner:          banner,
		BannerAlignment: resource.BannerAlignment,
		Links:           resource.Links,
		FooterText:      resource.FooterText,
		FooterLinks:     resource.FooterLinks,
		CreatedAt:       resource.CreatedAt,
		UpdatedAt:       resource.UpdatedAt,
	}}, nil
}
