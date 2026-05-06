package handlers

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/models"
)

const hostsTag = "Hosts"

type hostBody struct {
	ID               string              `json:"id"`
	HardwareUUID     string              `json:"hardware_uuid"`
	DisplayName      string              `json:"display_name"`
	Hostname         string              `json:"hostname"`
	ComputerName     string              `json:"computer_name"`
	HardwareSerial   string              `json:"hardware_serial"`
	HardwareModel    string              `json:"hardware_model"`
	Platform         string              `json:"platform"`
	PlatformLike     string              `json:"platform_like"`
	OSVersion        string              `json:"os_version"`
	OsqueryVersion   string              `json:"osquery_version"`
	OrbitVersion     string              `json:"orbit_version"`
	CPUBrand         string              `json:"cpu_brand"`
	CPULogicalCores  int                 `json:"cpu_logical_cores"`
	CPUPhysicalCores int                 `json:"cpu_physical_cores"`
	PhysicalMemory   int64               `json:"physical_memory"`
	HardwareVendor   string              `json:"hardware_vendor"`
	KernelVersion    string              `json:"kernel_version"`
	DeviceMappings   []deviceMappingBody `json:"device_mappings"`
	EnrolledAt       *time.Time          `json:"enrolled_at,omitempty"`
	LastSeenAt       *time.Time          `json:"last_seen_at,omitempty"`
	DetailUpdatedAt  *time.Time          `json:"detail_updated_at,omitempty"`
	CreatedAt        time.Time           `json:"created_at"`
	UpdatedAt        time.Time           `json:"updated_at"`
}

type deviceMappingBody struct {
	Email  string `json:"email"`
	Source string `json:"source"`
}

type hostSoftwareBody struct {
	ID                string                             `json:"id"`
	Name              string                             `json:"name"`
	DisplayName       string                             `json:"display_name"`
	IconURL           *string                            `json:"icon_url"`
	Source            string                             `json:"source"`
	ExtensionFor      string                             `json:"extension_for"`
	Status            any                                `json:"status"`
	InstalledVersions []hostSoftwareInstalledVersionBody `json:"installed_versions"`
	SoftwarePackage   any                                `json:"software_package"`
	AppStoreApp       any                                `json:"app_store_app"`
}

type hostSoftwareInstalledVersionBody struct {
	Version              string                         `json:"version"`
	BundleIdentifier     string                         `json:"bundle_identifier"`
	InstalledPaths       []string                       `json:"installed_paths"`
	SignatureInformation []pathSignatureInformationBody `json:"signature_information"`
	LastOpenedAt         *time.Time                     `json:"last_opened_at,omitempty"`
}

type pathSignatureInformationBody struct {
	InstalledPath    string `json:"installed_path"`
	TeamIdentifier   string `json:"team_identifier"`
	CDHashSHA256     string `json:"hash_sha256"`
	ExecutableSHA256 string `json:"executable_sha256"`
	ExecutablePath   string `json:"executable_path"`
}

type hostListOutput struct {
	Body hostListBody
}

type hostOutput struct {
	Body hostBody
}

type hostSoftwareOutput struct {
	Body hostSoftwareListBody
}

type hostListBody struct {
	Items []hostBody `json:"items"`
	Count int        `json:"count"`
}

type hostSoftwareListBody struct {
	Items []hostSoftwareBody `json:"items"`
	Count int                `json:"count"`
}

type hostGetInput struct {
	ID string `path:"id"`
}

type hostListInput struct {
	Q               string `query:"q,omitempty"`
	Page            int    `query:"page,omitempty"`
	PerPage         int    `query:"per_page,omitempty"`
	OrderKey        string `query:"order_key,omitempty"`
	OrderDirection  string `query:"order_direction,omitempty"`
	Platform        string `query:"platform,omitempty"`
	SoftwareTitleID string `query:"software_title_id,omitempty"`
	SoftwareID      string `query:"software_id,omitempty"`
}

func (i hostListInput) params() (models.HostListParams, error) {
	titleID, err := parseOptionalPositiveID(i.SoftwareTitleID, "software_title_id")
	if err != nil {
		return models.HostListParams{}, err
	}
	softwareID, err := parseOptionalPositiveID(i.SoftwareID, "software_id")
	if err != nil {
		return models.HostListParams{}, err
	}
	listParams := models.CleanListParams(models.ListParams{
		Q:              i.Q,
		Page:           i.Page,
		PerPage:        i.PerPage,
		OrderKey:       i.OrderKey,
		OrderDirection: i.OrderDirection,
	})
	return models.HostListParams{
		ListParams:      listParams,
		Platform:        strings.TrimSpace(i.Platform),
		SoftwareTitleID: titleID,
		SoftwareID:      softwareID,
	}, nil
}

