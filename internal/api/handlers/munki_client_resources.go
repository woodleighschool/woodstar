package handlers

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/munki/clientresources"
	"github.com/woodleighschool/woodstar/internal/storage"
)

const (
	clientResourcesPath       = "/api/munki/client-resources"
	clientResourcesBannerPath = clientResourcesPath + "/banner-uploads"
	clientResourcesLabel      = "Munki client resources"
)

type clientResourcesUploadInput struct {
	Body MunkiUploadRequest
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
	ingestor *storage.Ingestor,
	logger *slog.Logger,
) {
	registerGetMunkiClientResources(api, service, objects, logger)
	registerUpdateMunkiClientResources(api, service, objects, logger)
	registerDeleteMunkiClientResources(api, service, logger)
	registerCreateClientResourcesBannerUpload(api, ingestor, logger)
	registerDeleteClientResourcesBannerUpload(api, ingestor, logger)
}

type clientResourcesBannerUploadInput struct {
	ID int64 `path:"id"`
}

func registerGetMunkiClientResources(
	api huma.API,
	service *clientresources.Service,
	objects *storage.ObjectStore,
	logger *slog.Logger,
) {
	huma.Register(api, huma.Operation{
		OperationID: "get-munki-client-resources",
		Method:      http.MethodGet,
		Path:        clientResourcesPath,
		Tags:        []string{munkiClientResourcesTag},
		Summary:     "Get client resources",
		Errors:      []int{http.StatusNotFound},
	}, func(ctx context.Context, _ *struct{}) (*clientResourcesOutput, error) {
		resource, err := service.Get(ctx)
		if err != nil {
			return nil, resourceError(ctx, logger, "get-munki-client-resources", clientResourcesLabel, err)
		}
		output, err := clientResourcesResponse(ctx, objects, *resource)
		if err != nil {
			return nil, resourceError(ctx, logger, "get-munki-client-resources", clientResourcesLabel, err)
		}
		return output, nil
	})
}

func registerUpdateMunkiClientResources(
	api huma.API,
	service *clientresources.Service,
	objects *storage.ObjectStore,
	logger *slog.Logger,
) {
	huma.Register(api, huma.Operation{
		OperationID: "update-munki-client-resources",
		Method:      http.MethodPut,
		Path:        clientResourcesPath,
		Tags:        []string{munkiClientResourcesTag},
		Summary:     "Update client resources",
		Errors: []int{
			http.StatusBadRequest,
			http.StatusNotFound,
		},
	}, func(ctx context.Context, input *clientResourcesPutInput) (*clientResourcesOutput, error) {
		resource, err := service.Save(ctx, input.Body)
		if err != nil {
			return nil, resourceError(ctx, logger, "update-munki-client-resources", clientResourcesLabel, err)
		}
		output, err := clientResourcesResponse(ctx, objects, *resource)
		if err != nil {
			return nil, resourceError(ctx, logger, "update-munki-client-resources", clientResourcesLabel, err)
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
		Tags:          []string{munkiClientResourcesTag},
		Summary:       "Delete client resources",
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
	ingestor *storage.Ingestor,
	logger *slog.Logger,
) {
	huma.Register(api, huma.Operation{
		OperationID:   "create-munki-client-resources-banner-upload",
		Method:        http.MethodPost,
		Path:          clientResourcesBannerPath,
		Tags:          []string{munkiClientResourcesTag},
		Summary:       "Create a banner upload",
		DefaultStatus: http.StatusCreated,
		Errors:        []int{http.StatusBadRequest},
	}, func(ctx context.Context, input *clientResourcesUploadInput) (*munkiDirectUploadOutput, error) {
		object, target, err := ingestor.BeginDirect(
			ctx,
			clientresources.BannerObjectPrefix,
			input.Body.Filename,
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
		return newMunkiDirectUploadOutput(object, target), nil
	})
}

func registerDeleteClientResourcesBannerUpload(
	api huma.API,
	ingestor *storage.Ingestor,
	logger *slog.Logger,
) {
	huma.Register(api, huma.Operation{
		OperationID:   "delete-munki-client-resources-banner-upload",
		Method:        http.MethodDelete,
		Path:          clientResourcesBannerPath + "/{id}",
		Tags:          []string{munkiClientResourcesTag},
		Summary:       "Delete a banner upload",
		DefaultStatus: http.StatusNoContent,
		Errors:        []int{http.StatusBadRequest, http.StatusNotFound, http.StatusConflict},
	}, func(ctx context.Context, input *clientResourcesBannerUploadInput) (*struct{}, error) {
		if err := ingestor.Delete(ctx, input.ID, clientresources.BannerObjectPrefix); err != nil {
			return nil, resourceError(
				ctx,
				logger,
				"delete-munki-client-resources-banner-upload",
				clientResourcesLabel,
				err,
				"object_id",
				input.ID,
			)
		}
		return &struct{}{}, nil
	})
}

func clientResourcesResponse(
	ctx context.Context,
	objects *storage.ObjectStore,
	resource clientresources.ClientResources,
) (*clientResourcesOutput, error) {
	bannerObject, err := objects.GetByID(ctx, resource.BannerObjectID)
	if err != nil {
		return nil, err
	}
	if bannerObject.Prefix != clientresources.BannerObjectPrefix || !bannerObject.Available() {
		return nil, errors.New("configured client resources reference an invalid banner object")
	}
	banner := munkiObjectView(*bannerObject, clientResourcesBannerContentURL(bannerObject.ID))
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
