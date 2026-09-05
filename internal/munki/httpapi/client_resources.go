package httpapi

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/goodies/bloby"
	"github.com/woodleighschool/woodstar/internal/api"
	"github.com/woodleighschool/woodstar/internal/listing"
	"github.com/woodleighschool/woodstar/internal/munki/clientresources"
)

const (
	clientResourcesPath              = "/api/munki/client-resources"
	clientResourcesArchiveUploadPath = clientResourcesPath + "/archive-uploads"
	clientResourcesBannerUploadPath  = clientResourcesPath + "/banner-uploads"
	clientResourcesIDPath            = clientResourcesPath + "/{id}"
	clientResourcesLabel             = "Munki client resources"
)

type clientResourcesUploadInput struct {
	Body MunkiDirectUploadRequest
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
	Body api.Page[MunkiClientResources]
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

func (input clientResourcesListInput) params() listing.Params {
	return api.ListQueryInput{
		Page:    input.Page,
		PerPage: input.PerPage,
		Sort:    input.Sort,
	}.Params()
}

func registerMunkiClientResources(
	humaAPI huma.API,
	service *clientresources.Service,
	objects *bloby.Service,
	logger *slog.Logger,
) {
	registerListMunkiClientResources(humaAPI, service, objects, logger)
	registerCreateMunkiClientResources(humaAPI, service, objects, logger)
	registerGetMunkiClientResources(humaAPI, service, objects, logger)
	registerUpdateMunkiClientResources(humaAPI, service, objects, logger)
	registerDeleteMunkiClientResources(humaAPI, service, logger)
	registerCreateClientResourcesUpload(
		humaAPI,
		objects,
		logger,
		clientResourcesBannerUploadPath,
		clientresources.BannerObjectPrefix,
		"create-munki-client-resources-banner-upload",
		"Create a banner upload",
	)
	registerDeleteClientResourcesUpload(
		humaAPI,
		objects,
		logger,
		clientResourcesBannerUploadPath,
		clientresources.BannerObjectPrefix,
		"delete-munki-client-resources-banner-upload",
		"Delete a banner upload",
	)
	registerCreateClientResourcesUpload(
		humaAPI,
		objects,
		logger,
		clientResourcesArchiveUploadPath,
		clientresources.ArchiveObjectPrefix,
		"create-munki-client-resources-archive-upload",
		"Create a client resources archive upload",
	)
	registerDeleteClientResourcesUpload(
		humaAPI,
		objects,
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
	humaAPI huma.API,
	service *clientresources.Service,
	objects *bloby.Service,
	logger *slog.Logger,
) {
	huma.Register(humaAPI, huma.Operation{
		OperationID: "list-munki-client-resources",
		Method:      http.MethodGet,
		Path:        clientResourcesPath,
		Tags:        []string{api.TagMunkiClientResources},
		Summary:     "List client resources",
	}, func(ctx context.Context, input *clientResourcesListInput) (*clientResourcesListOutput, error) {
		resources, count, err := service.List(ctx, input.params())
		if err != nil {
			return nil, api.HandlerError(ctx, logger, "list-munki-client-resources", err)
		}
		items := make([]MunkiClientResources, len(resources))
		for i, resource := range resources {
			output, err := clientResourcesResponse(ctx, objects, resource)
			if err != nil {
				return nil, api.HandlerError(ctx, logger, "list-munki-client-resources", err)
			}
			items[i] = output.Body
		}
		return &clientResourcesListOutput{
			Body: api.Page[MunkiClientResources]{Items: items, Count: count},
		}, nil
	})
}

func registerCreateMunkiClientResources(
	humaAPI huma.API,
	service *clientresources.Service,
	objects *bloby.Service,
	logger *slog.Logger,
) {
	huma.Register(humaAPI, huma.Operation{
		OperationID:   "create-munki-client-resources",
		Method:        http.MethodPost,
		Path:          clientResourcesPath,
		Tags:          []string{api.TagMunkiClientResources},
		Summary:       "Create client resources",
		DefaultStatus: http.StatusCreated,
		Errors:        []int{http.StatusBadRequest, http.StatusNotFound, http.StatusConflict},
	}, func(ctx context.Context, input *clientResourcesCreateInput) (*clientResourcesOutput, error) {
		resource, err := service.Create(ctx, input.Body)
		if err != nil {
			return nil, api.ResourceError(ctx, logger, "create-munki-client-resources", clientResourcesLabel, err)
		}
		output, err := clientResourcesResponse(ctx, objects, *resource)
		if err != nil {
			return nil, api.HandlerError(ctx, logger, "create-munki-client-resources", err)
		}
		return output, nil
	})
}

func registerGetMunkiClientResources(
	humaAPI huma.API,
	service *clientresources.Service,
	objects *bloby.Service,
	logger *slog.Logger,
) {
	huma.Register(humaAPI, huma.Operation{
		OperationID: "get-munki-client-resources",
		Method:      http.MethodGet,
		Path:        clientResourcesIDPath,
		Tags:        []string{api.TagMunkiClientResources},
		Summary:     "Get client resources",
		Errors:      []int{http.StatusNotFound},
	}, func(ctx context.Context, input *clientResourcesGetInput) (*clientResourcesOutput, error) {
		resource, err := service.GetByID(ctx, input.ID)
		if err != nil {
			return nil, api.ResourceError(ctx, logger, "get-munki-client-resources", clientResourcesLabel, err, "id", input.ID)
		}
		output, err := clientResourcesResponse(ctx, objects, *resource)
		if err != nil {
			return nil, api.HandlerError(ctx, logger, "get-munki-client-resources", err, "id", input.ID)
		}
		return output, nil
	})
}

func registerUpdateMunkiClientResources(
	humaAPI huma.API,
	service *clientresources.Service,
	objects *bloby.Service,
	logger *slog.Logger,
) {
	huma.Register(humaAPI, huma.Operation{
		OperationID: "update-munki-client-resources",
		Method:      http.MethodPut,
		Path:        clientResourcesIDPath,
		Tags:        []string{api.TagMunkiClientResources},
		Summary:     "Update client resources",
		Errors:      []int{http.StatusBadRequest, http.StatusNotFound, http.StatusConflict},
	}, func(ctx context.Context, input *clientResourcesUpdateInput) (*clientResourcesOutput, error) {
		resource, err := service.Update(ctx, input.ID, input.Body)
		if err != nil {
			return nil, api.ResourceError(ctx, logger, "update-munki-client-resources", clientResourcesLabel, err, "id", input.ID)
		}
		output, err := clientResourcesResponse(ctx, objects, *resource)
		if err != nil {
			return nil, api.HandlerError(ctx, logger, "update-munki-client-resources", err, "id", input.ID)
		}
		return output, nil
	})
}

func registerDeleteMunkiClientResources(
	humaAPI huma.API,
	service *clientresources.Service,
	logger *slog.Logger,
) {
	huma.Register(humaAPI, huma.Operation{
		OperationID:   "delete-munki-client-resources",
		Method:        http.MethodDelete,
		Path:          clientResourcesIDPath,
		Tags:          []string{api.TagMunkiClientResources},
		Summary:       "Delete client resources",
		DefaultStatus: http.StatusNoContent,
		Errors:        []int{http.StatusNotFound},
	}, func(ctx context.Context, input *clientResourcesDeleteInput) (*struct{}, error) {
		if err := service.Delete(ctx, input.ID); err != nil {
			return nil, api.ResourceError(ctx, logger, "delete-munki-client-resources", clientResourcesLabel, err, "id", input.ID)
		}
		return &struct{}{}, nil
	})
}

func registerCreateClientResourcesUpload(
	humaAPI huma.API,
	objects *bloby.Service,
	logger *slog.Logger,
	path string,
	prefix string,
	operationID string,
	summary string,
) {
	huma.Register(humaAPI, huma.Operation{
		OperationID:   operationID,
		Method:        http.MethodPost,
		Path:          path,
		Tags:          []string{api.TagMunkiClientResources},
		Summary:       summary,
		DefaultStatus: http.StatusCreated,
		Errors:        []int{http.StatusBadRequest},
	}, func(ctx context.Context, input *clientResourcesUploadInput) (*munkiUploadOutput, error) {
		object, target, err := objects.BeginDirect(ctx, prefix, input.Body.Filename)
		if err != nil {
			return nil, api.ResourceError(ctx, logger, operationID, clientResourcesLabel, err)
		}
		return newMunkiUploadOutput(object, target), nil
	})
}

func registerDeleteClientResourcesUpload(
	humaAPI huma.API,
	objects *bloby.Service,
	logger *slog.Logger,
	path string,
	prefix string,
	operationID string,
	summary string,
) {
	huma.Register(humaAPI, huma.Operation{
		OperationID:   operationID,
		Method:        http.MethodDelete,
		Path:          path + "/{id}",
		Tags:          []string{api.TagMunkiClientResources},
		Summary:       summary,
		DefaultStatus: http.StatusNoContent,
		Errors:        []int{http.StatusBadRequest, http.StatusNotFound, http.StatusConflict},
	}, func(ctx context.Context, input *clientResourcesUploadIDInput) (*struct{}, error) {
		if err := objects.Delete(ctx, input.ID, prefix); err != nil {
			return nil, api.ResourceError(
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
	objects *bloby.Service,
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
