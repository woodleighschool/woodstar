package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"mime"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/munki"
)

const (
	munkiTag                 = "Munki"
	munkiSoftwareTitlePath   = "/api/munki/software-titles"
	munkiSoftwareTitleIDPath = "/api/munki/software-titles/{id}"
	munkiArtifactPath        = "/api/munki/artifacts"
	munkiArtifactContentPath = "/api/munki/artifacts/{id}/content"
	munkiArtifactUploadPath  = "/api/munki/artifact-uploads"
	munkiPackagePath         = "/api/munki/packages"
	munkiPackageImportPath   = "/api/munki/packages/import"
	munkiPackageIDPath       = "/api/munki/packages/{id}"
	munkiAssignmentPath      = "/api/munki/assignments"
	munkiAssignmentIDPath    = "/api/munki/assignments/{id}"
	munkiSoftwareTitleLabel  = "Munki software title"
	munkiArtifactLabel       = "Munki artifact"
	munkiPackageLabel        = "Munki package"
	munkiAssignmentLabel     = "Munki assignment"
)

type munkiArtifactStorage interface {
	PresignGet(context.Context, munki.Artifact) (string, error)
	PresignPut(context.Context, string, string, string) (munki.ArtifactUploadURL, error)
	Stat(context.Context, string) (munki.ArtifactObject, error)
}

type munkiListInput struct {
	ListQueryInput
}

type munkiSoftwareTitleGetInput struct {
	ID int64 `path:"id"`
}

type munkiSoftwareTitleCreateInput struct {
	Body munkiSoftwareTitleMutation
}

type munkiSoftwareTitlePatchInput struct {
	ID   int64 `path:"id"`
	Body munkiSoftwareTitleMutation
}

type munkiPackageListInput struct {
	ListQueryInput
	SoftwareID int64 `query:"software_id,omitempty"`
}

type munkiArtifactCreateInput struct {
	Body munkiArtifactMutation
}

type munkiArtifactUploadInput struct {
	Body munkiArtifactUploadMutation
}

type munkiArtifactContentInput struct {
	ID int64 `path:"id"`
}

type munkiPackageGetInput struct {
	ID int64 `path:"id"`
}

type munkiPackageCreateInput struct {
	Body munkiPackageMutation
}

type munkiPackagePatchInput struct {
	ID   int64 `path:"id"`
	Body munkiPackageMutation
}

type munkiPackageImportInput struct {
	Body munkiPackageImportMutation
}

type munkiAssignmentListInput struct {
	ListQueryInput
	SoftwareID int64 `query:"software_id,omitempty"`
}

type munkiAssignmentGetInput struct {
	ID int64 `path:"id"`
}

type munkiAssignmentCreateInput struct {
	Body munkiAssignmentMutation
}

type munkiAssignmentPatchInput struct {
	ID   int64 `path:"id"`
	Body munkiAssignmentMutation
}

type munkiAssignmentReorderInput struct {
	ID   int64 `path:"id"`
	Body munkiAssignmentReorderBody
}

type munkiAssignmentReorderBody struct {
	OrderedIDs []int64 `json:"ordered_ids"`
}

type munkiSoftwareTitleListOutput struct {
	Body Page[munkiSoftwareTitle]
}

type munkiSoftwareTitleOutput struct {
	Body munkiSoftwareTitle
}

type munkiSoftwareTitleDetailOutput struct {
	Body munkiSoftwareTitleDetail
}

type munkiPackageListOutput struct {
	Body Page[munkiPackage]
}

type munkiArtifactOutput struct {
	Body munkiArtifact
}

type munkiArtifactUploadOutput struct {
	Body munkiArtifactUpload
}

type munkiArtifactContentOutput struct {
	Status   int    `json:"-" default:"302"`
	Location string `                       header:"Location"`
}

type munkiPackageOutput struct {
	Body munkiPackage
}

type munkiAssignmentListOutput struct {
	Body Page[munkiAssignment]
}

type munkiAssignmentOutput struct {
	Body munkiAssignment
}

type munkiSoftwareTitleMutation struct {
	Name           string `json:"name"`
	DisplayName    string `json:"display_name,omitempty"`
	Description    string `json:"description,omitempty"`
	Category       string `json:"category,omitempty"`
	Developer      string `json:"developer,omitempty"`
	IconArtifactID *int64 `json:"icon_artifact_id,omitempty"`
}

type munkiSoftwareTitleDetail struct {
	ID             int64             `json:"id"`
	Name           string            `json:"name"`
	DisplayName    string            `json:"display_name"`
	Description    string            `json:"description"`
	Category       string            `json:"category"`
	Developer      string            `json:"developer"`
	IconArtifactID *int64            `json:"icon_artifact_id,omitempty"`
	IconURL        string            `json:"icon_url,omitempty"`
	Packages       []munkiPackage    `json:"packages"`
	Assignments    []munkiAssignment `json:"assignments"`
	CreatedAt      time.Time         `json:"created_at"`
	UpdatedAt      time.Time         `json:"updated_at"`
}

