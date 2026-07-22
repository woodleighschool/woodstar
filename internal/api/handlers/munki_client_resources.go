package handlers

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/munki/clientresources"
	"github.com/woodleighschool/woodstar/internal/storage"
)

const (
	clientResourcesPath              = "/api/munki/client-resources"
	clientResourcesArchiveUploadPath = clientResourcesPath + "/archive-uploads"
	clientResourcesBannerUploadPath  = clientResourcesPath + "/banner-uploads"
	clientResourcesIDPath            = clientResourcesPath + "/{id}"
	clientResourcesLabel             = "Munki client resources"
)

type clientResourcesUploadInput struct {
	Body MunkiUploadRequest
}

type clientResourcesCreateInput struct {
	Body clientresources.ClientResourcesMutation
}

type clientResourcesListInput struct {
	Page    int32  `query:"page,omitempty"     minimum:"1"`
	PerPage int32  `query:"per_page,omitempty" minimum:"1" maximum:"1000"`
	Sort    string `query:"sort,omitempty"`
}

type clientResourcesGetInput struct {
	ID int64 `path:"id"`
}

type clientResourcesUpdateInput struct {
	ID   int64 `path:"id"`
	Body clientresources.ClientResourcesMutation
}

type clientResourcesDeleteInput struct {
	ID int64 `path:"id"`
}

type clientResourcesListOutput struct {
	Body Page[MunkiClientResources]
}

type clientResourcesOutput struct {
	Body MunkiClientResources
}

type MunkiClientResources struct {
	ID        int64                        `json:"id"`
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

func (input clientResourcesListInput) params() dbutil.ListParams {
	return ListQueryInput{
		Page:    input.Page,
		PerPage: input.PerPage,
		Sort:    input.Sort,
	}.params()
}

func registerMunkiClientResources(
	api huma.API,
	service *clientresources.Service,
	objects *storage.ObjectStore,
	ingestor *storage.Ingestor,
	logger *slog.Logger,
) {
	registerListMunkiClientResources(api, service, objects, logger)
	registerCreateMunkiClientResources(api, service, objects, logger)
	registerGetMunkiClientResources(api, service, objects, logger)
	registerUpdateMunkiClientResources(api, service, objects, logger)
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

func registerListMunkiClientResources(
	api huma.API,
	service *clientresources.Service,
	objects *storage.ObjectStore,
	logger *slog.Logger,
) {
	huma.Register(api, huma.Operation{
		OperationID: "list-munki-client-resources",
		Method:      http.MethodGet,
		Path:        clientResourcesPath,
		Tags:        []string{munkiClientResourcesTag},
		Summary:     "List client resources",
	}, func(ctx context.Context, input *clientResourcesListInput) (*clientResourcesListOutput, error) {
		resources, count, err := service.List(ctx, input.params())
		if err != nil {
			return nil, handlerError(ctx, logger, "list-munki-client-resources", err)
		}
		items := make([]MunkiClientResources, len(resources))
		for i, resource := range resources {
			output, err := clientResourcesResponse(ctx, objects, resource)
			if err != nil {
				return nil, handlerError(ctx, logger, "list-munki-client-resources", err)
			}
			items[i] = output.Body
		}
		return &clientResourcesListOutput{
			Body: Page[MunkiClientResources]{Items: items, Count: count},
		}, nil
	})
}

func registerCreateMunkiClientResources(
	api huma.API,
	service *clientresources.Service,
	objects *storage.ObjectStore,
	logger *slog.Logger,
) {
	huma.Register(api, huma.Operation{
		OperationID:   "create-munki-client-resources",
		Method:        http.MethodPost,
		Path:          clientResourcesPath,
		Tags:          []string{munkiClientResourcesTag},
		Summary:       "Create client resources",
		DefaultStatus: http.StatusCreated,
		Errors:        []int{http.StatusBadRequest, http.StatusNotFound, http.StatusConflict},
	}, func(ctx context.Context, input *clientResourcesCreateInput) (*clientResourcesOutput, error) {
		resource, err := service.Create(ctx, input.Body)
		if err != nil {
			return nil, resourceError(ctx, logger, "create-munki-client-resources", clientResourcesLabel, err)
		}
		output, err := clientResourcesResponse(ctx, objects, *resource)
		if err != nil {
			return nil, handlerError(ctx, logger, "create-munki-client-resources", err)
		}
		return output, nil
	})
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
		Path:        clientResourcesIDPath,
		Tags:        []string{munkiClientResourcesTag},
		Summary:     "Get client resources",
		Errors:      []int{http.StatusNotFound},
	}, func(ctx context.Context, input *clientResourcesGetInput) (*clientResourcesOutput, error) {
		resource, err := service.GetByID(ctx, input.ID)
		if err != nil {
			return nil, resourceError(ctx, logger, "get-munki-client-resources", clientResourcesLabel, err, "id", input.ID)
		}
		output, err := clientResourcesResponse(ctx, objects, *resource)
		if err != nil {
			return nil, handlerError(ctx, logger, "get-munki-client-resources", err, "id", input.ID)
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
		Path:        clientResourcesIDPath,
		Tags:        []string{munkiClientResourcesTag},
		Summary:     "Update client resources",
		Errors:      []int{http.StatusBadRequest, http.StatusNotFound, http.StatusConflict},
	}, func(ctx context.Context, input *clientResourcesUpdateInput) (*clientResourcesOutput, error) {
		resource, err := service.Update(ctx, input.ID, input.Body)
		if err != nil {
			return nil, resourceError(ctx, logger, "update-munki-client-resources", clientResourcesLabel, err, "id", input.ID)
		}
		output, err := clientResourcesResponse(ctx, objects, *resource)
		if err != nil {
			return nil, handlerError(ctx, logger, "update-munki-client-resources", err, "id", input.ID)
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
		Path:          clientResourcesIDPath,
		Tags:          []string{munkiClientResourcesTag},
		Summary:       "Delete client resources",
		DefaultStatus: http.StatusNoContent,
		Errors:        []int{http.StatusNotFound},
	}, func(ctx context.Context, input *clientResourcesDeleteInput) (*struct{}, error) {
		if err := service.Delete(ctx, input.ID); err != nil {
			return nil, resourceError(ctx, logger, "delete-munki-client-resources", clientResourcesLabel, err, "id", input.ID)
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
		ID: resource.ID,
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
