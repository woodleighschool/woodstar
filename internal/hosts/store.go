package hosts

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodstar/internal/database"
	"github.com/woodleighschool/woodstar/internal/database/sqlc"
	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/labels"
)

// Store persists hosts.
type Store struct {
	db     *database.DB
	q      *sqlc.Queries
	labels hostLabelReader
}

type hostLabelReader interface {
	ListForHost(context.Context, int64) ([]labels.Label, error)
}

func NewStore(db *database.DB) *Store {
	return &Store{db: db, q: db.Queries(), labels: labels.NewStore(db)}
}

// UpsertOnOrbitEnroll creates or refreshes a host from Orbit enroll.
func (s *Store) UpsertOnOrbitEnroll(ctx context.Context, update InventoryUpdate) (*Host, error) {
	row, err := s.q.UpsertHostOnOrbitEnroll(ctx, sqlc.UpsertHostOnOrbitEnrollParams{
		HardwareUUID:            update.Hardware.UUID,
		DisplayName:             inventoryDisplayName(update.Hardware.UUID, update.Hostname, update.ComputerName),
		Hostname:                update.Hostname,
		ComputerName:            update.ComputerName,
		HardwareSerial:          update.Hardware.Serial,
		HardwareModelIdentifier: update.Hardware.ModelIdentifier,
		OrbitNodeKey:            update.OrbitNodeKey,
	})
	if err != nil {
		return nil, err
	}
	if err := s.q.AddHostToAllHostsLabel(ctx, sqlc.AddHostToAllHostsLabelParams{
		HostID:     row.ID,
		BuiltinKey: string(labels.BuiltinKeyAllHosts),
	}); err != nil {
		return nil, err
	}
	return new(hostFromSQLC(row)), nil
}

// UpsertOnOsqueryEnroll creates or refreshes a host from osquery enroll.
func (s *Store) UpsertOnOsqueryEnroll(ctx context.Context, update InventoryUpdate) (*Host, error) {
	row, err := s.q.UpsertHostOnOsqueryEnroll(ctx, sqlc.UpsertHostOnOsqueryEnrollParams{
		HardwareUUID:            update.Hardware.UUID,
		DisplayName:             inventoryDisplayName(update.Hardware.UUID, update.Hostname, update.ComputerName),
		Hostname:                update.Hostname,
		ComputerName:            update.ComputerName,
		HardwareSerial:          update.Hardware.Serial,
		HardwareModelIdentifier: update.Hardware.ModelIdentifier,
		OSName:                  update.OS.Name,
		OSVersion:               update.OS.Version,
		OSBuild:                 update.OS.Build,
		OSPlatform:              update.OS.Platform,
		OsqueryVersion:          update.Agents.Osquery.Version,
		OsqueryNodeKey:          update.OsqueryNodeKey,
		OrbitVersion:            update.Agents.Orbit.Version,
		CPUType:                 update.Hardware.CPU.Architecture,
		CPUSubtype:              update.Hardware.CPU.Subtype,
		CPUBrand:                update.Hardware.CPU.Brand,
		CPULogicalCores:         update.Hardware.CPU.LogicalCores,
		CPUPhysicalCores:        update.Hardware.CPU.PhysicalCores,
		MemoryBytes:             update.Hardware.MemoryBytes,
		HardwareVendor:          update.Hardware.Vendor,
		OSKernelVersion:         update.OS.KernelVersion,
	})
	if err != nil {
		return nil, err
	}
	if err := s.q.AddHostToAllHostsLabel(ctx, sqlc.AddHostToAllHostsLabelParams{
		HostID:     row.ID,
		BuiltinKey: string(labels.BuiltinKeyAllHosts),
	}); err != nil {
		return nil, err
	}
	return new(hostFromSQLC(row)), nil
}

func (s *Store) List(ctx context.Context, params HostListParams) ([]Host, int, error) {
	where, args, err := hostListWhere(params)
	if err != nil {
		return nil, 0, err
	}
	listQuery := hostListQuery(params, where, args)
	dbHosts, count, err := dbutil.ListWithCount[sqlc.Host](ctx, s.db.Pool(), listQuery)
	if err != nil {
		return nil, 0, err
	}
	hosts := make([]Host, len(dbHosts))
	for i, row := range dbHosts {
		hosts[i] = hostFromSQLC(row)
	}
	if err := s.attachUserAffinity(ctx, hosts); err != nil {
		return nil, 0, err
	}
	return hosts, count, nil
}

