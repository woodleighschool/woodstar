package models

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodstar/internal/database"
)

// HostSoftwareEntry is one installed software title reported by a host.
type HostSoftwareEntry struct {
	Name             string
	Version          string
	Source           string
	BundleIdentifier string
	InstalledPath    string
	LastOpenedAt     *time.Time
}

// HostSoftwareRow is software inventory projected for one host.
type HostSoftwareRow struct {
	ID               int64
	Name             string
	Version          string
	Source           string
	BundleIdentifier string
	LastSeenAt       time.Time
	LastOpenedAt     *time.Time
}

// SoftwareTitle is an aggregate software title row.
type SoftwareTitle struct {
	ID               int64
	Name             string
	Version          string
	Source           string
	BundleIdentifier string
	HostCount        int
	CreatedAt        time.Time
}

// SoftwareStore persists global software titles and host inventory joins.
type SoftwareStore struct {
	db *database.DB
}

// NewSoftwareStore returns a software store backed by db.
func NewSoftwareStore(db *database.DB) *SoftwareStore {
	return &SoftwareStore{db: db}
}

// ReplaceHostSoftware replaces a host's software snapshot in one transaction.
func (s *SoftwareStore) ReplaceHostSoftware(ctx context.Context, hostID int64, entries []HostSoftwareEntry) error {
	return s.db.WithTx(ctx, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `DELETE FROM host_software WHERE host_id = $1`, hostID); err != nil {
			return err
		}

		for _, entry := range entries {
			if entry.Name == "" {
				continue
			}
			var softwareID int64
			if err := tx.QueryRow(ctx, `
INSERT INTO software (name, version, source, bundle_identifier)
VALUES ($1, $2, $3, $4)
ON CONFLICT (name, version, source, bundle_identifier) DO UPDATE SET name = EXCLUDED.name
RETURNING id`,
				entry.Name,
				entry.Version,
				entry.Source,
				entry.BundleIdentifier,
			).Scan(&softwareID); err != nil {
				return err
			}
			if _, err := tx.Exec(ctx, `
INSERT INTO host_software (host_id, software_id, last_seen_at, last_opened_at)
VALUES ($1, $2, now(), $3)
ON CONFLICT (host_id, software_id) DO UPDATE SET
    last_seen_at = now(),
    last_opened_at = EXCLUDED.last_opened_at`,
				hostID,
				softwareID,
				entry.LastOpenedAt,
			); err != nil {
				return err
			}
		}
		return nil
	})
}

// ListForHost returns software installed on a host ordered by title.
func (s *SoftwareStore) ListForHost(ctx context.Context, hostID int64) ([]HostSoftwareRow, error) {
	rows, err := s.db.Query(ctx, `
SELECT software.id, software.name, software.version, software.source,
       software.bundle_identifier, host_software.last_seen_at, host_software.last_opened_at
FROM host_software
JOIN software ON software.id = host_software.software_id
WHERE host_software.host_id = $1
ORDER BY lower(software.name), software.version, software.source`, hostID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	software := make([]HostSoftwareRow, 0)
	for rows.Next() {
		var row HostSoftwareRow
		if err := rows.Scan(
			&row.ID,
			&row.Name,
			&row.Version,
			&row.Source,
			&row.BundleIdentifier,
			&row.LastSeenAt,
			&row.LastOpenedAt,
		); err != nil {
			return nil, err
		}
		software = append(software, row)
	}
	return software, rows.Err()
}

// ListTitlesWithHostCount returns global software titles with installed host counts.
func (s *SoftwareStore) ListTitlesWithHostCount(ctx context.Context) ([]SoftwareTitle, error) {
	rows, err := s.db.Query(ctx, `
SELECT software.id, software.name, software.version, software.source,
       software.bundle_identifier, count(host_software.host_id), software.created_at
FROM software
LEFT JOIN host_software ON host_software.software_id = software.id
GROUP BY software.id
ORDER BY count(host_software.host_id) DESC, lower(software.name), software.version`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	titles := make([]SoftwareTitle, 0)
	for rows.Next() {
		var title SoftwareTitle
		if err := rows.Scan(
			&title.ID,
			&title.Name,
			&title.Version,
			&title.Source,
			&title.BundleIdentifier,
			&title.HostCount,
			&title.CreatedAt,
		); err != nil {
			return nil, err
		}
		titles = append(titles, title)
	}
	return titles, rows.Err()
}
