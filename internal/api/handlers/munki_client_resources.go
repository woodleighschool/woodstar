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
	clientResourcesPath              = "/api/munki/client-resources"
	clientResourcesArchivePath       = clientResourcesPath + "/archive"
	clientResourcesArchiveUploadPath = clientResourcesPath + "/archive-uploads"
	clientResourcesBannerUploadPath  = clientResourcesPath + "/banner-uploads"
	clientResourcesLabel             = "Munki client resources"
)

type clientResourcesUploadInput struct {
	Body MunkiUploadRequest
}

type clientResourcesBuilderPutInput struct {
	Body clientresources.Builder
}

type clientResourcesArchivePutInput struct {
	Body struct {
		ObjectID int64 `json:"object_id" minimum:"1"`
	}
}

type clientResourcesOutput struct {
	Body MunkiClientResources
}

type MunkiClientResources struct {
	Archive   MunkiObjectView              `json:"archive"`
	Custom    bool                         `json:"custom"`
	Builder   *MunkiClientResourcesBuilder `json:"builder,omitempty"`
	CreatedAt time.Time                    `json:"created_at"`
	UpdatedAt time.Time                    `json:"updated_at"`
}

type MunkiClientResourcesBuilder struct {
	Banner       MunkiObjectView           `json:"banner"`
	BannerFit    clientresources.BannerFit `json:"banner_fit"`
	BannerFocalX int                       `json:"banner_focal_x" minimum:"0" maximum:"100"`
	Links        []clientresources.Link    `json:"links"`
	FooterText   string                    `json:"footer_text"`
	FooterLinks  []clientresources.Link    `json:"footer_links"`
}

func registerMunkiClientResources(
	api huma.API,
	service *clientresources.Service,
	objects *storage.ObjectStore,
	ingestor *storage.Ingestor,
	logger *slog.Logger,
) {
	registerGetMunkiClientResources(api, service, objects, logger)
	registerUpdateMunkiClientResourcesBuilder(api, service, objects, logger)
	registerPublishMunkiClientResourcesArchive(api, service, objects, logger)
	registerDeleteMunkiClientResources(api, service, logger)
	registerCreateClientResourcesUpload(
		api,
		ingestor,
		logger,
		clientResourcesBannerUploadPath,
		clientresources.BannerObjectPrefix,
		"create-munki-client-resources-banner-upload",
		"Create a banner upload",
	)
	registerDeleteClientResourcesUpload(
		api,
		ingestor,
		logger,
		clientResourcesBannerUploadPath,
		clientresources.BannerObjectPrefix,
		"delete-munki-client-resources-banner-upload",
		"Delete a banner upload",
	)
	registerCreateClientResourcesUpload(
		api,
		ingestor,
		logger,
		clientResourcesArchiveUploadPath,
		clientresources.ArchiveObjectPrefix,
		"create-munki-client-resources-archive-upload",
		"Create a client resources archive upload",
	)
	registerDeleteClientResourcesUpload(
		api,
		ingestor,
		logger,
		clientResourcesArchiveUploadPath,
		clientresources.ArchiveObjectPrefix,
		"delete-munki-client-resources-archive-upload",
		"Delete a client resources archive upload",
	)
}

