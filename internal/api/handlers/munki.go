package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/munki"
)

const (
	munkiTag                = "Munki"
	munkiSoftwareTitlePath  = "/api/munki/software-titles"
	munkiArtifactPath       = "/api/munki/artifacts"
	munkiReleasePath        = "/api/munki/releases"
	munkiAssignmentPath     = "/api/munki/assignments"
	munkiSoftwareTitleLabel = "Munki software title"
	munkiArtifactLabel      = "Munki artifact"
	munkiReleaseLabel       = "Munki release"
	munkiAssignmentLabel    = "Munki assignment"
)

type munkiListInput struct {
	ListQueryInput
}

type munkiSoftwareTitleListOutput struct {
	Body munkiSoftwareTitlePage
}

type munkiSoftwareTitleCreateInput struct {
	Body munkiSoftwareTitleMutation
}

type munkiSoftwareTitleOutput struct {
	Body munkiSoftwareTitle
}

type munkiReleaseListOutput struct {
	Body munkiReleasePage
}

type munkiArtifactListOutput struct {
	Body munkiArtifactPage
}

type munkiArtifactCreateInput struct {
	Body munkiArtifactMutation
}

type munkiArtifactOutput struct {
	Body munkiArtifact
}

type munkiReleaseCreateInput struct {
	Body munkiReleaseMutation
}

type munkiReleaseOutput struct {
	Body munkiRelease
}

type munkiAssignmentListOutput struct {
	Body munkiAssignmentPage
}

type munkiAssignmentCreateInput struct {
	Body munkiAssignmentMutation
}

type munkiAssignmentOutput struct {
	Body munkiAssignment
}

type munkiSoftwareTitlePage struct {
	Items []munkiSoftwareTitle `json:"items"`
	Count int                  `json:"count"`
}

type munkiReleasePage struct {
	Items []munkiRelease `json:"items"`
	Count int            `json:"count"`
}

type munkiArtifactPage struct {
	Items []munkiArtifact `json:"items"`
	Count int             `json:"count"`
}

type munkiAssignmentPage struct {
	Items []munkiAssignment `json:"items"`
	Count int               `json:"count"`
}

type munkiSoftwareTitleMutation struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name,omitempty"`
	Description string `json:"description,omitempty"`
	Category    string `json:"category,omitempty"`
	Developer   string `json:"developer,omitempty"`
}

func (m munkiSoftwareTitleMutation) domain() munki.SoftwareTitleMutation {
	return munki.SoftwareTitleMutation{
		Name:        m.Name,
		DisplayName: m.DisplayName,
		Description: m.Description,
		Category:    m.Category,
		Developer:   m.Developer,
	}
}

