package handlers

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/api/apihelpers"
	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/hosts"
	"github.com/woodleighschool/woodstar/internal/labels"
	"github.com/woodleighschool/woodstar/internal/software"
)

type hostBody struct {
	ID                      int64               `json:"id"`
	HardwareUUID            string              `json:"hardware_uuid"`
	DisplayName             string              `json:"display_name"`
	Hostname                string              `json:"hostname"`
	ComputerName            string              `json:"computer_name"`
	HardwareSerial          string              `json:"hardware_serial"`
	HardwareModel           string              `json:"hardware_model"`
	HardwareVersion         string              `json:"hardware_version"`
	OSName                  string              `json:"os_name"`
	Platform                string              `json:"platform"`
	PlatformLike            string              `json:"platform_like"`
	OSVersion               string              `json:"os_version"`
	OSBuild                 string              `json:"os_build"`
	OsqueryVersion          string              `json:"osquery_version"`
	OrbitVersion            string              `json:"orbit_version"`
	CPUType                 string              `json:"cpu_type"`
	CPUSubtype              string              `json:"cpu_subtype"`
	CPUBrand                string              `json:"cpu_brand"`
	CPULogicalCores         int                 `json:"cpu_logical_cores"`
	CPUPhysicalCores        int                 `json:"cpu_physical_cores"`
	PhysicalMemory          int64               `json:"physical_memory"`
	HardwareVendor          string              `json:"hardware_vendor"`
	KernelVersion           string              `json:"kernel_version"`
	UptimeSeconds           *int64              `json:"uptime_seconds,omitempty"`
	LastRestartedAt         *time.Time          `json:"last_restarted_at,omitempty"`
	DiskSpaceAvailableBytes *int64              `json:"disk_space_available_bytes,omitempty"`
	DiskSpaceTotalBytes     *int64              `json:"disk_space_total_bytes,omitempty"`
	PublicIP                string              `json:"public_ip,omitempty"`
	PrimaryIP               string              `json:"primary_ip,omitempty"`
	PrimaryMAC              string              `json:"primary_mac"`
	DistributedInterval     *int32              `json:"distributed_interval,omitempty"`
	ConfigTLSRefresh        *int32              `json:"config_tls_refresh,omitempty"`
	DeviceMappings          []deviceMappingBody `json:"device_mappings"`
	Labels                  []labelBody         `json:"labels"`
	Users                   []hostUserBody      `json:"users"`
	Batteries               []hostBatteryBody   `json:"batteries"`
	EnrolledAt              *time.Time          `json:"enrolled_at,omitempty"`
	LastSeenAt              *time.Time          `json:"last_seen_at,omitempty"`
	DetailUpdatedAt         *time.Time          `json:"detail_updated_at,omitempty"`
	LabelUpdatedAt          *time.Time          `json:"label_updated_at,omitempty"`
	SoftwareUpdatedAt       *time.Time          `json:"software_updated_at,omitempty"`
	CreatedAt               time.Time           `json:"created_at"`
	UpdatedAt               time.Time           `json:"updated_at"`
}

type hostUserBody struct {
	UID         string `json:"uid"`
	Username    string `json:"username"`
	Type        string `json:"type"`
	Description string `json:"description"`
	Directory   string `json:"directory"`
	Shell       string `json:"shell"`
}

type hostBatteryBody struct {
	SerialNumber     string   `json:"serial_number"`
	Manufacturer     string   `json:"manufacturer"`
	Model            string   `json:"model"`
	Chemistry        string   `json:"chemistry"`
	CycleCount       *int32   `json:"cycle_count,omitempty"`
	Health           string   `json:"health"`
	DesignedCapacity *int32   `json:"designed_capacity,omitempty"`
	MaxCapacity      *int32   `json:"max_capacity,omitempty"`
	CurrentCapacity  *int32   `json:"current_capacity,omitempty"`
	PercentRemaining *float64 `json:"percent_remaining,omitempty"`
}

type deviceMappingBody struct {
	Email  string `json:"email"`
	Source string `json:"source"`
}

type hostSoftwareBody struct {
	ID                int64                              `json:"id"`
	Name              string                             `json:"name"`
	DisplayName       string                             `json:"display_name"`
	Source            string                             `json:"source"`
	ExtensionFor      string                             `json:"extension_for"`
	InstalledVersions []hostSoftwareInstalledVersionBody `json:"installed_versions"`
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
	Status          string `query:"status,omitempty"`
	Platform        string `query:"platform,omitempty"`
	LabelID         string `query:"label_id,omitempty"`
	SoftwareTitleID string `query:"software_title_id,omitempty"`
	SoftwareID      string `query:"software_id,omitempty"`
}