type hostSoftwareInput struct {
	ID             string   `path:"id"`
	Q              string   `          query:"q,omitempty"`
	Page           int      `          query:"page,omitempty"`
	PerPage        int      `          query:"per_page,omitempty"`
	OrderKey       string   `          query:"order_key,omitempty"`
	OrderDirection string   `          query:"order_direction,omitempty"`
	Source         []string `          query:"source,omitempty"`
}

func (i hostSoftwareInput) params() (int64, models.HostSoftwareListParams, error) {
	id, err := parseHostID(i.ID)
	if err != nil {
		return 0, models.HostSoftwareListParams{}, err
	}
	listParams := models.CleanListParams(models.ListParams{
		Q:              i.Q,
		Page:           i.Page,
		PerPage:        i.PerPage,
		OrderKey:       i.OrderKey,
		OrderDirection: i.OrderDirection,
	})
	return id, models.HostSoftwareListParams{
		ListParams:      listParams,
		SoftwareSources: models.SplitListValues(i.Source),
	}, nil
}

// RegisterHosts registers admin host inventory endpoints.
// Reading hosts is open to admins and viewers; mutation is not exposed yet.
func RegisterHosts(
	api huma.API,
	store *models.HostStore,
	deviceMappings *models.DeviceMappingStore,
	software *models.SoftwareStore,
) {
	registerListHosts(api, store, deviceMappings)
	registerGetHost(api, store, deviceMappings)
	registerHostSoftware(api, store, software)
}

func registerListHosts(api huma.API, store *models.HostStore, mappings *models.DeviceMappingStore) {
	huma.Register(api, huma.Operation{
		OperationID: "list-hosts",
		Method:      http.MethodGet,
		Path:        "/api/hosts",
		Tags:        []string{hostsTag},
		Summary:     "List enrolled hosts",
		Errors:      []int{http.StatusUnauthorized},
	}, func(ctx context.Context, input *hostListInput) (*hostListOutput, error) {
		params, err := input.params()
		if err != nil {
			return nil, err
		}
		hosts, count, err := store.List(ctx, params)
		if err != nil {
			return nil, err
		}
		out := &hostListOutput{Body: hostListBody{Items: make([]hostBody, 0, len(hosts)), Count: count}}
		for i := range hosts {
			body, err := hostResponse(ctx, &hosts[i], mappings)
			if err != nil {
				return nil, err
			}
			out.Body.Items = append(out.Body.Items, body)
		}
		return out, nil
	})
}

func registerGetHost(api huma.API, store *models.HostStore, mappings *models.DeviceMappingStore) {
	huma.Register(api, huma.Operation{
		OperationID: "get-host",
		Method:      http.MethodGet,
		Path:        "/api/hosts/{id}",
		Tags:        []string{hostsTag},
		Summary:     "Get an enrolled host",
		Errors:      []int{http.StatusUnauthorized, http.StatusNotFound},
	}, func(ctx context.Context, input *hostGetInput) (*hostOutput, error) {
		id, err := parseHostID(input.ID)
		if err != nil {
			return nil, err
		}
		host, err := store.GetByID(ctx, id)
		if errors.Is(err, models.ErrNotFound) {
			return nil, huma.Error404NotFound("host not found")
		}
		if err != nil {
			return nil, err
		}
		body, err := hostResponse(ctx, host, mappings)
		if err != nil {
			return nil, err
		}
		return &hostOutput{Body: body}, nil
	})
}