type munkiSoftwareTitle struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	DisplayName string    `json:"display_name"`
	Description string    `json:"description"`
	Category    string    `json:"category"`
	Developer   string    `json:"developer"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func munkiSoftwareTitleFromDomain(title munki.SoftwareTitle) munkiSoftwareTitle {
	return munkiSoftwareTitle{
		ID:          title.ID,
		Name:        title.Name,
		DisplayName: title.DisplayName,
		Description: title.Description,
		Category:    title.Category,
		Developer:   title.Developer,
		CreatedAt:   title.CreatedAt,
		UpdatedAt:   title.UpdatedAt,
	}
}

type munkiReleaseMutation struct {
	SoftwareID                int64          `json:"software_id"`
	Name                      string         `json:"name"`
	Version                   string         `json:"version"`
	DisplayName               string         `json:"display_name,omitempty"`
	Pkginfo                   map[string]any `json:"pkginfo"`
	InstallerArtifactID       *int64         `json:"installer_artifact_id,omitempty"`
	InstallerArtifactLocation string         `json:"installer_artifact_location,omitempty"`
	Eligible                  bool           `json:"eligible"`
}

func (m munkiReleaseMutation) domain() (munki.ReleaseMutation, error) {
	pkginfo, err := json.Marshal(m.Pkginfo)
	if err != nil {
		return munki.ReleaseMutation{}, huma.Error400BadRequest("pkginfo must be a JSON object")
	}
	return munki.ReleaseMutation{
		SoftwareID:          m.SoftwareID,
		Name:                m.Name,
		Version:             m.Version,
		DisplayName:         m.DisplayName,
		Pkginfo:             pkginfo,
		InstallerArtifactID: m.InstallerArtifactID,
		Eligible:            m.Eligible,
	}, nil
}

type munkiRelease struct {
	ID                        int64          `json:"id"`
	SoftwareID                int64          `json:"software_id"`
	Name                      string         `json:"name"`
	Version                   string         `json:"version"`
	DisplayName               string         `json:"display_name"`
	Pkginfo                   map[string]any `json:"pkginfo"`
	InstallerArtifactID       *int64         `json:"installer_artifact_id,omitempty"`
	InstallerArtifactLocation string         `json:"installer_artifact_location,omitempty"`
	Eligible                  bool           `json:"eligible"`
	CreatedAt                 time.Time      `json:"created_at"`
	UpdatedAt                 time.Time      `json:"updated_at"`
}

func munkiReleaseFromDomain(release munki.Release) munkiRelease {
	return munkiRelease{
		ID:                        release.ID,
		SoftwareID:                release.SoftwareID,
		Name:                      release.Name,
		Version:                   release.Version,
		DisplayName:               release.DisplayName,
		Pkginfo:                   munkiPkginfoObject(release.Pkginfo),
		InstallerArtifactID:       release.InstallerArtifactID,
		InstallerArtifactLocation: release.InstallerArtifactLocation,
		Eligible:                  release.Eligible,
		CreatedAt:                 release.CreatedAt,
		UpdatedAt:                 release.UpdatedAt,
	}
}

type munkiAssignmentMutation struct {
	ReleaseID       int64                  `json:"release_id"`
	Intent          munki.AssignmentIntent `json:"intent"`
	AllHosts        bool                   `json:"all_hosts"`
	IncludeLabelIDs []int64                `json:"include_label_ids,omitempty"`
	ExcludeLabelIDs []int64                `json:"exclude_label_ids,omitempty"`
	IncludeHostIDs  []int64                `json:"include_host_ids,omitempty"`
	ExcludeHostIDs  []int64                `json:"exclude_host_ids,omitempty"`
}

func (m munkiAssignmentMutation) domain() munki.AssignmentMutation {
	return munki.AssignmentMutation{
		ReleaseID:       m.ReleaseID,
		Intent:          m.Intent,
		AllHosts:        m.AllHosts,
		IncludeLabelIDs: m.IncludeLabelIDs,
		ExcludeLabelIDs: m.ExcludeLabelIDs,
		IncludeHostIDs:  m.IncludeHostIDs,
		ExcludeHostIDs:  m.ExcludeHostIDs,
	}
}

type munkiAssignment struct {
	ID              int64                  `json:"id"`
	ReleaseID       int64                  `json:"release_id"`
	Intent          munki.AssignmentIntent `json:"intent"`
	AllHosts        bool                   `json:"all_hosts"`
	IncludeLabelIDs []int64                `json:"include_label_ids"`
	ExcludeLabelIDs []int64                `json:"exclude_label_ids"`
	IncludeHostIDs  []int64                `json:"include_host_ids"`
	ExcludeHostIDs  []int64                `json:"exclude_host_ids"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
}

func munkiAssignmentFromDomain(assignment munki.Assignment) munkiAssignment {
	return munkiAssignment{
		ID:              assignment.ID,
		ReleaseID:       assignment.ReleaseID,
		Intent:          assignment.Intent,
		AllHosts:        assignment.AllHosts,
		IncludeLabelIDs: assignment.IncludeLabelIDs,
		ExcludeLabelIDs: assignment.ExcludeLabelIDs,
		IncludeHostIDs:  assignment.IncludeHostIDs,
		ExcludeHostIDs:  assignment.ExcludeHostIDs,
		CreatedAt:       assignment.CreatedAt,
		UpdatedAt:       assignment.UpdatedAt,
	}
}

func RegisterMunki(api huma.API, store *munki.Store) {
	registerListMunkiSoftwareTitles(api, store)
	registerCreateMunkiSoftwareTitle(api, store)
	registerListMunkiArtifacts(api, store)
	registerCreateMunkiArtifact(api, store)
	registerListMunkiReleases(api, store)
	registerCreateMunkiRelease(api, store)
	registerListMunkiAssignments(api, store)
	registerCreateMunkiAssignment(api, store)
}

func registerListMunkiSoftwareTitles(api huma.API, store *munki.Store) {
	huma.Register(api, huma.Operation{
		OperationID: "list-munki-software-titles",
		Method:      http.MethodGet,
		Path:        munkiSoftwareTitlePath,
		Tags:        []string{munkiTag},
		Summary:     "List Munki software titles",
		Errors:      []int{http.StatusUnauthorized, http.StatusForbidden},
	}, func(ctx context.Context, input *munkiListInput) (*munkiSoftwareTitleListOutput, error) {
		rows, count, err := store.ListSoftwareTitles(ctx, input.params())
		if err != nil {
			return nil, resourceMutationError(munkiSoftwareTitleLabel, err)
		}
		return &munkiSoftwareTitleListOutput{
			Body: munkiSoftwareTitlePage{Items: munkiSoftwareTitlesFromDomain(rows), Count: count},
		}, nil
	})
}