func (i hostListInput) params() (hosts.HostListParams, error) {
	titleID, err := parseOptionalPositiveID(i.SoftwareTitleID, "software_title_id")
	if err != nil {
		return hosts.HostListParams{}, err
	}
	softwareID, err := parseOptionalPositiveID(i.SoftwareID, "software_id")
	if err != nil {
		return hosts.HostListParams{}, err
	}
	labelID, err := parseOptionalPositiveID(i.LabelID, "label_id")
	if err != nil {
		return hosts.HostListParams{}, err
	}
	listParams := dbutil.CleanListParams(dbutil.ListParams{
		Q:              i.Q,
		Page:           i.Page,
		PerPage:        i.PerPage,
		OrderKey:       i.OrderKey,
		OrderDirection: i.OrderDirection,
	})
	return hosts.HostListParams{
		ListParams:      listParams,
		Status:          strings.TrimSpace(i.Status),
		Platform:        strings.TrimSpace(i.Platform),
		LabelID:         labelID,
		SoftwareTitleID: titleID,
		SoftwareID:      softwareID,
	}, nil
}

type hostSoftwareInput struct {
	ID             string   `path:"id"`
	Q              string   `query:"q,omitempty"`
	Page           int      `query:"page,omitempty"`
	PerPage        int      `query:"per_page,omitempty"`
	OrderKey       string   `query:"order_key,omitempty"`
	OrderDirection string   `query:"order_direction,omitempty"`
	Source         []string `query:"source,omitempty"`
}

func (i hostSoftwareInput) params() (int64, software.HostSoftwareListParams, error) {
	id, err := apihelpers.ParseHostID(i.ID)
	if err != nil {
		return 0, software.HostSoftwareListParams{}, err
	}
	listParams := dbutil.CleanListParams(dbutil.ListParams{
		Q:              i.Q,
		Page:           i.Page,
		PerPage:        i.PerPage,
		OrderKey:       i.OrderKey,
		OrderDirection: i.OrderDirection,
	})
	return id, software.HostSoftwareListParams{
		ListParams:      listParams,
		SoftwareSources: dbutil.SplitListValues(i.Source),
	}, nil
}

type hostBulkIDsBody struct {
	IDs []int64 `json:"ids"`
}

type hostBulkDeleteInput struct {
	Body hostBulkIDsBody
}

func (i hostBulkDeleteInput) ids() ([]int64, error) {
	return apihelpers.CleanBulkIDs(i.Body.IDs, "host IDs")
}

// RegisterHosts registers admin host inventory endpoints.
// Reading hosts is open to admins and viewers. Deleting hosts is admin-only.
func RegisterHosts(
	api huma.API,
	hostStore *hosts.HostStore,
	deviceMappings *hosts.DeviceMappingStore,
	softwareStore *software.SoftwareStore,
	labelStore *labels.LabelStore,
) {
	registerListHosts(api, hostStore, deviceMappings)
	registerGetHost(api, hostStore, deviceMappings, labelStore)
	registerDeleteHost(api, hostStore)
	registerBulkDeleteHosts(api, hostStore)
	registerHostSoftware(api, hostStore, softwareStore)
}

func registerListHosts(api huma.API, hostStore *hosts.HostStore, deviceMappings *hosts.DeviceMappingStore) {
	huma.Register(api, huma.Operation{
		OperationID: "list-hosts",
		Method:      http.MethodGet,
		Path:        "/api/hosts",
		Tags:        []string{apihelpers.HostsTag},
		Summary:     "List enrolled hosts",
		Errors:      []int{http.StatusUnauthorized},
	}, func(ctx context.Context, input *hostListInput) (*hostListOutput, error) {
		params, err := input.params()
		if err != nil {
			return nil, err
		}
		hostRows, count, err := hostStore.List(ctx, params)
		if err != nil {
			return nil, apihelpers.ResourceMutationError("host", err)
		}
		out := &hostListOutput{Body: hostListBody{Items: make([]hostBody, 0, len(hostRows)), Count: count}}
		for i := range hostRows {
			body, err := hostResponse(ctx, &hostRows[i], deviceMappings)
			if err != nil {
				return nil, err
			}
			out.Body.Items = append(out.Body.Items, body)
		}
		return out, nil
	})
}

