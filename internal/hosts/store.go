package hosts

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodstar/internal/database"
	"github.com/woodleighschool/woodstar/internal/database/sqlc"
	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/platform"
)

// Store persists Orbit-managed Macs.
type Store struct {
	db *database.DB
	q  *sqlc.Queries
}

// NewStore returns a host store backed by db.
func NewStore(db *database.DB) *Store {
	return &Store{db: db, q: db.Queries()}
}

// UpsertOnOrbitEnroll inserts a new host or refreshes an existing one keyed by
// hardware UUID. Re-enrollment overwrites the orbit node key so prior keys
// stop authenticating. Newly-enrolled hosts are added to the All Hosts label.
func (s *Store) UpsertOnOrbitEnroll(ctx context.Context, params EnrollParams) (*Host, error) {
	params, err := cleanOrbitEnrollParams(params)
	if err != nil {
		return nil, err
	}
	row, err := s.q.UpsertHostOnOrbitEnroll(ctx, sqlc.UpsertHostOnOrbitEnrollParams{
		HardwareUUID:   params.HardwareUUID,
		DisplayName:    displayName(params.HardwareUUID, params.Hostname, params.ComputerName),
		Hostname:       params.Hostname,
		ComputerName:   params.ComputerName,
		HardwareSerial: params.HardwareSerial,
		HardwareModel:  params.HardwareModel,
		Platform:       params.Platform,
		PlatformLike:   params.PlatformLike,
		OrbitNodeKey:   params.OrbitNodeKey,
	})
	if err != nil {
		return nil, err
	}
	if err := s.q.AddHostToAllHostsLabel(ctx, sqlc.AddHostToAllHostsLabelParams{HostID: row.ID}); err != nil {
		return nil, err
	}
	return new(hostFromSQLC(row)), nil
}

// UpsertOnOsqueryEnroll refreshes the osquery node key and host inventory.
// Newly-enrolled hosts are added to the All Hosts label.
func (s *Store) UpsertOnOsqueryEnroll(ctx context.Context, update HostDetailUpdate) (*Host, error) {
	update, err := cleanHostDetailUpdate(update)
	if err != nil {
		return nil, err
	}
	row, err := s.q.UpsertHostOnOsqueryEnroll(ctx, sqlc.UpsertHostOnOsqueryEnrollParams{
		HardwareUUID:     update.HardwareUUID,
		DisplayName:      displayName(update.HardwareUUID, update.Hostname, update.ComputerName),
		Hostname:         update.Hostname,
		ComputerName:     update.ComputerName,
		HardwareSerial:   update.HardwareSerial,
		HardwareModel:    update.HardwareModel,
		HardwareVersion:  update.HardwareVersion,
		OSName:           update.OSName,
		OSVersion:        update.OSVersion,
		OSBuild:          update.OSBuild,
		Platform:         update.Platform,
		PlatformLike:     update.PlatformLike,
		OsqueryVersion:   update.OsqueryVersion,
		OsqueryNodeKey:   update.OsqueryNodeKey,
		OrbitVersion:     update.OrbitVersion,
		CPUBrand:         update.CPUBrand,
		CPULogicalCores:  update.CPULogicalCores,
		CPUPhysicalCores: update.CPUPhysicalCores,
		PhysicalMemory:   update.PhysicalMemory,
		HardwareVendor:   update.HardwareVendor,
		KernelVersion:    update.KernelVersion,
	})
	if err != nil {
		return nil, err
	}
	if err := s.q.AddHostToAllHostsLabel(ctx, sqlc.AddHostToAllHostsLabelParams{HostID: row.ID}); err != nil {
		return nil, err
	}
	return new(hostFromSQLC(row)), nil
}

// List returns active hosts and the total count matching params.
func (s *Store) List(ctx context.Context, params HostListParams) ([]Host, int, error) {
	params = cleanHostListParams(params)
	where, args, err := hostListWhere(params)
	if err != nil {
		return nil, 0, err
	}
	var count int
	if err := s.db.Pool().
		QueryRow(ctx, "SELECT count(*)::integer FROM hosts "+where, args...).
		Scan(&count); err != nil {
		return nil, 0, err
	}
	query, args, err := hostListSQLWithWhere(params, where, args)
	if err != nil {
		return nil, 0, err
	}
	rows, err := s.db.Pool().Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	dbHosts, err := pgx.CollectRows(rows, pgx.RowToStructByName[sqlc.Host])
	if err != nil {
		return nil, 0, err
	}
	hosts := make([]Host, len(dbHosts))
	for i, row := range dbHosts {
		hosts[i] = hostFromSQLC(row)
	}
	return hosts, count, nil
}