func (s *Store) GetByID(ctx context.Context, id int64) (*Host, error) {
	row, err := s.q.GetHostByID(ctx, sqlc.GetHostByIDParams{ID: id})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, dbutil.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return new(hostFromSQLC(row)), nil
}

// GetByHardwareSerial returns the existing host with serial.
func (s *Store) GetByHardwareSerial(ctx context.Context, serial string) (*Host, error) {
	serial = strings.TrimSpace(serial)
	if serial == "" {
		return nil, dbutil.ErrNotFound
	}
	rows, err := s.q.ListHostsByHardwareSerial(ctx, sqlc.ListHostsByHardwareSerialParams{HardwareSerial: serial})
	if err != nil {
		return nil, err
	}
	switch len(rows) {
	case 0:
		return nil, dbutil.ErrNotFound
	case 1:
		return new(hostFromSQLC(rows[0])), nil
	default:
		return nil, fmt.Errorf("multiple hosts have hardware serial %q", serial)
	}
}

func (s *Store) Delete(ctx context.Context, id int64) error {
	_, err := s.q.DeleteHost(ctx, sqlc.DeleteHostParams{ID: id})
	if errors.Is(err, pgx.ErrNoRows) {
		return dbutil.ErrNotFound
	}
	return err
}

// DeleteMany removes hosts. Missing IDs are fine.
func (s *Store) DeleteMany(ctx context.Context, ids []int64) (int, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	deleted, err := s.q.DeleteHosts(ctx, sqlc.DeleteHostsParams{Ids: ids})
	if err != nil {
		return 0, err
	}
	return len(deleted), nil
}

func (s *Store) GetByOrbitNodeKey(ctx context.Context, nodeKey string) (*Host, error) {
	row, err := s.q.TouchHostByOrbitNodeKey(ctx, sqlc.TouchHostByOrbitNodeKeyParams{OrbitNodeKey: nodeKey})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, dbutil.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return new(hostFromSQLC(row)), nil
}

func (s *Store) GetByOsqueryNodeKey(ctx context.Context, nodeKey string) (*Host, error) {
	row, err := s.q.TouchHostByOsqueryNodeKey(ctx, sqlc.TouchHostByOsqueryNodeKeyParams{OsqueryNodeKey: nodeKey})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, dbutil.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return new(hostFromSQLC(row)), nil
}

func (s *Store) ApplyInventory(ctx context.Context, hostID int64, update InventoryUpdate) error {
	return s.q.ApplyHostInventory(ctx, sqlc.ApplyHostInventoryParams{
		ID:                                hostID,
		Hostname:                          update.Hostname,
		ComputerName:                      update.ComputerName,
		HardwareSerial:                    update.Hardware.Serial,
		HardwareModelIdentifier:           update.Hardware.ModelIdentifier,
		OSName:                            update.OS.Name,
		OSVersion:                         update.OS.Version,
		OSBuild:                           update.OS.Build,
		OSPlatform:                        update.OS.Platform,
		OsqueryVersion:                    update.Agents.Osquery.Version,
		OrbitVersion:                      update.Agents.Orbit.Version,
		CPUType:                           update.Hardware.CPU.Architecture,
		CPUSubtype:                        update.Hardware.CPU.Subtype,
		CPUBrand:                          update.Hardware.CPU.Brand,
		CPULogicalCores:                   update.Hardware.CPU.LogicalCores,
		CPUPhysicalCores:                  update.Hardware.CPU.PhysicalCores,
		MemoryBytes:                       update.Hardware.MemoryBytes,
		HardwareVendor:                    update.Hardware.Vendor,
		OSKernelVersion:                   update.OS.KernelVersion,
		LastRestartedAt:                   update.Timestamps.LastRestartedAt,
		BootVolumeAvailableBytes:          update.Storage.BootVolume.AvailableBytes,
		BootVolumeTotalBytes:              update.Storage.BootVolume.TotalBytes,
		LastRemoteIP:                      update.Network.LastRemoteIP,
		PrimaryIP:                         update.Network.PrimaryIP,
		PrimaryMAC:                        update.Network.PrimaryMAC,
		OsqueryDistributedIntervalSeconds: update.Agents.Osquery.DistributedIntervalSeconds,
		OsqueryConfigRefreshSeconds:       update.Agents.Osquery.ConfigRefreshSeconds,
	})
}