type munkiSoftwareTitle struct {
	ID             int64     `json:"id"`
	Name           string    `json:"name"`
	DisplayName    string    `json:"display_name"`
	Description    string    `json:"description"`
	Category       string    `json:"category"`
	Developer      string    `json:"developer"`
	IconArtifactID *int64    `json:"icon_artifact_id,omitempty"`
	IconURL        string    `json:"icon_url,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
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

type munkiArtifactMutation struct {
	Kind        munki.ArtifactKind `json:"kind"`
	DisplayName string             `json:"display_name,omitempty"`
	Location    string             `json:"location"`
	ContentType string             `json:"content_type,omitempty"`
	SizeBytes   int64              `json:"size_bytes"`
	SHA256      string             `json:"sha256"`
	StorageKey  string             `json:"storage_key"`
}

type munkiArtifactUploadMutation struct {
	Kind        munki.ArtifactKind `json:"kind"`
	Filename    string             `json:"filename"`
	DisplayName string             `json:"display_name,omitempty"`
	ContentType string             `json:"content_type,omitempty"`
	SizeBytes   int64              `json:"size_bytes"`
	SHA256      string             `json:"sha256"`
}

type munkiArtifactUpload struct {
	UploadURL string                `json:"upload_url"`
	Headers   map[string]string     `json:"headers,omitempty"`
	Artifact  munkiArtifactMutation `json:"artifact"`
}

type munkiPackage struct {
	ID                        int64               `json:"id"`
	SoftwareID                int64               `json:"software_id"`
	SoftwareName              string              `json:"software_name"`
	SoftwareDisplayName       string              `json:"software_display_name"`
	Name                      string              `json:"name"`
	Version                   string              `json:"version"`
	DisplayName               string              `json:"display_name"`
	Description               string              `json:"description"`
	Category                  string              `json:"category"`
	Developer                 string              `json:"developer"`
	InstallerType             munki.InstallerType `json:"installer_type"`
	UnattendedInstall         bool                `json:"unattended_install"`
	UnattendedUninstall       bool                `json:"unattended_uninstall"`
	Uninstallable             bool                `json:"uninstallable"`
	UninstallMethod           string              `json:"uninstall_method"`
	RestartAction             munki.RestartAction `json:"restart_action,omitempty"`
	MinimumMunkiVersion       string              `json:"minimum_munki_version"`
	MinimumOSVersion          string              `json:"minimum_os_version"`
	MaximumOSVersion          string              `json:"maximum_os_version"`
	SupportedArchitectures    []string            `json:"supported_architectures"`
	BlockingApplications      []string            `json:"blocking_applications"`
	Requires                  []string            `json:"requires"`
	UpdateFor                 []string            `json:"update_for"`
	OnDemand                  bool                `json:"on_demand"`
	Precache                  bool                `json:"precache"`
	ExtraPkginfo              json.RawMessage     `json:"extra_pkginfo,omitempty"`
	Pkginfo                   json.RawMessage     `json:"pkginfo,omitempty"`
	InstallerArtifactID       *int64              `json:"installer_artifact_id,omitempty"`
	InstallerArtifactLocation string              `json:"installer_artifact_location,omitempty"`
	IconArtifactID            *int64              `json:"icon_artifact_id,omitempty"`
	IconArtifactLocation      string              `json:"icon_artifact_location,omitempty"`
	IconURL                   string              `json:"icon_url,omitempty"`
	Eligible                  bool                `json:"eligible"`
	CreatedAt                 time.Time           `json:"created_at"`
	UpdatedAt                 time.Time           `json:"updated_at"`
}

type munkiPackageMutation struct {
	SoftwareID             int64               `json:"software_id"`
	Name                   string              `json:"name"`
	Version                string              `json:"version"`
	DisplayName            string              `json:"display_name,omitempty"`
	Description            string              `json:"description,omitempty"`
	Category               string              `json:"category,omitempty"`
	Developer              string              `json:"developer,omitempty"`
	InstallerType          munki.InstallerType `json:"installer_type,omitempty"`
	UnattendedInstall      bool                `json:"unattended_install,omitempty"`
	UnattendedUninstall    bool                `json:"unattended_uninstall,omitempty"`
	Uninstallable          bool                `json:"uninstallable,omitempty"`
	UninstallMethod        string              `json:"uninstall_method,omitempty"`
	RestartAction          munki.RestartAction `json:"restart_action,omitempty"`
	MinimumMunkiVersion    string              `json:"minimum_munki_version,omitempty"`
	MinimumOSVersion       string              `json:"minimum_os_version,omitempty"`
	MaximumOSVersion       string              `json:"maximum_os_version,omitempty"`
	SupportedArchitectures []string            `json:"supported_architectures,omitempty"`
	BlockingApplications   []string            `json:"blocking_applications,omitempty"`
	Requires               []string            `json:"requires,omitempty"`
	UpdateFor              []string            `json:"update_for,omitempty"`
	OnDemand               bool                `json:"on_demand,omitempty"`
	Precache               bool                `json:"precache,omitempty"`
	ExtraPkginfo           json.RawMessage     `json:"extra_pkginfo,omitempty"`
	InstallerArtifactID    *int64              `json:"installer_artifact_id,omitempty"`
	IconArtifactID         *int64              `json:"icon_artifact_id,omitempty"`
	Eligible               bool                `json:"eligible"`
}

type munkiPackageImportMutation struct {
	SoftwareID          int64           `json:"software_id,omitempty"`
	Pkginfo             json.RawMessage `json:"pkginfo"`
	InstallerArtifactID *int64          `json:"installer_artifact_id,omitempty"`
	IconArtifactID      *int64          `json:"icon_artifact_id,omitempty"`
	Eligible            *bool           `json:"eligible,omitempty"`
}

type munkiAssignmentMutation struct {
	SoftwareID       int64                   `json:"software_id"`
	Priority         int32                   `json:"priority"`
	LabelID          int64                   `json:"label_id"`
	Effect           munki.AssignmentEffect  `json:"effect"`
	Action           *munki.AssignmentAction `json:"action,omitempty"`
	OptionalInstall  bool                    `json:"optional_install,omitempty"`
	FeaturedItem     bool                    `json:"featured_item,omitempty"`
	PackageSelection *munki.PackageSelection `json:"package_selection,omitempty"`
	PinnedPackageID  *int64                  `json:"pinned_package_id,omitempty"`
}

type munkiAssignment struct {
	ID                   int64                   `json:"id"`
	SoftwareID           int64                   `json:"software_id"`
	SoftwareDisplayName  string                  `json:"software_display_name"`
	Priority             int32                   `json:"priority"`
	LabelID              int64                   `json:"label_id"`
	Effect               munki.AssignmentEffect  `json:"effect"`
	Action               *munki.AssignmentAction `json:"action,omitempty"`
	OptionalInstall      bool                    `json:"optional_install"`
	FeaturedItem         bool                    `json:"featured_item"`
	PackageSelection     *munki.PackageSelection `json:"package_selection,omitempty"`
	PinnedPackageID      *int64                  `json:"pinned_package_id,omitempty"`
	PinnedPackageName    string                  `json:"pinned_package_name,omitempty"`
	PinnedPackageVersion string                  `json:"pinned_package_version,omitempty"`
	CreatedAt            time.Time               `json:"created_at"`
	UpdatedAt            time.Time               `json:"updated_at"`
}

func (input munkiPackageListInput) params() munki.PackageListParams {
	return munki.PackageListParams{
		ListParams: input.ListQueryInput.params(),
		SoftwareID: input.SoftwareID,
	}
}

func (input munkiAssignmentListInput) params() munki.AssignmentListParams {
	return munki.AssignmentListParams{
		ListParams: input.ListQueryInput.params(),
		SoftwareID: input.SoftwareID,
	}
}

// RegisterMunki registers admin endpoints for Munki-managed software.
func RegisterMunki(api huma.API, store *munki.Store, artifactStorage munkiArtifactStorage) {
	registerListMunkiSoftwareTitles(api, store)
	registerCreateMunkiSoftwareTitle(api, store)
	registerGetMunkiSoftwareTitle(api, store)
	registerPatchMunkiSoftwareTitle(api, store)
	registerCreateMunkiArtifact(api, store, artifactStorage)
	registerCreateMunkiArtifactUpload(api, artifactStorage)
	registerGetMunkiArtifactContent(api, store, artifactStorage)
	registerListMunkiPackages(api, store)
	registerCreateMunkiPackage(api, store)
	registerImportMunkiPackage(api, store)
	registerGetMunkiPackage(api, store)
	registerPatchMunkiPackage(api, store)
	registerListMunkiAssignments(api, store)
	registerCreateMunkiAssignment(api, store)
	registerGetMunkiAssignment(api, store)
	registerPatchMunkiAssignment(api, store)
	registerReorderMunkiAssignments(api, store)
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
			Body: Page[munkiSoftwareTitle]{Items: munkiSoftwareTitlesFromDomain(rows), Count: count},
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

func registerGetMunkiSoftwareTitle(api huma.API, store *munki.Store) {
	huma.Register(api, huma.Operation{
		OperationID: "get-munki-software-title",
		Method:      http.MethodGet,
		Path:        munkiSoftwareTitleIDPath,
		Tags:        []string{munkiTag},
		Summary:     "Get a Munki software title",
		Errors:      []int{http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound},
	}, func(ctx context.Context, input *munkiSoftwareTitleGetInput) (*munkiSoftwareTitleDetailOutput, error) {
		detail, err := store.LoadSoftwareTitleDetail(ctx, input.ID)
		if err != nil {
			return nil, resourceMutationError(munkiSoftwareTitleLabel, err)
		}
		return &munkiSoftwareTitleDetailOutput{Body: munkiSoftwareTitleDetailFromDomain(*detail)}, nil
	})
}

func registerPatchMunkiSoftwareTitle(api huma.API, store *munki.Store) {
	huma.Register(api, huma.Operation{
		OperationID: "update-munki-software-title",
		Method:      http.MethodPatch,
		Path:        munkiSoftwareTitleIDPath,
		Tags:        []string{munkiTag},
		Summary:     "Update a Munki software title",
		Errors: []int{
			http.StatusBadRequest,
			http.StatusUnauthorized,
			http.StatusForbidden,
			http.StatusNotFound,
			http.StatusConflict,
		},
	}, func(ctx context.Context, input *munkiSoftwareTitlePatchInput) (*munkiSoftwareTitleOutput, error) {
		title, err := store.UpdateSoftwareTitle(ctx, input.ID, input.Body.domain())
		if err != nil {
			return nil, resourceMutationError(munkiSoftwareTitleLabel, err)
		}
		return &munkiSoftwareTitleOutput{Body: munkiSoftwareTitleFromDomain(*title)}, nil
	})
}

func registerCreateMunkiArtifact(api huma.API, store *munki.Store, artifactStorage munkiArtifactStorage) {
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
			http.StatusServiceUnavailable,
		},
	}, func(ctx context.Context, input *munkiArtifactCreateInput) (*munkiArtifactOutput, error) {
		mutation := input.Body.domain()
		if err := verifyMunkiArtifactObject(ctx, artifactStorage, mutation); err != nil {
			return nil, err
		}
		artifact, err := store.CreateArtifact(ctx, mutation)
		if err != nil {
			return nil, resourceMutationError(munkiArtifactLabel, err)
		}
		return &munkiArtifactOutput{Body: munkiArtifactFromDomain(*artifact)}, nil
	})
}

func registerCreateMunkiArtifactUpload(api huma.API, uploads munkiArtifactStorage) {
	huma.Register(api, huma.Operation{
		OperationID:   "create-munki-artifact-upload",
		Method:        http.MethodPost,
		Path:          munkiArtifactUploadPath,
		Tags:          []string{munkiTag},
		Summary:       "Create a Munki artifact upload URL",
		DefaultStatus: http.StatusCreated,
		Errors: []int{
			http.StatusBadRequest,
			http.StatusUnauthorized,
			http.StatusForbidden,
			http.StatusServiceUnavailable,
		},
	}, func(ctx context.Context, input *munkiArtifactUploadInput) (*munkiArtifactUploadOutput, error) {
		if uploads == nil {
			return nil, huma.Error503ServiceUnavailable("Munki artifact storage is not configured")
		}
		target, err := input.Body.target()
		if err != nil {
			return nil, resourceMutationError(munkiArtifactLabel, err)
		}
		upload, err := uploads.PresignPut(ctx, target.StorageKey, target.ContentType, target.SHA256)
		if err != nil {
			return nil, err
		}
		return &munkiArtifactUploadOutput{
			Body: munkiArtifactUpload{
				UploadURL: upload.URL,
				Headers:   upload.Headers,
				Artifact:  target,
			},
		}, nil
	})
}

func registerGetMunkiArtifactContent(api huma.API, store *munki.Store, artifactStorage munkiArtifactStorage) {
	huma.Register(api, huma.Operation{
		OperationID: "get-munki-artifact-content",
		Method:      http.MethodGet,
		Path:        munkiArtifactContentPath,
		Tags:        []string{munkiTag},
		Summary:     "Get a Munki artifact content URL",
		Errors: []int{
			http.StatusUnauthorized,
			http.StatusForbidden,
			http.StatusNotFound,
			http.StatusServiceUnavailable,
		},
	}, func(ctx context.Context, input *munkiArtifactContentInput) (*munkiArtifactContentOutput, error) {
		if artifactStorage == nil {
			return nil, huma.Error503ServiceUnavailable("Munki artifact storage is not configured")
		}
		artifact, err := store.GetArtifact(ctx, input.ID)
		if err != nil {
			return nil, resourceMutationError(munkiArtifactLabel, err)
		}
		location, err := artifactStorage.PresignGet(ctx, *artifact)
		if err != nil {
			return nil, err
		}
		return &munkiArtifactContentOutput{Status: http.StatusFound, Location: location}, nil
	})
}

func registerListMunkiPackages(api huma.API, store *munki.Store) {
	huma.Register(api, huma.Operation{
		OperationID: "list-munki-packages",
		Method:      http.MethodGet,
		Path:        munkiPackagePath,
		Tags:        []string{munkiTag},
		Summary:     "List Munki packages",
		Errors:      []int{http.StatusUnauthorized, http.StatusForbidden},
	}, func(ctx context.Context, input *munkiPackageListInput) (*munkiPackageListOutput, error) {
		rows, count, err := store.ListPackages(ctx, input.params())
		if err != nil {
			return nil, resourceMutationError(munkiPackageLabel, err)
		}
		return &munkiPackageListOutput{
			Body: Page[munkiPackage]{Items: munkiPackagesFromDomain(rows), Count: count},
		}, nil
	})
}

func registerCreateMunkiPackage(api huma.API, store *munki.Store) {
	huma.Register(api, huma.Operation{
		OperationID:   "create-munki-package",
		Method:        http.MethodPost,
		Path:          munkiPackagePath,
		Tags:          []string{munkiTag},
		Summary:       "Create a Munki package",
		DefaultStatus: http.StatusCreated,
		Errors: []int{
			http.StatusBadRequest,
			http.StatusUnauthorized,
			http.StatusForbidden,
			http.StatusNotFound,
			http.StatusConflict,
		},
	}, func(ctx context.Context, input *munkiPackageCreateInput) (*munkiPackageOutput, error) {
		pkg, err := store.CreatePackage(ctx, input.Body.domain())
		if err != nil {
			return nil, resourceMutationError(munkiPackageLabel, err)
		}
		return &munkiPackageOutput{Body: munkiPackageFromDomain(*pkg)}, nil
	})
}

func registerImportMunkiPackage(api huma.API, store *munki.Store) {
	huma.Register(api, huma.Operation{
		OperationID:   "import-munki-package",
		Method:        http.MethodPost,
		Path:          munkiPackageImportPath,
		Tags:          []string{munkiTag},
		Summary:       "Import a Munki pkginfo package",
		DefaultStatus: http.StatusCreated,
		Errors: []int{
			http.StatusBadRequest,
			http.StatusUnauthorized,
			http.StatusForbidden,
			http.StatusNotFound,
			http.StatusConflict,
		},
	}, func(ctx context.Context, input *munkiPackageImportInput) (*munkiPackageOutput, error) {
		pkg, err := store.ImportPackage(ctx, input.Body.domain())
		if err != nil {
			return nil, resourceMutationError(munkiPackageLabel, err)
		}
		return &munkiPackageOutput{Body: munkiPackageFromDomain(*pkg)}, nil
	})
}

func registerGetMunkiPackage(api huma.API, store *munki.Store) {
	huma.Register(api, huma.Operation{
		OperationID: "get-munki-package",
		Method:      http.MethodGet,
		Path:        munkiPackageIDPath,
		Tags:        []string{munkiTag},
		Summary:     "Get a Munki package",
		Errors:      []int{http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound},
	}, func(ctx context.Context, input *munkiPackageGetInput) (*munkiPackageOutput, error) {
		pkg, err := store.GetPackage(ctx, input.ID)
		if err != nil {
			return nil, resourceMutationError(munkiPackageLabel, err)
		}
		return &munkiPackageOutput{Body: munkiPackageFromDomain(*pkg)}, nil
	})
}

func registerPatchMunkiPackage(api huma.API, store *munki.Store) {
	huma.Register(api, huma.Operation{
		OperationID: "update-munki-package",
		Method:      http.MethodPatch,
		Path:        munkiPackageIDPath,
		Tags:        []string{munkiTag},
		Summary:     "Update a Munki package",
		Errors: []int{
			http.StatusBadRequest,
			http.StatusUnauthorized,
			http.StatusForbidden,
			http.StatusNotFound,
			http.StatusConflict,
		},
	}, func(ctx context.Context, input *munkiPackagePatchInput) (*munkiPackageOutput, error) {
		pkg, err := store.UpdatePackage(ctx, input.ID, input.Body.domain())
		if err != nil {
			return nil, resourceMutationError(munkiPackageLabel, err)
		}
		return &munkiPackageOutput{Body: munkiPackageFromDomain(*pkg)}, nil
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
	}, func(ctx context.Context, input *munkiAssignmentListInput) (*munkiAssignmentListOutput, error) {
		rows, count, err := store.ListAssignments(ctx, input.params())
		if err != nil {
			return nil, resourceMutationError(munkiAssignmentLabel, err)
		}
		return &munkiAssignmentListOutput{
			Body: Page[munkiAssignment]{Items: munkiAssignmentsFromDomain(rows), Count: count},
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
		Errors: []int{
			http.StatusBadRequest,
			http.StatusUnauthorized,
			http.StatusForbidden,
			http.StatusNotFound,
			http.StatusConflict,
		},
	}, func(ctx context.Context, input *munkiAssignmentCreateInput) (*munkiAssignmentOutput, error) {
		assignment, err := store.CreateAssignment(ctx, input.Body.domain())
		if err != nil {
			return nil, resourceMutationError(munkiAssignmentLabel, err)
		}
		return &munkiAssignmentOutput{Body: munkiAssignmentFromDomain(*assignment)}, nil
	})
}

func registerGetMunkiAssignment(api huma.API, store *munki.Store) {
	huma.Register(api, huma.Operation{
		OperationID: "get-munki-assignment",
		Method:      http.MethodGet,
		Path:        munkiAssignmentIDPath,
		Tags:        []string{munkiTag},
		Summary:     "Get a Munki assignment",
		Errors:      []int{http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound},
	}, func(ctx context.Context, input *munkiAssignmentGetInput) (*munkiAssignmentOutput, error) {
		assignment, err := store.GetAssignment(ctx, input.ID)
		if err != nil {
			return nil, resourceMutationError(munkiAssignmentLabel, err)
		}
		return &munkiAssignmentOutput{Body: munkiAssignmentFromDomain(*assignment)}, nil
	})
}

func registerPatchMunkiAssignment(api huma.API, store *munki.Store) {
	huma.Register(api, huma.Operation{
		OperationID: "update-munki-assignment",
		Method:      http.MethodPatch,
		Path:        munkiAssignmentIDPath,
		Tags:        []string{munkiTag},
		Summary:     "Update a Munki assignment",
		Errors: []int{
			http.StatusBadRequest,
			http.StatusUnauthorized,
			http.StatusForbidden,
			http.StatusNotFound,
			http.StatusConflict,
		},
	}, func(ctx context.Context, input *munkiAssignmentPatchInput) (*munkiAssignmentOutput, error) {
		assignment, err := store.UpdateAssignment(ctx, input.ID, input.Body.domain())
		if err != nil {
			return nil, resourceMutationError(munkiAssignmentLabel, err)
		}
		return &munkiAssignmentOutput{Body: munkiAssignmentFromDomain(*assignment)}, nil
	})
}

func registerReorderMunkiAssignments(api huma.API, store *munki.Store) {
	huma.Register(api, huma.Operation{
		OperationID: "reorder-munki-assignments",
		Method:      http.MethodPut,
		Path:        "/api/munki/software-titles/{id}/assignments/order",
		Tags:        []string{munkiTag},
		Summary:     "Reorder Munki assignments",
		Errors:      []int{http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden},
	}, func(ctx context.Context, input *munkiAssignmentReorderInput) (*struct{}, error) {
		if err := store.ReorderAssignments(ctx, input.ID, input.Body.OrderedIDs); err != nil {
			return nil, resourceMutationError(munkiAssignmentLabel, err)
		}
		return &struct{}{}, nil
	})
}

func (input munkiListInput) params() dbutil.ListParams {
	return input.ListQueryInput.params()
}

func munkiSoftwareTitleDetailFromDomain(detail munki.SoftwareTitleDetail) munkiSoftwareTitleDetail {
	return munkiSoftwareTitleDetail{
		ID:             detail.ID,
		Name:           detail.Name,
		DisplayName:    detail.DisplayName,
		Description:    detail.Description,
		Category:       detail.Category,
		Developer:      detail.Developer,
		IconArtifactID: detail.IconArtifactID,
		IconURL:        munkiSoftwareIconURL(detail.SoftwareTitle),
		Packages:       munkiPackagesFromDomain(detail.Packages),
		Assignments:    munkiAssignmentsFromDomain(detail.Assignments),
		CreatedAt:      detail.CreatedAt,
		UpdatedAt:      detail.UpdatedAt,
	}
}

func (body munkiSoftwareTitleMutation) domain() munki.SoftwareTitleMutation {
	return munki.SoftwareTitleMutation{
		Name:           body.Name,
		DisplayName:    body.DisplayName,
		Description:    body.Description,
		Category:       body.Category,
		Developer:      body.Developer,
		IconArtifactID: body.IconArtifactID,
	}
}

func munkiSoftwareTitleFromDomain(title munki.SoftwareTitle) munkiSoftwareTitle {
	return munkiSoftwareTitle{
		ID:             title.ID,
		Name:           title.Name,
		DisplayName:    title.DisplayName,
		Description:    title.Description,
		Category:       title.Category,
		Developer:      title.Developer,
		IconArtifactID: title.IconArtifactID,
		IconURL:        munkiSoftwareIconURL(title),
		CreatedAt:      title.CreatedAt,
		UpdatedAt:      title.UpdatedAt,
	}
}

func munkiSoftwareTitlesFromDomain(rows []munki.SoftwareTitle) []munkiSoftwareTitle {
	items := make([]munkiSoftwareTitle, len(rows))
	for i, row := range rows {
		items[i] = munkiSoftwareTitleFromDomain(row)
	}
	return items
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

func verifyMunkiArtifactObject(
	ctx context.Context,
	artifactStorage munkiArtifactStorage,
	mutation munki.ArtifactMutation,
) error {
	if artifactStorage == nil {
		return huma.Error503ServiceUnavailable("Munki artifact storage is not configured")
	}
	object, err := artifactStorage.Stat(ctx, mutation.StorageKey)
	if errors.Is(err, munki.ErrNotFound) {
		return resourceMutationError(
			munkiArtifactLabel,
			fmt.Errorf("%w: uploaded object does not exist", dbutil.ErrInvalidInput),
		)
	}
	if err != nil {
		return err
	}
	if object.SizeBytes != mutation.SizeBytes {
		return resourceMutationError(
			munkiArtifactLabel,
			fmt.Errorf("%w: uploaded object size does not match artifact metadata", dbutil.ErrInvalidInput),
		)
	}
	if object.SHA256 != "" && object.SHA256 != mutation.SHA256 {
		return resourceMutationError(
			munkiArtifactLabel,
			fmt.Errorf("%w: uploaded object checksum does not match artifact metadata", dbutil.ErrInvalidInput),
		)
	}
	return nil
}

func (body munkiArtifactMutation) domain() munki.ArtifactMutation {
	return munki.ArtifactMutation{
		Kind:        body.Kind,
		DisplayName: body.DisplayName,
		Location:    body.Location,
		ContentType: body.ContentType,
		SizeBytes:   body.SizeBytes,
		SHA256:      body.SHA256,
		StorageKey:  body.StorageKey,
	}
}

func (body munkiArtifactUploadMutation) target() (munkiArtifactMutation, error) {
	filename := cleanArtifactFilename(body.Filename)
	if filename == "" {
		return munkiArtifactMutation{}, fmt.Errorf("%w: filename is required", dbutil.ErrInvalidInput)
	}
	contentType := strings.TrimSpace(body.ContentType)
	if contentType == "" {
		contentType = artifactContentType(filename)
	}
	target := munkiArtifactMutation{
		Kind:        body.Kind,
		DisplayName: body.DisplayName,
		Location:    artifactUploadLocation(body.SHA256, filename),
		ContentType: contentType,
		SizeBytes:   body.SizeBytes,
		SHA256:      strings.TrimSpace(body.SHA256),
	}
	target.StorageKey = artifactStorageKey(target.Kind, target.Location)
	if target.DisplayName == "" {
		target.DisplayName = filename
	}
	if err := target.domain().Validate(); err != nil {
		return munkiArtifactMutation{}, err
	}
	return target, nil
}

func munkiPackageFromDomain(pkg munki.Package) munkiPackage {
	return munkiPackage{
		ID:                        pkg.ID,
		SoftwareID:                pkg.SoftwareID,
		SoftwareName:              pkg.SoftwareName,
		SoftwareDisplayName:       pkg.SoftwareDisplayName,
		Name:                      pkg.Name,
		Version:                   pkg.Version,
		DisplayName:               pkg.DisplayName,
		Description:               pkg.Description,
		Category:                  pkg.Category,
		Developer:                 pkg.Developer,
		InstallerType:             pkg.InstallerType,
		UnattendedInstall:         pkg.UnattendedInstall,
		UnattendedUninstall:       pkg.UnattendedUninstall,
		Uninstallable:             pkg.Uninstallable,
		UninstallMethod:           pkg.UninstallMethod,
		RestartAction:             pkg.RestartAction,
		MinimumMunkiVersion:       pkg.MinimumMunkiVersion,
		MinimumOSVersion:          pkg.MinimumOSVersion,
		MaximumOSVersion:          pkg.MaximumOSVersion,
		SupportedArchitectures:    pkg.SupportedArchitectures,
		BlockingApplications:      pkg.BlockingApplications,
		Requires:                  pkg.Requires,
		UpdateFor:                 pkg.UpdateFor,
		OnDemand:                  pkg.OnDemand,
		Precache:                  pkg.Precache,
		ExtraPkginfo:              pkg.ExtraPkginfo,
		Pkginfo:                   pkg.Pkginfo,
		InstallerArtifactID:       pkg.InstallerArtifactID,
		InstallerArtifactLocation: pkg.InstallerArtifactLocation,
		IconArtifactID:            pkg.IconArtifactID,
		IconArtifactLocation:      pkg.IconArtifactLocation,
		IconURL:                   munkiPackageIconURL(pkg),
		Eligible:                  pkg.Eligible,
		CreatedAt:                 pkg.CreatedAt,
		UpdatedAt:                 pkg.UpdatedAt,
	}
}

func munkiPackagesFromDomain(rows []munki.Package) []munkiPackage {
	items := make([]munkiPackage, len(rows))
	for i, row := range rows {
		items[i] = munkiPackageFromDomain(row)
	}
	return items
}

func (body munkiPackageMutation) domain() munki.PackageMutation {
	return munki.PackageMutation{
		SoftwareID:             body.SoftwareID,
		Name:                   body.Name,
		Version:                body.Version,
		DisplayName:            body.DisplayName,
		Description:            body.Description,
		Category:               body.Category,
		Developer:              body.Developer,
		InstallerType:          body.InstallerType,
		UnattendedInstall:      body.UnattendedInstall,
		UnattendedUninstall:    body.UnattendedUninstall,
		Uninstallable:          body.Uninstallable,
		UninstallMethod:        body.UninstallMethod,
		RestartAction:          body.RestartAction,
		MinimumMunkiVersion:    body.MinimumMunkiVersion,
		MinimumOSVersion:       body.MinimumOSVersion,
		MaximumOSVersion:       body.MaximumOSVersion,
		SupportedArchitectures: body.SupportedArchitectures,
		BlockingApplications:   body.BlockingApplications,
		Requires:               body.Requires,
		UpdateFor:              body.UpdateFor,
		OnDemand:               body.OnDemand,
		Precache:               body.Precache,
		ExtraPkginfo:           body.ExtraPkginfo,
		InstallerArtifactID:    body.InstallerArtifactID,
		IconArtifactID:         body.IconArtifactID,
		Eligible:               body.Eligible,
	}
}

func (body munkiPackageImportMutation) domain() munki.PackageImportMutation {
	return munki.PackageImportMutation{
		SoftwareID:          body.SoftwareID,
		Pkginfo:             body.Pkginfo,
		InstallerArtifactID: body.InstallerArtifactID,
		IconArtifactID:      body.IconArtifactID,
		Eligible:            body.Eligible,
	}
}

func cleanArtifactFilename(filename string) string {
	filename = strings.TrimSpace(strings.ReplaceAll(filename, `\`, "/"))
	filename = path.Base(filename)
	if filename == "." || filename == "/" || filename == "" {
		return ""
	}
	return filename
}

func artifactUploadLocation(sha256 string, filename string) string {
	sha256 = strings.TrimSpace(sha256)
	if len(sha256) >= 12 {
		return sha256[:12] + "/" + filename
	}
	return filename
}

func artifactStorageKey(kind munki.ArtifactKind, location string) string {
	switch kind {
	case munki.ArtifactKindPackage:
		return "pkgs/" + location
	case munki.ArtifactKindIcon:
		return "icons/" + location
	default:
		return string(kind) + "/" + location
	}
}

func artifactContentType(filename string) string {
	if contentType := mime.TypeByExtension(path.Ext(filename)); contentType != "" {
		return contentType
	}
	return "application/octet-stream"
}

func munkiPackageIconURL(pkg munki.Package) string {
	artifactID := pkg.EffectiveIconArtifactID()
	if artifactID == nil {
		return ""
	}
	return fmt.Sprintf("/api/munki/artifacts/%d/content", *artifactID)
}

func munkiSoftwareIconURL(title munki.SoftwareTitle) string {
	if title.IconArtifactID == nil {
		return ""
	}
	return fmt.Sprintf("/api/munki/artifacts/%d/content", *title.IconArtifactID)
}

func munkiAssignmentFromDomain(assignment munki.Assignment) munkiAssignment {
	return munkiAssignment{
		ID:                   assignment.ID,
		SoftwareID:           assignment.SoftwareID,
		SoftwareDisplayName:  assignment.SoftwareDisplayName,
		Priority:             assignment.Priority,
		LabelID:              assignment.LabelID,
		Effect:               assignment.Effect,
		Action:               assignment.Action,
		OptionalInstall:      assignment.OptionalInstall,
		FeaturedItem:         assignment.FeaturedItem,
		PackageSelection:     assignment.PackageSelection,
		PinnedPackageID:      assignment.PinnedPackageID,
		PinnedPackageName:    assignment.PinnedPackageName,
		PinnedPackageVersion: assignment.PinnedPackageVersion,
		CreatedAt:            assignment.CreatedAt,
		UpdatedAt:            assignment.UpdatedAt,
	}
}

func munkiAssignmentsFromDomain(rows []munki.Assignment) []munkiAssignment {
	items := make([]munkiAssignment, len(rows))
	for i, row := range rows {
		items[i] = munkiAssignmentFromDomain(row)
	}
	return items
}

func (body munkiAssignmentMutation) domain() munki.AssignmentMutation {
	return munki.AssignmentMutation{
		SoftwareID:       body.SoftwareID,
		Priority:         body.Priority,
		LabelID:          body.LabelID,
		Effect:           body.Effect,
		Action:           body.Action,
		OptionalInstall:  body.OptionalInstall,
		FeaturedItem:     body.FeaturedItem,
		PackageSelection: body.PackageSelection,
		PinnedPackageID:  body.PinnedPackageID,
	}
}