// GetByID returns a single active host by database ID.
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

// Delete removes one host and cascades inventory, labels, check results, and report results.
func (s *Store) Delete(ctx context.Context, id int64) error {
	_, err := s.q.DeleteHost(ctx, sqlc.DeleteHostParams{ID: id})
	if errors.Is(err, pgx.ErrNoRows) {
		return dbutil.ErrNotFound
	}
	return err
}

// DeleteMany removes multiple hosts. Missing IDs are ignored so repeated bulk actions are idempotent.
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

// GetByOrbitNodeKey returns an active host and refreshes last_seen_at.
func (s *Store) GetByOrbitNodeKey(ctx context.Context, nodeKey string) (*Host, error) {
	nodeKey = strings.TrimSpace(nodeKey)
	if nodeKey == "" {
		return nil, dbutil.ErrNotFound
	}
	row, err := s.q.TouchHostByOrbitNodeKey(ctx, sqlc.TouchHostByOrbitNodeKeyParams{OrbitNodeKey: nodeKey})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, dbutil.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return new(hostFromSQLC(row)), nil
}

// GetByOsqueryNodeKey returns an active host and refreshes last_seen_at.
func (s *Store) GetByOsqueryNodeKey(ctx context.Context, nodeKey string) (*Host, error) {
	nodeKey = strings.TrimSpace(nodeKey)
	if nodeKey == "" {
		return nil, dbutil.ErrNotFound
	}
	row, err := s.q.TouchHostByOsqueryNodeKey(ctx, sqlc.TouchHostByOsqueryNodeKeyParams{OsqueryNodeKey: nodeKey})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, dbutil.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return new(hostFromSQLC(row)), nil
}

// ApplyDetail updates the host fields reported by successful detail queries.
func (s *Store) ApplyDetail(ctx context.Context, hostID int64, update HostDetailUpdate) error {
	return s.q.ApplyHostDetail(ctx, sqlc.ApplyHostDetailParams{
		ID:                      hostID,
		Hostname:                update.Hostname,
		ComputerName:            update.ComputerName,
		HardwareSerial:          update.HardwareSerial,
		HardwareModel:           update.HardwareModel,
		HardwareVersion:         update.HardwareVersion,
		OSName:                  update.OSName,
		OSVersion:               update.OSVersion,
		OSBuild:                 update.OSBuild,
		Platform:                update.Platform,
		PlatformLike:            update.PlatformLike,
		OsqueryVersion:          update.OsqueryVersion,
		OrbitVersion:            update.OrbitVersion,
		CPUType:                 update.CPUType,
		CPUSubtype:              update.CPUSubtype,
		CPUBrand:                update.CPUBrand,
		CPULogicalCores:         int32(update.CPULogicalCores),
		CPUPhysicalCores:        int32(update.CPUPhysicalCores),
		PhysicalMemory:          update.PhysicalMemory,
		HardwareVendor:          update.HardwareVendor,
		KernelVersion:           update.KernelVersion,
		UptimeSeconds:           update.UptimeSeconds,
		LastRestartedAt:         update.LastRestartedAt,
		DiskSpaceAvailableBytes: update.DiskSpaceAvailableBytes,
		DiskSpaceTotalBytes:     update.DiskSpaceTotalBytes,
		PublicIP:                update.PublicIP,
		PrimaryIP:               update.PrimaryIP,
		PrimaryMAC:              update.PrimaryMAC,
		DistributedInterval:     update.DistributedInterval,
		ConfigTLSRefresh:        update.ConfigTLSRefresh,
	})
}

// ReplaceUsers replaces the local user inventory for hostID.
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

// ReplaceBatteries replaces the battery inventory for hostID.
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

// ReplaceCertificates replaces the certificate inventory for hostID.
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

// ListUsers returns local users reported for hostID.
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

// ListBatteries returns batteries reported for hostID.
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

// ListCertificates returns certificates reported for hostID.
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