type clientResourcesUploadIDInput struct {
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

func registerUpdateMunkiClientResourcesBuilder(
	api huma.API,
	service *clientresources.Service,
	objects *storage.ObjectStore,
	logger *slog.Logger,
) {
	huma.Register(api, huma.Operation{
		OperationID: "update-munki-client-resources-builder",
		Method:      http.MethodPut,
		Path:        clientResourcesPath,
		Tags:        []string{munkiClientResourcesTag},
		Summary:     "Update client resources from the builder",
		Errors:      []int{http.StatusBadRequest, http.StatusNotFound},
	}, func(ctx context.Context, input *clientResourcesBuilderPutInput) (*clientResourcesOutput, error) {
		resource, err := service.SaveBuilder(ctx, input.Body)
		if err != nil {
			return nil, resourceError(
				ctx,
				logger,
				"update-munki-client-resources-builder",
				clientResourcesLabel,
				err,
			)
		}
		output, err := clientResourcesResponse(ctx, objects, *resource)
		if err != nil {
			return nil, resourceError(
				ctx,
				logger,
				"update-munki-client-resources-builder",
				clientResourcesLabel,
				err,
			)
		}
		return output, nil
	})
}

func registerPublishMunkiClientResourcesArchive(
	api huma.API,
	service *clientresources.Service,
	objects *storage.ObjectStore,
	logger *slog.Logger,
) {
	huma.Register(api, huma.Operation{
		OperationID: "publish-munki-client-resources-archive",
		Method:      http.MethodPut,
		Path:        clientResourcesArchivePath,
		Tags:        []string{munkiClientResourcesTag},
		Summary:     "Publish a client resources archive",
		Errors:      []int{http.StatusBadRequest, http.StatusNotFound},
	}, func(ctx context.Context, input *clientResourcesArchivePutInput) (*clientResourcesOutput, error) {
		resource, err := service.PublishArchive(ctx, input.Body.ObjectID)
		if err != nil {
			return nil, resourceError(
				ctx,
				logger,
				"publish-munki-client-resources-archive",
				clientResourcesLabel,
				err,
			)
		}
		output, err := clientResourcesResponse(ctx, objects, *resource)
		if err != nil {
			return nil, resourceError(
				ctx,
				logger,
				"publish-munki-client-resources-archive",
				clientResourcesLabel,
				err,
			)
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
		Summary:       "Undeploy client resources",
		DefaultStatus: http.StatusNoContent,
		Errors:        []int{http.StatusNotFound},
	}, func(ctx context.Context, _ *struct{}) (*struct{}, error) {
		if err := service.Delete(ctx); err != nil {
			return nil, resourceError(ctx, logger, "delete-munki-client-resources", clientResourcesLabel, err)
		}
		return &struct{}{}, nil
	})
}

func registerCreateClientResourcesUpload(
	api huma.API,
	ingestor *storage.Ingestor,
	logger *slog.Logger,
	path string,
	prefix string,
	operationID string,
	summary string,
) {
	huma.Register(api, huma.Operation{
		OperationID:   operationID,
		Method:        http.MethodPost,
		Path:          path,
		Tags:          []string{munkiClientResourcesTag},
		Summary:       summary,
		DefaultStatus: http.StatusCreated,
		Errors:        []int{http.StatusBadRequest},
	}, func(ctx context.Context, input *clientResourcesUploadInput) (*munkiDirectUploadOutput, error) {
		object, target, err := ingestor.BeginDirect(ctx, prefix, input.Body.Filename)
		if err != nil {
			return nil, resourceError(ctx, logger, operationID, clientResourcesLabel, err)
		}
		return newMunkiDirectUploadOutput(object, target), nil
	})
}

func registerDeleteClientResourcesUpload(
	api huma.API,
	ingestor *storage.Ingestor,
	logger *slog.Logger,
	path string,
	prefix string,
	operationID string,
	summary string,
) {
	huma.Register(api, huma.Operation{
		OperationID:   operationID,
		Method:        http.MethodDelete,
		Path:          path + "/{id}",
		Tags:          []string{munkiClientResourcesTag},
		Summary:       summary,
		DefaultStatus: http.StatusNoContent,
		Errors:        []int{http.StatusBadRequest, http.StatusNotFound, http.StatusConflict},
	}, func(ctx context.Context, input *clientResourcesUploadIDInput) (*struct{}, error) {
		if err := ingestor.Delete(ctx, input.ID, prefix); err != nil {
			return nil, resourceError(
				ctx,
				logger,
				operationID,
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
	archiveObject, err := objects.GetByID(ctx, resource.ArchiveObjectID)
	if err != nil {
		return nil, err
	}
	if archiveObject.Prefix != clientresources.ArchiveObjectPrefix || !archiveObject.Available() {
		return nil, errors.New("configured client resources reference an invalid archive object")
	}
	response := MunkiClientResources{
		Archive: munkiObjectView(
			*archiveObject,
			contentURL(clientResourcesArchiveUploadPath, archiveObject.ID),
		),
		Custom:    resource.Custom,
		CreatedAt: resource.CreatedAt,
		UpdatedAt: resource.UpdatedAt,
	}
	if resource.Builder == nil {
		return &clientResourcesOutput{Body: response}, nil
	}

	bannerObject, err := objects.GetByID(ctx, resource.Builder.BannerObjectID)
	if err != nil {
		return nil, err
	}
	if bannerObject.Prefix != clientresources.BannerObjectPrefix || !bannerObject.Available() {
		return nil, errors.New("configured client resources reference an invalid banner object")
	}
	response.Builder = &MunkiClientResourcesBuilder{
		Banner: munkiObjectView(
			*bannerObject,
			contentURL(clientResourcesBannerUploadPath, bannerObject.ID),
		),
		BannerFit:    resource.Builder.BannerFit,
		BannerFocalX: resource.Builder.BannerFocalX,
		Links:        resource.Builder.Links,
		FooterText:   resource.Builder.FooterText,
		FooterLinks:  resource.Builder.FooterLinks,
	}
	return &clientResourcesOutput{Body: response}, nil
}
