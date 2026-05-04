package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/models"
)

const softwareTag = "Software"

type softwareTitleBody struct {
	ID               string    `json:"id"`
	Name             string    `json:"name"`
	Version          string    `json:"version"`
	Source           string    `json:"source"`
	BundleIdentifier string    `json:"bundle_identifier"`
	HostCount        int       `json:"host_count"`
	CreatedAt        time.Time `json:"created_at"`
}

type softwareListOutput struct {
	Body []softwareTitleBody
}

// RegisterSoftware registers admin software inventory endpoints.
func RegisterSoftware(api huma.API, store *models.SoftwareStore) {
	huma.Register(api, huma.Operation{
		OperationID: "list-software",
		Method:      http.MethodGet,
		Path:        "/api/software",
		Tags:        []string{softwareTag},
		Summary:     "List software titles",
		Errors:      []int{http.StatusUnauthorized},
	}, func(ctx context.Context, _ *struct{}) (*softwareListOutput, error) {
		titles, err := store.ListTitlesWithHostCount(ctx)
		if err != nil {
			return nil, err
		}
		out := &softwareListOutput{Body: make([]softwareTitleBody, 0, len(titles))}
		for _, title := range titles {
			out.Body = append(out.Body, softwareTitleResponse(title))
		}
		return out, nil
	})
}

func softwareTitleResponse(title models.SoftwareTitle) softwareTitleBody {
	return softwareTitleBody{
		ID:               models.HostIDString(title.ID),
		Name:             title.Name,
		Version:          title.Version,
		Source:           title.Source,
		BundleIdentifier: title.BundleIdentifier,
		HostCount:        title.HostCount,
		CreatedAt:        title.CreatedAt,
	}
}