func registerGetHost(
	api huma.API,
	hostStore *hosts.HostStore,
	deviceMappings *hosts.DeviceMappingStore,
	labelStore *labels.LabelStore,
) {
	huma.Register(api, huma.Operation{
		OperationID: "get-host",
		Method:      http.MethodGet,
		Path:        "/api/hosts/{id}",
		Tags:        []string{apihelpers.HostsTag},
		Summary:     "Get an enrolled host",
		Errors:      []int{http.StatusUnauthorized, http.StatusNotFound},
	}, func(ctx context.Context, input *hostGetInput) (*hostOutput, error) {
		id, err := apihelpers.ParseHostID(input.ID)
		if err != nil {
			return nil, err
		}
		host, err := hostStore.GetByID(ctx, id)
		if errors.Is(err, dbutil.ErrNotFound) {
			return nil, huma.Error404NotFound("host not found")
		}
		if err != nil {
			return nil, err
		}
		if err := loadHostDetailChildren(ctx, hostStore, labelStore, host); err != nil {
			return nil, err
		}
		body, err := hostResponse(ctx, host, deviceMappings)
		if err != nil {
			return nil, err
		}
		return &hostOutput{Body: body}, nil
	})
}

func registerDeleteHost(api huma.API, hostStore *hosts.HostStore) {
	huma.Register(api, huma.Operation{
		OperationID: "delete-host",
		Method:      http.MethodDelete,
		Path:        "/api/hosts/{id}",
		Tags:        []string{apihelpers.HostsTag},
		Summary:     "Delete an enrolled host",
		Errors:      []int{http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound},
	}, func(ctx context.Context, input *hostGetInput) (*struct{}, error) {
		if _, err := requireAdmin(ctx); err != nil {
			return nil, err
		}
		id, err := apihelpers.ParseHostID(input.ID)
		if err != nil {
			return nil, err
		}
		if err := hostStore.Delete(ctx, id); err != nil {
			return nil, apihelpers.ResourceMutationError("host", err)
		}
		return &struct{}{}, nil
	})
}

func registerBulkDeleteHosts(api huma.API, hostStore *hosts.HostStore) {
	huma.Register(api, huma.Operation{
		OperationID: "bulk-delete-hosts",
		Method:      http.MethodPost,
		Path:        "/api/hosts/bulk-delete",
		Tags:        []string{apihelpers.HostsTag},
		Summary:     "Delete enrolled hosts",
		Errors:      []int{http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden},
	}, func(ctx context.Context, input *hostBulkDeleteInput) (*struct{}, error) {
		if _, err := requireAdmin(ctx); err != nil {
			return nil, err
		}
		ids, err := input.ids()
		if err != nil {
			return nil, err
		}
		if _, err := hostStore.DeleteMany(ctx, ids); err != nil {
			return nil, err
		}
		return &struct{}{}, nil
	})
}

func loadHostDetailChildren(
	ctx context.Context,
	hostStore *hosts.HostStore,
	labelStore *labels.LabelStore,
	host *hosts.Host,
) error {
	hostLabels, err := labelStore.ListForHost(ctx, host.ID)
	if err != nil {
		return err
	}
	host.Labels = hostLabels
	users, err := hostStore.ListUsers(ctx, host.ID)
	if err != nil {
		return err
	}
	host.Users = users
	batteries, err := hostStore.ListBatteries(ctx, host.ID)
	if err != nil {
		return err
	}
	host.Batteries = batteries
	return nil
}

