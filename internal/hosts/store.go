package hosts

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodstar/internal/database"
	"github.com/woodleighschool/woodstar/internal/database/sqlc"
	"github.com/woodleighschool/woodstar/internal/dbutil"
)

// Store persists hosts.
type Store struct {
	db *database.DB
	q  *sqlc.Queries
}

func NewStore(db *database.DB) *Store {
	return &Store{db: db, q: db.Queries()}
}

// UpsertOnOrbitEnroll creates or refreshes a host from Orbit enroll.
func (s *Store) UpsertOnOrbitEnroll(ctx context.Context, update DetailUpdate) (*Host, error) {
	row, err := s.q.UpsertHostOnOrbitEnroll(ctx, sqlc.UpsertHostOnOrbitEnrollParams{
		HardwareUUID:   update.HardwareUUID,
		DisplayName:    displayName(update.HardwareUUID, update.Hostname, update.ComputerName),
		Hostname:       update.Hostname,
		ComputerName:   update.ComputerName,
		HardwareSerial: update.HardwareSerial,
		HardwareModel:  update.HardwareModel,
		OrbitNodeKey:   update.OrbitNodeKey,
	})
	if err != nil {
		return nil, err
	}
	if err := s.q.AddHostToAllHostsLabel(ctx, sqlc.AddHostToAllHostsLabelParams{HostID: row.ID}); err != nil {
		return nil, err
	}
	return new(hostFromSQLC(row)), nil
}

// UpsertOnOsqueryEnroll creates or refreshes a host from osquery enroll.
func (s *Store) UpsertOnOsqueryEnroll(ctx context.Context, update DetailUpdate) (*Host, error) {
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

func (s *Store) List(ctx context.Context, params ListParams) ([]Host, int, error) {
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
	if err := s.attachDeviceMappings(ctx, hosts); err != nil {
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

func (s *Store) ApplyDetail(ctx context.Context, hostID int64, update DetailUpdate) error {
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

func (s *Store) MarkDetailFresh(ctx context.Context, hostID int64, detailQueryHash string) error {
	return s.q.MarkHostDetailFresh(ctx, sqlc.MarkHostDetailFreshParams{ID: hostID, DetailQueryHash: detailQueryHash})
}

func (s *Store) attachDeviceMappings(ctx context.Context, hosts []Host) error {
	if len(hosts) == 0 {
		return nil
	}
	hostIDs := make([]int64, len(hosts))
	for i := range hosts {
		hostIDs[i] = hosts[i].ID
	}
	rows, err := s.q.ListHostDeviceMappingsForHosts(ctx, sqlc.ListHostDeviceMappingsForHostsParams{
		HostIds: hostIDs,
	})
	if err != nil {
		return err
	}
	grouped := groupHostDeviceMappings(rows, len(hostIDs))
	for i := range hosts {
		hosts[i].DeviceMappings = grouped[hosts[i].ID]
	}
	return nil
}

func hostListSQLWithWhere(params ListParams, where string, args []any) (string, []any, error) {
	return dbutil.ListQuery{
		SelectSQL: "SELECT * FROM hosts",
		WhereSQL:  where,
		Args:      args,
		OrderKeys: map[string]dbutil.OrderExpr{
			"display_name":               {SQL: "lower(display_name)"},
			"hardware_serial":            {SQL: "lower(hardware_serial)"},
			"hardware_model":             {SQL: "lower(hardware_model)"},
			"hardware_uuid":              {SQL: "hardware_uuid"},
			"os_version":                 {SQL: "lower(os_version)"},
			"osquery_version":            {SQL: "lower(osquery_version)"},
			"last_seen_at":               {SQL: "last_seen_at", NullOrder: dbutil.NullsLast},
			"last_restarted_at":          {SQL: "last_restarted_at", NullOrder: dbutil.NullsLast},
			"disk_space_available_bytes": {SQL: "disk_space_available_bytes", NullOrder: dbutil.NullsLast},
			"physical_memory":            {SQL: "physical_memory"},
			"primary_ip":                 {SQL: "primary_ip", NullOrder: dbutil.NullsLast},
			"public_ip":                  {SQL: "public_ip", NullOrder: dbutil.NullsLast},
		},
		DefaultOrder: []dbutil.OrderExpr{{SQL: "lower(display_name)"}, {SQL: "id"}},
		Params:       params.ListParams,
	}.Build()
}

func hostListWhere(params ListParams) (string, []any, error) {
	var where dbutil.WhereBuilder
	if params.Q != "" {
		search := where.Arg("%" + params.Q + "%")
		where.Add(`(
			display_name ILIKE ` + search + `
			OR hostname ILIKE ` + search + `
			OR computer_name ILIKE ` + search + `
			OR hardware_serial ILIKE ` + search + `
			OR hardware_uuid ILIKE ` + search + `
			OR hardware_model ILIKE ` + search + `
			OR os_version ILIKE ` + search + `
			OR EXISTS (
				SELECT 1 FROM host_emails he
				WHERE he.host_id = hosts.id AND he.email ILIKE ` + search + `
			)
		)`)
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
	if params.LabelID > 0 {
		labelID := where.Arg(params.LabelID)
		where.Add(`EXISTS (
			SELECT 1 FROM label_membership lm
			WHERE lm.host_id = hosts.id AND lm.label_id = ` + labelID + `::bigint
		)`)
	}
	if params.SoftwareID > 0 {
		softwareID := where.Arg(params.SoftwareID)
		where.Add(`EXISTS (
			SELECT 1 FROM host_software hs
			WHERE hs.host_id = hosts.id AND hs.software_id = ` + softwareID + `::bigint
		)`)
	}
	if params.SoftwareTitleID > 0 {
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

// displayName picks the nicest name we have.
func displayName(hardwareUUID, hostname, computerName string) string {
	if computerName != "" {
		return computerName
	}
	if hostname != "" {
		return hostname
	}
	return hardwareUUID
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