func registerHostSoftware(api huma.API, hosts *models.HostStore, software *models.SoftwareStore) {
	huma.Register(api, huma.Operation{
		OperationID: "list-host-software",
		Method:      http.MethodGet,
		Path:        "/api/hosts/{id}/software",
		Tags:        []string{hostsTag},
		Summary:     "List software installed on a host",
		Errors:      []int{http.StatusUnauthorized, http.StatusNotFound},
	}, func(ctx context.Context, input *hostSoftwareInput) (*hostSoftwareOutput, error) {
		id, params, err := input.params()
		if err != nil {
			return nil, err
		}
		if _, err := hosts.GetByID(ctx, id); errors.Is(err, models.ErrNotFound) {
			return nil, huma.Error404NotFound("host not found")
		} else if err != nil {
			return nil, err
		}
		rows, count, err := software.ListForHost(ctx, id, params)
		if err != nil {
			return nil, err
		}
		out := &hostSoftwareOutput{
			Body: hostSoftwareListBody{Items: make([]hostSoftwareBody, 0, len(rows)), Count: count},
		}
		for _, row := range rows {
			out.Body.Items = append(out.Body.Items, hostSoftwareResponse(row))
		}
		return out, nil
	})
}

func hostResponse(ctx context.Context, host *models.Host, mappings *models.DeviceMappingStore) (hostBody, error) {
	mappingRows, err := mappings.ListForHost(ctx, host.ID)
	if err != nil {
		return hostBody{}, err
	}
	bodyMappings := make([]deviceMappingBody, 0, len(mappingRows))
	for _, mapping := range mappingRows {
		bodyMappings = append(bodyMappings, deviceMappingBody{Email: mapping.Email, Source: mapping.Source})
	}
	return hostBody{
		ID:               models.HostIDString(host.ID),
		HardwareUUID:     host.HardwareUUID,
		DisplayName:      host.DisplayName,
		Hostname:         host.Hostname,
		ComputerName:     host.ComputerName,
		HardwareSerial:   host.HardwareSerial,
		HardwareModel:    host.HardwareModel,
		Platform:         host.Platform,
		PlatformLike:     host.PlatformLike,
		OSVersion:        host.OSVersion,
		OsqueryVersion:   host.OsqueryVersion,
		OrbitVersion:     host.OrbitVersion,
		CPUBrand:         host.CPUBrand,
		CPULogicalCores:  host.CPULogicalCores,
		CPUPhysicalCores: host.CPUPhysicalCores,
		PhysicalMemory:   host.PhysicalMemory,
		HardwareVendor:   host.HardwareVendor,
		KernelVersion:    host.KernelVersion,
		DeviceMappings:   bodyMappings,
		EnrolledAt:       host.EnrolledAt,
		LastSeenAt:       host.LastSeenAt,
		DetailUpdatedAt:  host.DetailUpdatedAt,
		CreatedAt:        host.CreatedAt,
		UpdatedAt:        host.UpdatedAt,
	}, nil
}

func hostSoftwareResponse(row models.HostSoftwareRow) hostSoftwareBody {
	versions := make([]hostSoftwareInstalledVersionBody, 0, len(row.InstalledVersions))
	for _, version := range row.InstalledVersions {
		signatures := make([]pathSignatureInformationBody, 0, len(version.SignatureInformation))
		for _, signature := range version.SignatureInformation {
			signatures = append(signatures, pathSignatureInformationBody{
				InstalledPath:    signature.InstalledPath,
				TeamIdentifier:   signature.TeamIdentifier,
				CDHashSHA256:     signature.CDHashSHA256,
				ExecutableSHA256: signature.ExecutableSHA256,
				ExecutablePath:   signature.ExecutablePath,
			})
		}
		versions = append(versions, hostSoftwareInstalledVersionBody{
			Version:              version.Version,
			BundleIdentifier:     version.BundleIdentifier,
			InstalledPaths:       version.InstalledPaths,
			SignatureInformation: signatures,
			LastOpenedAt:         version.LastOpenedAt,
		})
	}
	return hostSoftwareBody{
		ID:                models.HostIDString(row.ID),
		Name:              row.Name,
		DisplayName:       row.DisplayName,
		IconURL:           row.IconURL,
		Source:            row.Source,
		ExtensionFor:      row.ExtensionFor,
		Status:            nil,
		InstalledVersions: versions,
		SoftwarePackage:   nil,
		AppStoreApp:       nil,
	}
}

func parseHostID(id string) (int64, error) {
	parsed, err := strconv.ParseInt(id, 10, 64)
	if err != nil || parsed <= 0 {
		return 0, huma.Error404NotFound("host not found")
	}
	return parsed, nil
}

func parseOptionalPositiveID(id string, name string) (int64, error) {
	if id == "" {
		return 0, nil
	}
	parsed, err := strconv.ParseInt(id, 10, 64)
	if err != nil || parsed <= 0 {
		return 0, huma.Error400BadRequest(name + " must be a positive integer")
	}
	return parsed, nil
}