func registerHostSoftware(api huma.API, hostStore *hosts.HostStore, softwareStore *software.SoftwareStore) {
	huma.Register(api, huma.Operation{
		OperationID: "list-host-software",
		Method:      http.MethodGet,
		Path:        "/api/hosts/{id}/software",
		Tags:        []string{apihelpers.HostsTag},
		Summary:     "List software installed on a host",
		Errors:      []int{http.StatusUnauthorized, http.StatusNotFound},
	}, func(ctx context.Context, input *hostSoftwareInput) (*hostSoftwareOutput, error) {
		id, params, err := input.params()
		if err != nil {
			return nil, err
		}
		if _, err := hostStore.GetByID(ctx, id); errors.Is(err, dbutil.ErrNotFound) {
			return nil, huma.Error404NotFound("host not found")
		} else if err != nil {
			return nil, err
		}
		rows, count, err := softwareStore.ListForHost(ctx, id, params)
		if err != nil {
			return nil, apihelpers.ResourceMutationError("software", err)
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

func hostResponse(ctx context.Context, host *hosts.Host, mappings *hosts.DeviceMappingStore) (hostBody, error) {
	mappingRows, err := mappings.ListForHost(ctx, host.ID)
	if err != nil {
		return hostBody{}, err
	}
	body := hostBody{
		ID:                      host.ID,
		HardwareUUID:            host.HardwareUUID,
		DisplayName:             host.DisplayName,
		Hostname:                host.Hostname,
		ComputerName:            host.ComputerName,
		HardwareSerial:          host.HardwareSerial,
		HardwareModel:           host.HardwareModel,
		HardwareVersion:         host.HardwareVersion,
		OSName:                  host.OSName,
		Platform:                host.Platform,
		PlatformLike:            host.PlatformLike,
		OSVersion:               host.OSVersion,
		OSBuild:                 host.OSBuild,
		OsqueryVersion:          host.OsqueryVersion,
		OrbitVersion:            host.OrbitVersion,
		CPUType:                 host.CPUType,
		CPUSubtype:              host.CPUSubtype,
		CPUBrand:                host.CPUBrand,
		CPULogicalCores:         host.CPULogicalCores,
		CPUPhysicalCores:        host.CPUPhysicalCores,
		PhysicalMemory:          host.PhysicalMemory,
		HardwareVendor:          host.HardwareVendor,
		KernelVersion:           host.KernelVersion,
		UptimeSeconds:           host.UptimeSeconds,
		LastRestartedAt:         host.LastRestartedAt,
		DiskSpaceAvailableBytes: host.DiskSpaceAvailableBytes,
		DiskSpaceTotalBytes:     host.DiskSpaceTotalBytes,
		PrimaryMAC:              host.PrimaryMAC,
		DistributedInterval:     host.DistributedInterval,
		ConfigTLSRefresh:        host.ConfigTLSRefresh,
		DeviceMappings:          deviceMappingResponses(mappingRows),
		Labels:                  labelResponses(host.Labels),
		Users:                   hostUserResponses(host.Users),
		Batteries:               hostBatteryResponses(host.Batteries),
		EnrolledAt:              host.EnrolledAt,
		LastSeenAt:              host.LastSeenAt,
		DetailUpdatedAt:         host.DetailUpdatedAt,
		LabelUpdatedAt:          host.LabelUpdatedAt,
		SoftwareUpdatedAt:       host.SoftwareUpdatedAt,
		CreatedAt:               host.CreatedAt,
		UpdatedAt:               host.UpdatedAt,
	}
	if host.PublicIP != nil {
		body.PublicIP = host.PublicIP.String()
	}
	if host.PrimaryIP != nil {
		body.PrimaryIP = host.PrimaryIP.String()
	}
	return body, nil
}

func deviceMappingResponses(rows []hosts.HostDeviceMapping) []deviceMappingBody {
	out := make([]deviceMappingBody, 0, len(rows))
	for _, mapping := range rows {
		out = append(out, deviceMappingBody{Email: mapping.Email, Source: mapping.Source})
	}
	return out
}

func labelResponses(labels []labels.Label) []labelBody {
	out := make([]labelBody, 0, len(labels))
	for i := range labels {
		out = append(out, labelResponse(&labels[i]))
	}
	return out
}

func hostUserResponses(users []hosts.HostUser) []hostUserBody {
	out := make([]hostUserBody, 0, len(users))
	for _, user := range users {
		out = append(out, hostUserBody{
			UID:         user.UID,
			Username:    user.Username,
			Type:        user.Type,
			Description: user.Description,
			Directory:   user.Directory,
			Shell:       user.Shell,
		})
	}
	return out
}

func hostBatteryResponses(batteries []hosts.HostBattery) []hostBatteryBody {
	out := make([]hostBatteryBody, 0, len(batteries))
	for _, battery := range batteries {
		out = append(out, hostBatteryBody{
			SerialNumber:     battery.SerialNumber,
			Manufacturer:     battery.Manufacturer,
			Model:            battery.Model,
			Chemistry:        battery.Chemistry,
			CycleCount:       battery.CycleCount,
			Health:           battery.Health,
			DesignedCapacity: battery.DesignedCapacity,
			MaxCapacity:      battery.MaxCapacity,
			CurrentCapacity:  battery.CurrentCapacity,
			PercentRemaining: battery.PercentRemaining,
		})
	}
	return out
}

func hostSoftwareResponse(row software.HostSoftwareRow) hostSoftwareBody {
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
		ID:                row.ID,
		Name:              row.Name,
		DisplayName:       row.DisplayName,
		Source:            row.Source,
		ExtensionFor:      row.ExtensionFor,
		InstalledVersions: versions,
	}
}