func registerCreateMunkiSoftwareTitle(api huma.API, store *munki.Store) {
	huma.Register(api, huma.Operation{
		OperationID:   "create-munki-software-title",
		Method:        http.MethodPost,
		Path:          munkiSoftwareTitlePath,
		Tags:          []string{munkiTag},
		Summary:       "Create a Munki software title",
		DefaultStatus: http.StatusCreated,
		Errors: []int{
			http.StatusBadRequest,
			http.StatusUnauthorized,
			http.StatusForbidden,
			http.StatusNotFound,
			http.StatusConflict,
		},
	}, func(ctx context.Context, input *munkiSoftwareTitleCreateInput) (*munkiSoftwareTitleOutput, error) {
		title, err := store.CreateSoftwareTitle(ctx, input.Body.domain())
		if err != nil {
			return nil, resourceMutationError(munkiSoftwareTitleLabel, err)
		}
		return &munkiSoftwareTitleOutput{Body: munkiSoftwareTitleFromDomain(*title)}, nil
	})
}

func registerListMunkiReleases(api huma.API, store *munki.Store) {
	huma.Register(api, huma.Operation{
		OperationID: "list-munki-releases",
		Method:      http.MethodGet,
		Path:        munkiReleasePath,
		Tags:        []string{munkiTag},
		Summary:     "List Munki releases",
		Errors:      []int{http.StatusUnauthorized, http.StatusForbidden},
	}, func(ctx context.Context, input *munkiListInput) (*munkiReleaseListOutput, error) {
		rows, count, err := store.ListReleases(ctx, input.params())
		if err != nil {
			return nil, resourceMutationError(munkiReleaseLabel, err)
		}
		return &munkiReleaseListOutput{Body: munkiReleasePage{Items: munkiReleasesFromDomain(rows), Count: count}}, nil
	})
}

func registerCreateMunkiRelease(api huma.API, store *munki.Store) {
	huma.Register(api, huma.Operation{
		OperationID:   "create-munki-release",
		Method:        http.MethodPost,
		Path:          munkiReleasePath,
		Tags:          []string{munkiTag},
		Summary:       "Create a Munki release",
		DefaultStatus: http.StatusCreated,
		Errors: []int{
			http.StatusBadRequest,
			http.StatusUnauthorized,
			http.StatusForbidden,
			http.StatusNotFound,
			http.StatusConflict,
		},
	}, func(ctx context.Context, input *munkiReleaseCreateInput) (*munkiReleaseOutput, error) {
		params, err := input.Body.domain()
		if err != nil {
			return nil, err
		}
		release, err := store.CreateRelease(ctx, params)
		if err != nil {
			return nil, resourceMutationError(munkiReleaseLabel, err)
		}
		return &munkiReleaseOutput{Body: munkiReleaseFromDomain(*release)}, nil
	})
}

func registerListMunkiAssignments(api huma.API, store *munki.Store) {
	huma.Register(api, huma.Operation{
		OperationID: "list-munki-assignments",
		Method:      http.MethodGet,
		Path:        munkiAssignmentPath,
		Tags:        []string{munkiTag},
		Summary:     "List Munki assignments",
		Errors:      []int{http.StatusUnauthorized, http.StatusForbidden},
	}, func(ctx context.Context, input *munkiListInput) (*munkiAssignmentListOutput, error) {
		rows, count, err := store.ListAssignments(ctx, input.params())
		if err != nil {
			return nil, resourceMutationError(munkiAssignmentLabel, err)
		}
		return &munkiAssignmentListOutput{
			Body: munkiAssignmentPage{Items: munkiAssignmentsFromDomain(rows), Count: count},
		}, nil
	})
}

func registerCreateMunkiAssignment(api huma.API, store *munki.Store) {
	huma.Register(api, huma.Operation{
		OperationID:   "create-munki-assignment",
		Method:        http.MethodPost,
		Path:          munkiAssignmentPath,
		Tags:          []string{munkiTag},
		Summary:       "Create a Munki assignment",
		DefaultStatus: http.StatusCreated,
		Errors:        []int{http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden, http.StatusConflict},
	}, func(ctx context.Context, input *munkiAssignmentCreateInput) (*munkiAssignmentOutput, error) {
		assignment, err := store.CreateAssignment(ctx, input.Body.domain())
		if err != nil {
			return nil, resourceMutationError(munkiAssignmentLabel, err)
		}
		return &munkiAssignmentOutput{Body: munkiAssignmentFromDomain(*assignment)}, nil
	})
}

func munkiSoftwareTitlesFromDomain(rows []munki.SoftwareTitle) []munkiSoftwareTitle {
	items := make([]munkiSoftwareTitle, len(rows))
	for i, row := range rows {
		items[i] = munkiSoftwareTitleFromDomain(row)
	}
	return items
}

func munkiReleasesFromDomain(rows []munki.Release) []munkiRelease {
	items := make([]munkiRelease, len(rows))
	for i, row := range rows {
		items[i] = munkiReleaseFromDomain(row)
	}
	return items
}

func munkiAssignmentsFromDomain(rows []munki.Assignment) []munkiAssignment {
	items := make([]munkiAssignment, len(rows))
	for i, row := range rows {
		items[i] = munkiAssignmentFromDomain(row)
	}
	return items
}

func munkiPkginfoObject(raw []byte) map[string]any {
	var object map[string]any
	if err := json.Unmarshal(raw, &object); err != nil {
		return map[string]any{}
	}
	return object
}
