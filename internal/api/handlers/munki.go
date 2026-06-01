package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/munki"
)

const (
	munkiTag                 = "Munki"
	munkiSoftwareTitlePath   = "/api/munki/software-titles"
	munkiSoftwareTitleIDPath = "/api/munki/software-titles/{id}"
	munkiPackagePath         = "/api/munki/packages"
	munkiPackageIDPath       = "/api/munki/packages/{id}"
	munkiDeploymentPath      = "/api/munki/deployments"
	munkiDeploymentIDPath    = "/api/munki/deployments/{id}"
	munkiSoftwareTitleLabel  = "Munki software title"
	munkiPackageLabel        = "Munki package"
	munkiDeploymentLabel     = "Munki deployment"
)

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

type munkiPackageGetInput struct {
	ID int64 `path:"id"`
}

type munkiPackageCreateInput struct {
	Body munkiPackageMutation
}

type munkiDeploymentListInput struct {
	ListQueryInput
	SoftwareID int64 `query:"software_id,omitempty"`
}

type munkiDeploymentGetInput struct {
	ID int64 `path:"id"`
}

type munkiDeploymentCreateInput struct {
	Body munkiDeploymentMutation
}

type munkiDeploymentReorderInput struct {
	ID   int64 `path:"id"`
	Body munkiDeploymentReorderBody
}

type munkiDeploymentReorderBody struct {
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

type munkiPackageOutput struct {
	Body munkiPackage
}

type munkiDeploymentListOutput struct {
	Body Page[munkiDeployment]
}

type munkiDeploymentOutput struct {
	Body munkiDeployment
}

type munkiSoftwareTitleMutation struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name,omitempty"`
	Description string `json:"description,omitempty"`
	Category    string `json:"category,omitempty"`
	Developer   string `json:"developer,omitempty"`
}

