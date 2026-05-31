package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/munki"
)

type munkiArtifactMutation struct {
	Kind        munki.ArtifactKind `json:"kind"`
	DisplayName string             `json:"display_name,omitempty"`
	Location    string             `json:"location"`
	ContentType string             `json:"content_type,omitempty"`
	SizeBytes   int64              `json:"size_bytes"`
	SHA256      string             `json:"sha256"`
	StorageKey  string             `json:"storage_key"`
}

func (m munkiArtifactMutation) domain() munki.ArtifactMutation {
	return munki.ArtifactMutation{
		Kind:        m.Kind,
		DisplayName: m.DisplayName,
		Location:    m.Location,
		ContentType: m.ContentType,
		SizeBytes:   m.SizeBytes,
		SHA256:      m.SHA256,
		StorageKey:  m.StorageKey,
	}
}

type munkiArtifact struct {
	ID          int64              `json:"id"`
	Kind        munki.ArtifactKind `json:"kind"`
	DisplayName string             `json:"display_name"`
	Location    string             `json:"location"`
	ContentType string             `json:"content_type"`
	SizeBytes   int64              `json:"size_bytes"`
	SHA256      string             `json:"sha256"`
	StorageKey  string             `json:"storage_key"`
	CreatedAt   time.Time          `json:"created_at"`
	UpdatedAt   time.Time          `json:"updated_at"`
}

func munkiArtifactFromDomain(artifact munki.Artifact) munkiArtifact {
	return munkiArtifact{
		ID:          artifact.ID,
		Kind:        artifact.Kind,
		DisplayName: artifact.DisplayName,
		Location:    artifact.Location,
		ContentType: artifact.ContentType,
		SizeBytes:   artifact.SizeBytes,
		SHA256:      artifact.SHA256,
		StorageKey:  artifact.StorageKey,
		CreatedAt:   artifact.CreatedAt,
		UpdatedAt:   artifact.UpdatedAt,
	}
}

func registerListMunkiArtifacts(api huma.API, store *munki.Store) {
	huma.Register(api, huma.Operation{
		OperationID: "list-munki-artifacts",
		Method:      http.MethodGet,
		Path:        munkiArtifactPath,
		Tags:        []string{munkiTag},
		Summary:     "List Munki artifacts",
		Errors:      []int{http.StatusUnauthorized, http.StatusForbidden},
	}, func(ctx context.Context, input *munkiListInput) (*munkiArtifactListOutput, error) {
		rows, count, err := store.ListArtifacts(ctx, input.params())
		if err != nil {
			return nil, resourceMutationError(munkiArtifactLabel, err)
		}
		return &munkiArtifactListOutput{
			Body: munkiArtifactPage{Items: munkiArtifactsFromDomain(rows), Count: count},
		}, nil
	})
}

func registerCreateMunkiArtifact(api huma.API, store *munki.Store) {
	huma.Register(api, huma.Operation{
		OperationID:   "create-munki-artifact",
		Method:        http.MethodPost,
		Path:          munkiArtifactPath,
		Tags:          []string{munkiTag},
		Summary:       "Create a Munki artifact",
		DefaultStatus: http.StatusCreated,
		Errors: []int{
			http.StatusBadRequest,
			http.StatusUnauthorized,
			http.StatusForbidden,
			http.StatusConflict,
		},
	}, func(ctx context.Context, input *munkiArtifactCreateInput) (*munkiArtifactOutput, error) {
		artifact, err := store.CreateArtifact(ctx, input.Body.domain())
		if err != nil {
			return nil, resourceMutationError(munkiArtifactLabel, err)
		}
		return &munkiArtifactOutput{Body: munkiArtifactFromDomain(*artifact)}, nil
	})
}

func munkiArtifactsFromDomain(rows []munki.Artifact) []munkiArtifact {
	items := make([]munkiArtifact, len(rows))
	for i, row := range rows {
		items[i] = munkiArtifactFromDomain(row)
	}
	return items
}