func (s *Store) ReplaceUsers(ctx context.Context, hostID int64, users []HostUser) error {
	return s.db.WithTx(ctx, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		if err := q.DeleteHostUsers(ctx, sqlc.DeleteHostUsersParams{HostID: hostID}); err != nil {
			return err
		}
		for _, user := range users {
			if user.UID == "" || user.Username == "" {
				continue
			}
			if err := q.InsertHostUser(ctx, sqlc.InsertHostUserParams{
				HostID:      hostID,
				UID:         user.UID,
				Username:    user.Username,
				Type:        user.Type,
				Description: user.Description,
				Directory:   user.Directory,
				Shell:       user.Shell,
			}); err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *Store) ReplaceBatteries(ctx context.Context, hostID int64, batteries []HostBattery) error {
	return s.db.WithTx(ctx, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		if err := q.DeleteHostBatteries(ctx, sqlc.DeleteHostBatteriesParams{HostID: hostID}); err != nil {
			return err
		}
		for _, battery := range batteries {
			if battery.SerialNumber == "" {
				continue
			}
			if err := q.InsertHostBattery(ctx, sqlc.InsertHostBatteryParams{
				HostID:           hostID,
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
			}); err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *Store) ReplaceCertificates(ctx context.Context, hostID int64, certificates []HostCertificate) error {
	return s.db.WithTx(ctx, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		if err := q.DeleteHostCertificates(ctx, sqlc.DeleteHostCertificatesParams{HostID: hostID}); err != nil {
			return err
		}
		for _, certificate := range certificates {
			if certificate.SHA1 == "" {
				continue
			}
			if err := q.InsertHostCertificate(ctx, sqlc.InsertHostCertificateParams{
				HostID:                    hostID,
				Sha1:                      certificate.SHA1,
				CommonName:                certificate.CommonName,
				SubjectCountry:            certificate.Subject.Country,
				SubjectOrganization:       certificate.Subject.Organization,
				SubjectOrganizationalUnit: certificate.Subject.OrganizationalUnit,
				SubjectCommonName:         certificate.Subject.CommonName,
				IssuerCountry:             certificate.Issuer.Country,
				IssuerOrganization:        certificate.Issuer.Organization,
				IssuerOrganizationalUnit:  certificate.Issuer.OrganizationalUnit,
				IssuerCommonName:          certificate.Issuer.CommonName,
				KeyAlgorithm:              certificate.KeyAlgorithm,
				KeyStrength:               certificate.KeyStrength,
				KeyUsage:                  certificate.KeyUsage,
				SigningAlgorithm:          certificate.SigningAlgorithm,
				NotValidAfter:             certificate.NotValidAfter,
				NotValidBefore:            certificate.NotValidBefore,
				Serial:                    certificate.Serial,
				CertificateAuthority:      certificate.CertificateAuthority,
				Source:                    certificate.Source,
				Username:                  certificate.Username,
				Path:                      certificate.Path,
			}); err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *Store) ListUsers(ctx context.Context, hostID int64) ([]HostUser, error) {
	rows, err := s.q.ListHostUsers(ctx, sqlc.ListHostUsersParams{HostID: hostID})
	if err != nil {
		return nil, err
	}
	users := make([]HostUser, len(rows))
	for i, row := range rows {
		users[i] = hostUserFromSQLC(row)
	}
	return users, nil
}

func (s *Store) ListBatteries(ctx context.Context, hostID int64) ([]HostBattery, error) {
	rows, err := s.q.ListHostBatteries(ctx, sqlc.ListHostBatteriesParams{HostID: hostID})
	if err != nil {
		return nil, err
	}
	batteries := make([]HostBattery, len(rows))
	for i, row := range rows {
		batteries[i] = hostBatteryFromSQLC(row)
	}
	return batteries, nil
}

func (s *Store) ListCertificates(ctx context.Context, hostID int64) ([]HostCertificate, error) {
	rows, err := s.q.ListHostCertificates(ctx, sqlc.ListHostCertificatesParams{HostID: hostID})
	if err != nil {
		return nil, err
	}
	certificates := make([]HostCertificate, len(rows))
	for i, row := range rows {
		certificates[i] = hostCertificateFromSQLC(row)
	}
	return certificates, nil
}

func (s *Store) MarkInventoryFresh(ctx context.Context, hostID int64, inventoryQueryHash string) error {
	return s.q.MarkHostInventoryFresh(ctx, sqlc.MarkHostInventoryFreshParams{
		ID:                 hostID,
		InventoryQueryHash: inventoryQueryHash,
	})
}

func (s *Store) attachUserAffinity(ctx context.Context, hosts []Host) error {
	if len(hosts) == 0 {
		return nil
	}
	hostIDs := make([]int64, len(hosts))
	for i := range hosts {
		hostIDs[i] = hosts[i].ID
	}
	affinity, err := s.loadUserAffinity(ctx, hostIDs)
	if err != nil {
		return err
	}
	for i := range hosts {
		hosts[i].UserAffinity = affinity[hosts[i].ID]
	}
	return nil
}

func hostListQuery(params HostListParams, where string, args []any) dbutil.ListQuery {
	return dbutil.ListQuery{
		SelectSQL: "SELECT * FROM hosts",
		WhereSQL:  where,
		Args:      args,
		OrderKeys: map[string]dbutil.OrderExpr{
			"display_name":                        {SQL: "lower(display_name)"},
			"hardware.serial":                     {SQL: "lower(hardware_serial)"},
			"hardware.model_identifier":           {SQL: "lower(hardware_model_identifier)"},
			"hardware.uuid":                       {SQL: "hardware_uuid"},
			"os.version":                          {SQL: "lower(os_version)"},
			"agents.osquery.version":              {SQL: "lower(osquery_version)"},
			"timestamps.last_seen_at":             {SQL: "last_seen_at", NullOrder: dbutil.NullsLast},
			"timestamps.last_restarted_at":        {SQL: "last_restarted_at", NullOrder: dbutil.NullsLast},
			"storage.boot_volume.available_bytes": {SQL: "boot_volume_available_bytes", NullOrder: dbutil.NullsLast},
			"hardware.memory_bytes":               {SQL: "memory_bytes"},
			"network.primary_ip":                  {SQL: "primary_ip", NullOrder: dbutil.NullsLast},
			"network.last_remote_ip":              {SQL: "last_remote_ip", NullOrder: dbutil.NullsLast},
		},
		DefaultOrder: []dbutil.OrderExpr{{SQL: "lower(display_name)"}, {SQL: "id"}},
		Params:       params.ListParams,
	}
}

func hostListWhere(params HostListParams) (string, []any, error) {
	var where dbutil.WhereBuilder
	if params.Q != "" {
		search := where.Arg("%" + params.Q + "%")
		where.Add(`(
			display_name ILIKE ` + search + `
			OR hostname ILIKE ` + search + `
				OR computer_name ILIKE ` + search + `
				OR hardware_serial ILIKE ` + search + `
				OR hardware_uuid ILIKE ` + search + `
				OR hardware_model_identifier ILIKE ` + search + `
				OR os_version ILIKE ` + search + `
				OR EXISTS (
					SELECT 1 FROM host_user_affinity_mappings he
					WHERE he.host_id = hosts.id AND he.email ILIKE ` + search + `
				)
			)`)
	}
	if len(params.IDs) > 0 {
		ids := where.Arg(params.IDs)
		where.Add("id = ANY(" + ids + "::bigint[])")
	}
	switch params.Status {
	case "":
	case "online":
		where.Add("last_seen_at >= now() - interval '5 minutes'")
	case "offline":
		where.Add("(last_seen_at IS NULL OR last_seen_at < now() - interval '5 minutes')")
	default:
		return "", nil, fmt.Errorf("%w: unknown status %q", dbutil.ErrInvalidInput, params.Status)
	}
	if params.LabelID != 0 {
		labelID := where.Arg(params.LabelID)
		where.Add(`EXISTS (
			SELECT 1 FROM label_membership lm
			WHERE lm.host_id = hosts.id AND lm.label_id = ` + labelID + `::bigint
		)`)
	}
	if params.SoftwareID != 0 {
		softwareID := where.Arg(params.SoftwareID)
		where.Add(`EXISTS (
			SELECT 1 FROM host_software hs
			WHERE hs.host_id = hosts.id AND hs.software_id = ` + softwareID + `::bigint
		)`)
	}
	if params.SoftwareTitleID != 0 {
		softwareTitleID := where.Arg(params.SoftwareTitleID)
		where.Add(`EXISTS (
			SELECT 1
			FROM host_software hs
			JOIN software s ON s.id = hs.software_id
			WHERE hs.host_id = hosts.id AND s.title_id = ` + softwareTitleID + `::bigint
		)`)
	}
	whereSQL, args := where.Build()
	return whereSQL, args, nil
}

// inventoryDisplayName persists the canonical host label exposed by the API.
func inventoryDisplayName(hardwareUUID, hostname, computerName string) string {
	if computerName != "" {
		return computerName
	}
	if hostname != "" {
		return hostname
	}
	return hardwareUUID
}

func statusFromLastSeen(lastSeen *time.Time, now time.Time) HostStatus {
	if lastSeen == nil || lastSeen.Before(now.Add(-hostOnlineWindow)) {
		return HostStatusOffline
	}
	return HostStatusOnline
}

func (s *Store) loadUserAffinity(ctx context.Context, hostIDs []int64) (map[int64]HostUserAffinity, error) {
	affinity := make(map[int64]HostUserAffinity, len(hostIDs))
	for _, hostID := range hostIDs {
		affinity[hostID] = HostUserAffinity{Mappings: []HostUserAffinityMapping{}}
	}
	if len(hostIDs) == 0 {
		return affinity, nil
	}

	mappingRows, err := s.q.ListHostUserAffinityMappingsForHosts(ctx, sqlc.ListHostUserAffinityMappingsForHostsParams{
		HostIds: hostIDs,
	})
	if err != nil {
		return nil, err
	}
	grouped := groupHostUserAffinityMappings(mappingRows)
	primaryRows, err := s.q.ListHostUserAffinityPrimaries(ctx, sqlc.ListHostUserAffinityPrimariesParams{
		HostIds: hostIDs,
	})
	if err != nil {
		return nil, err
	}
	for _, row := range primaryRows {
		hostAffinity := affinity[row.HostID]
		hostAffinity.Primary = &HostUserAffinityPrimary{
			Email:      row.Email,
			Username:   row.Username,
			Name:       row.Name,
			Department: row.Department,
			Groups:     row.Groups,
			Source:     UserAffinitySource(row.Source),
		}
		affinity[row.HostID] = hostAffinity
	}
	for hostID, mappings := range grouped {
		hostAffinity := affinity[hostID]
		hostAffinity.Mappings = mappings
		affinity[hostID] = hostAffinity
	}
	return affinity, nil
}

func hostFromSQLC(s sqlc.Host) Host {
	return Host{
		ID:           s.ID,
		DisplayName:  s.DisplayName,
		Status:       statusFromLastSeen(s.LastSeenAt, time.Now()),
		Hostname:     s.Hostname,
		ComputerName: s.ComputerName,
		Enrollment: HostEnrollment{
			Agent:      s.EnrollmentAgent,
			EnrolledAt: s.EnrolledAt,
		},
		Hardware: HostHardware{
			UUID:            s.HardwareUUID,
			Serial:          s.HardwareSerial,
			Vendor:          s.HardwareVendor,
			ModelIdentifier: s.HardwareModelIdentifier,
			MemoryBytes:     s.MemoryBytes,
			CPU: HostCPU{
				Architecture:  s.CPUType,
				Subtype:       s.CPUSubtype,
				Brand:         s.CPUBrand,
				LogicalCores:  s.CPULogicalCores,
				PhysicalCores: s.CPUPhysicalCores,
			},
		},
		OS: HostOS{
			Platform:      s.OSPlatform,
			Name:          s.OSName,
			Version:       s.OSVersion,
			Build:         s.OSBuild,
			KernelVersion: s.OSKernelVersion,
		},
		Storage: HostStorage{
			BootVolume: HostBootVolume{
				AvailableBytes: s.BootVolumeAvailableBytes,
				TotalBytes:     s.BootVolumeTotalBytes,
			},
		},
		Network: HostNetwork{
			PrimaryIP:    s.PrimaryIP,
			PrimaryMAC:   s.PrimaryMAC,
			LastRemoteIP: s.LastRemoteIP,
		},
		Agents: HostAgents{
			Osquery: HostOsqueryAgent{
				Version:                    s.OsqueryVersion,
				DistributedIntervalSeconds: s.OsqueryDistributedIntervalSeconds,
				ConfigRefreshSeconds:       s.OsqueryConfigRefreshSeconds,
			},
			Orbit: HostOrbitAgent{Version: s.OrbitVersion},
		},
		UserAffinity: HostUserAffinity{Mappings: []HostUserAffinityMapping{}},
		Timestamps: HostTimestamps{
			CreatedAt:          s.CreatedAt,
			UpdatedAt:          s.UpdatedAt,
			LastSeenAt:         s.LastSeenAt,
			InventoryUpdatedAt: s.InventoryUpdatedAt,
			LastRestartedAt:    s.LastRestartedAt,
		},
		OrbitNodeKey:       s.OrbitNodeKey,
		OsqueryNodeKey:     s.OsqueryNodeKey,
		InventoryQueryHash: s.InventoryQueryHash,
	}
}

func hostUserFromSQLC(s sqlc.HostUser) HostUser {
	return HostUser{
		ID:          s.ID,
		HostID:      s.HostID,
		UID:         s.UID,
		Username:    s.Username,
		Type:        s.Type,
		Description: s.Description,
		Directory:   s.Directory,
		Shell:       s.Shell,
		CreatedAt:   s.CreatedAt,
		UpdatedAt:   s.UpdatedAt,
	}
}

func hostBatteryFromSQLC(s sqlc.HostBattery) HostBattery {
	return HostBattery{
		ID:               s.ID,
		HostID:           s.HostID,
		SerialNumber:     s.SerialNumber,
		Manufacturer:     s.Manufacturer,
		Model:            s.Model,
		Chemistry:        s.Chemistry,
		CycleCount:       s.CycleCount,
		Health:           s.Health,
		DesignedCapacity: s.DesignedCapacity,
		MaxCapacity:      s.MaxCapacity,
		CurrentCapacity:  s.CurrentCapacity,
		PercentRemaining: s.PercentRemaining,
		CreatedAt:        s.CreatedAt,
		UpdatedAt:        s.UpdatedAt,
	}
}

func hostCertificateFromSQLC(s sqlc.HostCertificate) HostCertificate {
	return HostCertificate{
		ID:         s.ID,
		HostID:     s.HostID,
		SHA1:       s.Sha1,
		CommonName: s.CommonName,
		Subject: CertificateName{
			Country:            s.SubjectCountry,
			Organization:       s.SubjectOrganization,
			OrganizationalUnit: s.SubjectOrganizationalUnit,
			CommonName:         s.SubjectCommonName,
		},
		Issuer: CertificateName{
			Country:            s.IssuerCountry,
			Organization:       s.IssuerOrganization,
			OrganizationalUnit: s.IssuerOrganizationalUnit,
			CommonName:         s.IssuerCommonName,
		},
		KeyAlgorithm:         s.KeyAlgorithm,
		KeyStrength:          s.KeyStrength,
		KeyUsage:             s.KeyUsage,
		SigningAlgorithm:     s.SigningAlgorithm,
		NotValidAfter:        s.NotValidAfter,
		NotValidBefore:       s.NotValidBefore,
		Serial:               s.Serial,
		CertificateAuthority: s.CertificateAuthority,
		Source:               s.Source,
		Username:             s.Username,
		Path:                 s.Path,
		CreatedAt:            s.CreatedAt,
		UpdatedAt:            s.UpdatedAt,
	}
}