type munkiSoftwareTitleDetail struct {
	ID          int64             `json:"id"`
	Name        string            `json:"name"`
	DisplayName string            `json:"display_name"`
	Description string            `json:"description"`
	Category    string            `json:"category"`
	Developer   string            `json:"developer"`
	Packages    []munkiPackage    `json:"packages"`
	Deployments []munkiDeployment `json:"deployments"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
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

type munkiPackage struct {
	ID                  int64                 `json:"id"`
	SoftwareID          int64                 `json:"software_id"`
	SoftwareName        string                `json:"software_name"`
	SoftwareDisplayName string                `json:"software_display_name"`
	Name                string                `json:"name"`
	Version             string                `json:"version"`
	DisplayName         string                `json:"display_name"`
	Description         string                `json:"description"`
	Category            string                `json:"category"`
	Developer           string                `json:"developer"`
	Metadata            munki.PackageMetadata `json:"metadata"`
	Eligible            bool                  `json:"eligible"`
	CreatedAt           time.Time             `json:"created_at"`
	UpdatedAt           time.Time             `json:"updated_at"`
}

type munkiPackageMutation struct {
	SoftwareID  int64                 `json:"software_id"`
	Name        string                `json:"name"`
	Version     string                `json:"version"`
	DisplayName string                `json:"display_name,omitempty"`
	Description string                `json:"description,omitempty"`
	Category    string                `json:"category,omitempty"`
	Developer   string                `json:"developer,omitempty"`
	Metadata    munki.PackageMetadata `json:"metadata"`
	Eligible    bool                  `json:"eligible"`
}

type munkiDeploymentMutation struct {
	PackageID       int64                  `json:"package_id"`
	Intent          munki.DeploymentIntent `json:"intent"`
	AllHosts        bool                   `json:"all_hosts"`
	IncludeLabelIDs []int64                `json:"include_label_ids,omitempty"`
	ExcludeLabelIDs []int64                `json:"exclude_label_ids,omitempty"`
	IncludeHostIDs  []int64                `json:"include_host_ids,omitempty"`
	ExcludeHostIDs  []int64                `json:"exclude_host_ids,omitempty"`
}

type munkiDeployment struct {
	ID                  int64                  `json:"id"`
	PackageID           int64                  `json:"package_id"`
	PackageName         string                 `json:"package_name"`
	PackageVersion      string                 `json:"package_version"`
	SoftwareID          int64                  `json:"software_id"`
	SoftwareDisplayName string                 `json:"software_display_name"`
	Intent              munki.DeploymentIntent `json:"intent"`
	Position            int32                  `json:"position"`
	AllHosts            bool                   `json:"all_hosts"`
	IncludeLabelIDs     []int64                `json:"include_label_ids"`
	ExcludeLabelIDs     []int64                `json:"exclude_label_ids"`
	IncludeHostIDs      []int64                `json:"include_host_ids"`
	ExcludeHostIDs      []int64                `json:"exclude_host_ids"`
	CreatedAt           time.Time              `json:"created_at"`
	UpdatedAt           time.Time              `json:"updated_at"`
}

func (input munkiPackageListInput) params() munki.PackageListParams {
	return munki.PackageListParams{
		ListParams: input.ListQueryInput.params(),
		SoftwareID: input.SoftwareID,
	}
}

func (input munkiDeploymentListInput) params() munki.DeploymentListParams {
	return munki.DeploymentListParams{
		ListParams: input.ListQueryInput.params(),
		SoftwareID: input.SoftwareID,
	}
}

// RegisterMunki registers admin endpoints for Munki-managed software.
func RegisterMunki(api huma.API, store *munki.Store) {
	registerListMunkiSoftwareTitles(api, store)
	registerCreateMunkiSoftwareTitle(api, store)
	registerGetMunkiSoftwareTitle(api, store)
	registerPatchMunkiSoftwareTitle(api, store)
	registerListMunkiPackages(api, store)
	registerCreateMunkiPackage(api, store)
	registerGetMunkiPackage(api, store)
	registerListMunkiDeployments(api, store)
	registerCreateMunkiDeployment(api, store)
	registerGetMunkiDeployment(api, store)
	registerReorderMunkiDeployments(api, store)
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

func registerListMunkiDeployments(api huma.API, store *munki.Store) {
	huma.Register(api, huma.Operation{
		OperationID: "list-munki-deployments",
		Method:      http.MethodGet,
		Path:        munkiDeploymentPath,
		Tags:        []string{munkiTag},
		Summary:     "List Munki deployments",
		Errors:      []int{http.StatusUnauthorized, http.StatusForbidden},
	}, func(ctx context.Context, input *munkiDeploymentListInput) (*munkiDeploymentListOutput, error) {
		rows, count, err := store.ListDeployments(ctx, input.params())
		if err != nil {
			return nil, resourceMutationError(munkiDeploymentLabel, err)
		}
		return &munkiDeploymentListOutput{
			Body: Page[munkiDeployment]{Items: munkiDeploymentsFromDomain(rows), Count: count},
		}, nil
	})
}

func registerCreateMunkiDeployment(api huma.API, store *munki.Store) {
	huma.Register(api, huma.Operation{
		OperationID:   "create-munki-deployment",
		Method:        http.MethodPost,
		Path:          munkiDeploymentPath,
		Tags:          []string{munkiTag},
		Summary:       "Create a Munki deployment",
		DefaultStatus: http.StatusCreated,
		Errors: []int{
			http.StatusBadRequest,
			http.StatusUnauthorized,
			http.StatusForbidden,
			http.StatusNotFound,
			http.StatusConflict,
		},
	}, func(ctx context.Context, input *munkiDeploymentCreateInput) (*munkiDeploymentOutput, error) {
		deployment, err := store.CreateDeployment(ctx, input.Body.domain())
		if err != nil {
			return nil, resourceMutationError(munkiDeploymentLabel, err)
		}
		return &munkiDeploymentOutput{Body: munkiDeploymentFromDomain(*deployment)}, nil
	})
}

func registerGetMunkiDeployment(api huma.API, store *munki.Store) {
	huma.Register(api, huma.Operation{
		OperationID: "get-munki-deployment",
		Method:      http.MethodGet,
		Path:        munkiDeploymentIDPath,
		Tags:        []string{munkiTag},
		Summary:     "Get a Munki deployment",
		Errors:      []int{http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound},
	}, func(ctx context.Context, input *munkiDeploymentGetInput) (*munkiDeploymentOutput, error) {
		deployment, err := store.GetDeployment(ctx, input.ID)
		if err != nil {
			return nil, resourceMutationError(munkiDeploymentLabel, err)
		}
		return &munkiDeploymentOutput{Body: munkiDeploymentFromDomain(*deployment)}, nil
	})
}

func registerReorderMunkiDeployments(api huma.API, store *munki.Store) {
	huma.Register(api, huma.Operation{
		OperationID: "reorder-munki-deployments",
		Method:      http.MethodPut,
		Path:        "/api/munki/software-titles/{id}/deployments/order",
		Tags:        []string{munkiTag},
		Summary:     "Reorder Munki deployments",
		Errors:      []int{http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden},
	}, func(ctx context.Context, input *munkiDeploymentReorderInput) (*struct{}, error) {
		if err := store.ReorderDeployments(ctx, input.ID, input.Body.OrderedIDs); err != nil {
			return nil, resourceMutationError(munkiDeploymentLabel, err)
		}
		return &struct{}{}, nil
	})
}

func (input munkiListInput) params() dbutil.ListParams {
	return input.ListQueryInput.params()
}

func munkiSoftwareTitleDetailFromDomain(detail munki.SoftwareTitleDetail) munkiSoftwareTitleDetail {
	return munkiSoftwareTitleDetail{
		ID:          detail.ID,
		Name:        detail.Name,
		DisplayName: detail.DisplayName,
		Description: detail.Description,
		Category:    detail.Category,
		Developer:   detail.Developer,
		Packages:    munkiPackagesFromDomain(detail.Packages),
		Deployments: munkiDeploymentsFromDomain(detail.Deployments),
		CreatedAt:   detail.CreatedAt,
		UpdatedAt:   detail.UpdatedAt,
	}
}

func (body munkiSoftwareTitleMutation) domain() munki.SoftwareTitleMutation {
	return munki.SoftwareTitleMutation{
		Name:        body.Name,
		DisplayName: body.DisplayName,
		Description: body.Description,
		Category:    body.Category,
		Developer:   body.Developer,
	}
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

func munkiSoftwareTitlesFromDomain(rows []munki.SoftwareTitle) []munkiSoftwareTitle {
	items := make([]munkiSoftwareTitle, len(rows))
	for i, row := range rows {
		items[i] = munkiSoftwareTitleFromDomain(row)
	}
	return items
}

func munkiPackageFromDomain(pkg munki.Package) munkiPackage {
	return munkiPackage{
		ID:                  pkg.ID,
		SoftwareID:          pkg.SoftwareID,
		SoftwareName:        pkg.SoftwareName,
		SoftwareDisplayName: pkg.SoftwareDisplayName,
		Name:                pkg.Name,
		Version:             pkg.Version,
		DisplayName:         pkg.DisplayName,
		Description:         pkg.Description,
		Category:            pkg.Category,
		Developer:           pkg.Developer,
		Metadata:            pkg.Metadata,
		Eligible:            pkg.Eligible,
		CreatedAt:           pkg.CreatedAt,
		UpdatedAt:           pkg.UpdatedAt,
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
		SoftwareID:  body.SoftwareID,
		Name:        body.Name,
		Version:     body.Version,
		DisplayName: body.DisplayName,
		Description: body.Description,
		Category:    body.Category,
		Developer:   body.Developer,
		Metadata:    body.Metadata,
		Eligible:    body.Eligible,
	}
}

func munkiDeploymentFromDomain(deployment munki.Deployment) munkiDeployment {
	return munkiDeployment{
		ID:                  deployment.ID,
		PackageID:           deployment.PackageID,
		PackageName:         deployment.PackageName,
		PackageVersion:      deployment.PackageVersion,
		SoftwareID:          deployment.SoftwareID,
		SoftwareDisplayName: deployment.SoftwareDisplayName,
		Intent:              deployment.Intent,
		Position:            deployment.Position,
		AllHosts:            deployment.AllHosts,
		IncludeLabelIDs:     deployment.IncludeLabelIDs,
		ExcludeLabelIDs:     deployment.ExcludeLabelIDs,
		IncludeHostIDs:      deployment.IncludeHostIDs,
		ExcludeHostIDs:      deployment.ExcludeHostIDs,
		CreatedAt:           deployment.CreatedAt,
		UpdatedAt:           deployment.UpdatedAt,
	}
}

func munkiDeploymentsFromDomain(rows []munki.Deployment) []munkiDeployment {
	items := make([]munkiDeployment, len(rows))
	for i, row := range rows {
		items[i] = munkiDeploymentFromDomain(row)
	}
	return items
}

func (body munkiDeploymentMutation) domain() munki.DeploymentMutation {
	return munki.DeploymentMutation{
		PackageID:       body.PackageID,
		Intent:          body.Intent,
		AllHosts:        body.AllHosts,
		IncludeLabelIDs: body.IncludeLabelIDs,
		ExcludeLabelIDs: body.ExcludeLabelIDs,
		IncludeHostIDs:  body.IncludeHostIDs,
		ExcludeHostIDs:  body.ExcludeHostIDs,
	}
}
