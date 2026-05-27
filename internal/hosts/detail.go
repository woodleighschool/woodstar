package hosts

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodstar/internal/database/sqlc"
	"github.com/woodleighschool/woodstar/internal/labels"
)

func (s *Store) LoadDetail(ctx context.Context, host *Host) (*HostDetail, error) {
	batch := &pgx.Batch{}
	batch.Queue(hostDetailLabelsSQL, host.ID)
	batch.Queue(hostDetailUsersSQL, host.ID)
	batch.Queue(hostDetailBatteriesSQL, host.ID)
	batch.Queue(hostDetailCertificatesSQL, host.ID)
	batch.Queue(hostDetailDeviceMappingsSQL, host.ID)

	results := s.db.Pool().SendBatch(ctx, batch)
	defer results.Close()

	hostLabels, err := collectHostDetailLabels(results)
	if err != nil {
		return nil, err
	}
	hostUsers, err := collectHostDetailUsers(results)
	if err != nil {
		return nil, err
	}
	batteries, err := collectHostDetailBatteries(results)
	if err != nil {
		return nil, err
	}
	certificates, err := collectHostDetailCertificates(results)
	if err != nil {
		return nil, err
	}
	mappings, err := collectHostDetailDeviceMappings(results)
	if err != nil {
		return nil, err
	}

	detailHost := *host
	detailHost.DeviceMappings = mappings
	return &HostDetail{
		Host:         detailHost,
		Labels:       hostLabels,
		Users:        hostUsers,
		Batteries:    batteries,
		Certificates: certificates,
	}, nil
}

func collectHostDetailLabels(results pgx.BatchResults) ([]labels.Label, error) {
	rows, err := results.Query()
	if err != nil {
		return nil, err
	}
	records, err := pgx.CollectRows(rows, pgx.RowToStructByName[hostDetailLabelRecord])
	if err != nil {
		return nil, err
	}
	out := make([]labels.Label, len(records))
	for i, record := range records {
		out[i] = labelFromHostDetailRecord(record)
	}
	return out, nil
}

func collectHostDetailUsers(results pgx.BatchResults) ([]HostUser, error) {
	rows, err := results.Query()
	if err != nil {
		return nil, err
	}
	records, err := pgx.CollectRows(rows, pgx.RowToStructByName[sqlc.HostUser])
	if err != nil {
		return nil, err
	}
	out := make([]HostUser, len(records))
	for i, record := range records {
		out[i] = hostUserFromSQLC(record)
	}
	return out, nil
}

func collectHostDetailBatteries(results pgx.BatchResults) ([]HostBattery, error) {
	rows, err := results.Query()
	if err != nil {
		return nil, err
	}
	records, err := pgx.CollectRows(rows, pgx.RowToStructByName[sqlc.HostBattery])
	if err != nil {
		return nil, err
	}
	out := make([]HostBattery, len(records))
	for i, record := range records {
		out[i] = hostBatteryFromSQLC(record)
	}
	return out, nil
}

func collectHostDetailCertificates(results pgx.BatchResults) ([]HostCertificate, error) {
	rows, err := results.Query()
	if err != nil {
		return nil, err
	}
	records, err := pgx.CollectRows(rows, pgx.RowToStructByName[sqlc.HostCertificate])
	if err != nil {
		return nil, err
	}
	out := make([]HostCertificate, len(records))
	for i, record := range records {
		out[i] = hostCertificateFromSQLC(record)
	}
	return out, nil
}

func collectHostDetailDeviceMappings(results pgx.BatchResults) ([]HostDeviceMapping, error) {
	rows, err := results.Query()
	if err != nil {
		return nil, err
	}
	records, err := pgx.CollectRows(rows, pgx.RowToStructByName[sqlc.HostEmail])
	if err != nil {
		return nil, err
	}
	out := make([]HostDeviceMapping, len(records))
	for i, record := range records {
		out[i] = hostDeviceMappingFromSQLC(record)
	}
	return out, nil
}

type hostDetailLabelRecord struct {
	ID                  int64
	Name                string
	Description         string
	Query               *string
	LabelType           string
	LabelMembershipType string
	CreatedAt           time.Time
	UpdatedAt           time.Time
	HostsCount          int32
}

func labelFromHostDetailRecord(row hostDetailLabelRecord) labels.Label {
	return labels.Label{
		ID:                  row.ID,
		Name:                row.Name,
		Description:         row.Description,
		Query:               row.Query,
		LabelType:           labels.LabelType(row.LabelType),
		LabelMembershipType: row.LabelMembershipType,
		HostsCount:          int(row.HostsCount),
		CreatedAt:           row.CreatedAt,
		UpdatedAt:           row.UpdatedAt,
	}
}

const hostDetailLabelsSQL = `
SELECT
	l.id,
	l.name,
	l.description,
	l.query,
	l.label_type,
	l.label_membership_type,
	l.created_at,
	l.updated_at,
	count(lm_all.host_id)::integer AS hosts_count
FROM labels l
JOIN label_membership lm_host ON lm_host.label_id = l.id AND lm_host.host_id = $1
LEFT JOIN label_membership lm_all ON lm_all.label_id = l.id
GROUP BY l.id
ORDER BY lower(l.name), l.id`

const hostDetailUsersSQL = `
SELECT id, host_id, uid, username, type, description, directory, shell, created_at, updated_at
FROM host_users
WHERE host_id = $1
ORDER BY username, uid, id`

const hostDetailBatteriesSQL = `
SELECT id, host_id, serial_number, manufacturer, model, chemistry, cycle_count, health,
       designed_capacity, max_capacity, current_capacity, percent_remaining, created_at, updated_at
FROM host_batteries
WHERE host_id = $1
ORDER BY serial_number, id`

const hostDetailCertificatesSQL = `
SELECT id, host_id, sha1, common_name, subject_country, subject_organization,
       subject_organizational_unit, subject_common_name, issuer_country, issuer_organization,
       issuer_organizational_unit, issuer_common_name, key_algorithm, key_strength,
       key_usage, signing_algorithm, not_valid_after, not_valid_before, serial,
       certificate_authority, source, username, path, created_at, updated_at
FROM host_certificates
WHERE host_id = $1
ORDER BY common_name, sha1, id`

const hostDetailDeviceMappingsSQL = `
SELECT id, host_id, email, source, created_at, updated_at
FROM host_emails
WHERE host_id = $1
ORDER BY source`
