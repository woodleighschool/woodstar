package handlers

import (
	"context"
	"errors"
	"net/http"
	"strconv"
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
	ID               string     `json:"id"`
	Name             string     `json:"name"`
	Version          string     `json:"version"`
	Source           string     `json:"source"`
	BundleIdentifier string     `json:"bundle_identifier"`
	LastSeenAt       time.Time  `json:"last_seen_at"`
	LastOpenedAt     *time.Time `json:"last_opened_at,omitempty"`
}

type hostListOutput struct {
	Body []hostBody
}

type hostOutput struct {
	Body hostBody
}

type hostSoftwareOutput struct {
	Body []hostSoftwareBody
}

type hostGetInput struct {
	ID string `path:"id"`
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
	}, func(ctx context.Context, _ *struct{}) (*hostListOutput, error) {
		hosts, err := store.List(ctx)
		if err != nil {
			return nil, err
		}
		out := &hostListOutput{Body: make([]hostBody, 0, len(hosts))}
		for i := range hosts {
			body, err := hostResponse(ctx, &hosts[i], mappings)
			if err != nil {
				return nil, err
			}
			out.Body = append(out.Body, body)
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
	}, func(ctx context.Context, input *hostGetInput) (*hostSoftwareOutput, error) {
		id, err := parseHostID(input.ID)
		if err != nil {
			return nil, err
		}
		if _, err := hosts.GetByID(ctx, id); errors.Is(err, models.ErrNotFound) {
			return nil, huma.Error404NotFound("host not found")
		} else if err != nil {
			return nil, err
		}
		rows, err := software.ListForHost(ctx, id)
		if err != nil {
			return nil, err
		}
		out := &hostSoftwareOutput{Body: make([]hostSoftwareBody, 0, len(rows))}
		for _, row := range rows {
			out.Body = append(out.Body, hostSoftwareResponse(row))
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
	return hostSoftwareBody{
		ID:               models.HostIDString(row.ID),
		Name:             row.Name,
		Version:          row.Version,
		Source:           row.Source,
		BundleIdentifier: row.BundleIdentifier,
		LastSeenAt:       row.LastSeenAt,
		LastOpenedAt:     row.LastOpenedAt,
	}
}

func parseHostID(id string) (int64, error) {
	parsed, err := strconv.ParseInt(id, 10, 64)
	if err != nil || parsed <= 0 {
		return 0, huma.Error404NotFound("host not found")
	}
	return parsed, nil
}