// MarkDetailFresh records that all built-in detail queries completed.
func (s *Store) MarkDetailFresh(ctx context.Context, hostID int64, detailQueryHash string) error {
	return s.q.MarkHostDetailFresh(ctx, sqlc.MarkHostDetailFreshParams{ID: hostID, DetailQueryHash: detailQueryHash})
}

func cleanHostListParams(params HostListParams) HostListParams {
	params.ListParams = dbutil.CleanListParams(params.ListParams)
	params.Status = strings.TrimSpace(params.Status)
	params.Platform = platform.CleanPlatform(params.Platform)
	return params
}

func hostListSQLWithWhere(params HostListParams, where string, args []any) (string, []any, error) {
	return dbutil.ListQuery{
		SelectSQL: "SELECT * FROM hosts",
		WhereSQL:  where,
		Args:      args,
		OrderKeys: map[string]dbutil.OrderExpr{
			"display_name":               {SQL: "lower(display_name)"},
			"platform":                   {SQL: "lower(platform)"},
			"hardware_serial":            {SQL: "lower(hardware_serial)"},
			"hardware_model":             {SQL: "lower(hardware_model)"},
			"hardware_uuid":              {SQL: "hardware_uuid"},
			"os_version":                 {SQL: "lower(os_version)"},
			"osquery_version":            {SQL: "lower(osquery_version)"},
			"last_seen_at":               {SQL: "last_seen_at", NullsLast: true},
			"last_restarted_at":          {SQL: "last_restarted_at", NullsLast: true},
			"disk_space_available_bytes": {SQL: "disk_space_available_bytes", NullsLast: true},
			"physical_memory":            {SQL: "physical_memory"},
			"primary_ip":                 {SQL: "primary_ip", NullsLast: true},
			"public_ip":                  {SQL: "public_ip", NullsLast: true},
		},
		DefaultOrder: []dbutil.OrderExpr{{SQL: "lower(display_name)"}, {SQL: "id"}},
		Params:       params.ListParams,
	}.Build()
}

func hostListWhere(params HostListParams) (string, []any, error) {
	clauses := []string{"deleted_at IS NULL"}
	args := make([]any, 0)
	if params.Q != "" {
		args = append(args, "%"+params.Q+"%")
		placeholder := fmt.Sprintf("$%d", len(args))
		clauses = append(clauses, `(
			display_name ILIKE `+placeholder+`
			OR hostname ILIKE `+placeholder+`
			OR computer_name ILIKE `+placeholder+`
			OR hardware_serial ILIKE `+placeholder+`
			OR hardware_uuid ILIKE `+placeholder+`
			OR hardware_model ILIKE `+placeholder+`
			OR os_version ILIKE `+placeholder+`
			OR EXISTS (
				SELECT 1 FROM host_emails he
				WHERE he.host_id = hosts.id AND he.email ILIKE `+placeholder+`
			)
		)`)
	}
	if params.Platform != "" {
		args = append(args, params.Platform)
		placeholder := fmt.Sprintf("$%d", len(args))
		clauses = append(clauses, `(
			platform = `+placeholder+`
			OR (`+placeholder+` = 'darwin' AND platform IN ('darwin', 'macos'))
			OR (`+placeholder+` = 'linux' AND platform <> '' AND platform NOT IN ('darwin', 'macos', 'windows'))
		)`)
	}
	switch params.Status {
	case "":
	case "online":
		clauses = append(clauses, "last_seen_at >= now() - interval '5 minutes'")
	case "offline":
		clauses = append(clauses, "(last_seen_at IS NULL OR last_seen_at < now() - interval '5 minutes')")
	default:
		return "", nil, fmt.Errorf("%w: unknown status %q", dbutil.ErrInvalidInput, params.Status)
	}
	if params.LabelID > 0 {
		args = append(args, params.LabelID)
		clauses = append(clauses, fmt.Sprintf(`EXISTS (
			SELECT 1 FROM label_membership lm
			WHERE lm.host_id = hosts.id AND lm.label_id = $%d::bigint
		)`, len(args)))
	}
	if params.SoftwareID > 0 {
		args = append(args, params.SoftwareID)
		clauses = append(clauses, fmt.Sprintf(`EXISTS (
			SELECT 1 FROM host_software hs
			WHERE hs.host_id = hosts.id AND hs.software_id = $%d::bigint
		)`, len(args)))
	}
	if params.SoftwareTitleID > 0 {
		args = append(args, params.SoftwareTitleID)
		clauses = append(clauses, fmt.Sprintf(`EXISTS (
			SELECT 1
			FROM host_software hs
			JOIN software s ON s.id = hs.software_id
			WHERE hs.host_id = hosts.id AND s.title_id = $%d::bigint
		)`, len(args)))
	}
	return "WHERE " + strings.Join(clauses, " AND "), args, nil
}

func cleanOrbitEnrollParams(params EnrollParams) (EnrollParams, error) {
	params.HardwareUUID = strings.TrimSpace(params.HardwareUUID)
	params.OrbitNodeKey = strings.TrimSpace(params.OrbitNodeKey)
	if params.HardwareUUID == "" {
		return EnrollParams{}, errors.New("hardware_uuid is required")
	}
	if params.OrbitNodeKey == "" {
		return EnrollParams{}, errors.New("orbit_node_key is required")
	}
	return params, nil
}

func cleanHostDetailUpdate(update HostDetailUpdate) (HostDetailUpdate, error) {
	update.HardwareUUID = strings.TrimSpace(update.HardwareUUID)
	update.OsqueryNodeKey = strings.TrimSpace(update.OsqueryNodeKey)
	if update.HardwareUUID == "" {
		return HostDetailUpdate{}, errors.New("hardware_uuid is required")
	}
	if update.OsqueryNodeKey == "" {
		return HostDetailUpdate{}, errors.New("osquery_node_key is required")
	}
	return update, nil
}

// displayName picks the most user-friendly identifier from enrollment values.
func displayName(hardwareUUID, hostname, computerName string) string {
	if v := strings.TrimSpace(computerName); v != "" {
		return v
	}
	if v := strings.TrimSpace(hostname); v != "" {
		return v
	}
	return strings.TrimSpace(hardwareUUID)
}

func hostFromSQLC(s sqlc.Host) Host {
	return Host{
		ID:                      s.ID,
		HardwareUUID:            s.HardwareUUID,
		DisplayName:             s.DisplayName,
		Hostname:                s.Hostname,
		ComputerName:            s.ComputerName,
		HardwareSerial:          s.HardwareSerial,
		HardwareModel:           s.HardwareModel,
		HardwareVersion:         s.HardwareVersion,
		HardwareVendor:          s.HardwareVendor,
		OSName:                  s.OSName,
		OSVersion:               s.OSVersion,
		OSBuild:                 s.OSBuild,
		Platform:                s.Platform,
		PlatformLike:            s.PlatformLike,
		OsqueryVersion:          s.OsqueryVersion,
		OrbitVersion:            s.OrbitVersion,
		OrbitNodeKey:            s.OrbitNodeKey,
		OsqueryNodeKey:          s.OsqueryNodeKey,
		CPUType:                 s.CPUType,
		CPUSubtype:              s.CPUSubtype,
		CPUBrand:                s.CPUBrand,
		CPULogicalCores:         s.CPULogicalCores,
		CPUPhysicalCores:        s.CPUPhysicalCores,
		PhysicalMemory:          s.PhysicalMemory,
		KernelVersion:           s.KernelVersion,
		UptimeSeconds:           s.UptimeSeconds,
		LastRestartedAt:         s.LastRestartedAt,
		DiskSpaceAvailableBytes: s.DiskSpaceAvailableBytes,
		DiskSpaceTotalBytes:     s.DiskSpaceTotalBytes,
		PublicIP:                s.PublicIP,
		PrimaryIP:               s.PrimaryIP,
		PrimaryMAC:              s.PrimaryMAC,
		DistributedInterval:     s.DistributedInterval,
		ConfigTLSRefresh:        s.ConfigTLSRefresh,
		DetailQueryHash:         s.DetailQueryHash,
		EnrolledAt:              s.EnrolledAt,
		LastSeenAt:              s.LastSeenAt,
		DetailUpdatedAt:         s.DetailUpdatedAt,
		LabelUpdatedAt:          s.LabelUpdatedAt,
		SoftwareUpdatedAt:       s.SoftwareUpdatedAt,
		CreatedAt:               s.CreatedAt,
		UpdatedAt:               s.UpdatedAt,
		DeletedAt:               s.DeletedAt,
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
