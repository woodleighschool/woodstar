package models

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodstar/internal/database"
	"github.com/woodleighschool/woodstar/internal/database/sqlc"
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
	q  *sqlc.Queries
}

// NewSoftwareStore returns a software store backed by db.
func NewSoftwareStore(db *database.DB) *SoftwareStore {
	return &SoftwareStore{db: db, q: db.Queries()}
}

// ReplaceHostSoftware replaces a host's software snapshot in one transaction.
func (s *SoftwareStore) ReplaceHostSoftware(ctx context.Context, hostID int64, entries []HostSoftwareEntry) error {
	return s.db.WithTx(ctx, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		if err := q.DeleteHostSoftware(ctx, sqlc.DeleteHostSoftwareParams{HostID: hostID}); err != nil {
			return err
		}

		for _, entry := range entries {
			if entry.Name == "" {
				continue
			}
			softwareID, err := softwareIDFor(ctx, q, entry)
			if err != nil {
				return err
			}
			if err := q.UpsertHostSoftware(ctx, sqlc.UpsertHostSoftwareParams{
				HostID:       hostID,
				SoftwareID:   softwareID,
				LastOpenedAt: entry.LastOpenedAt,
			}); err != nil {
				return err
			}
		}
		return nil
	})
}

func softwareIDFor(ctx context.Context, q *sqlc.Queries, entry HostSoftwareEntry) (int64, error) {
	return q.UpsertSoftwareTitle(ctx, sqlc.UpsertSoftwareTitleParams{
		Name:             entry.Name,
		Version:          entry.Version,
		Source:           entry.Source,
		BundleIdentifier: entry.BundleIdentifier,
	})
}

// ListForHost returns software installed on a host ordered by title.
func (s *SoftwareStore) ListForHost(ctx context.Context, hostID int64) ([]HostSoftwareRow, error) {
	rows, err := s.q.ListSoftwareForHost(ctx, sqlc.ListSoftwareForHostParams{HostID: hostID})
	if err != nil {
		return nil, err
	}

	software := make([]HostSoftwareRow, 0, len(rows))
	for _, row := range rows {
		software = append(software, hostSoftwareFromRecord(row))
	}
	return software, nil
}

// ListTitlesWithHostCount returns global software titles with installed host counts.
func (s *SoftwareStore) ListTitlesWithHostCount(ctx context.Context) ([]SoftwareTitle, error) {
	rows, err := s.q.ListSoftwareTitlesWithHostCount(ctx)
	if err != nil {
		return nil, err
	}

	titles := make([]SoftwareTitle, 0, len(rows))
	for _, row := range rows {
		titles = append(titles, softwareTitleFromRecord(row))
	}
	return titles, nil
}

func hostSoftwareFromRecord(row sqlc.ListSoftwareForHostRow) HostSoftwareRow {
	return HostSoftwareRow{
		ID:               row.ID,
		Name:             row.Name,
		Version:          row.Version,
		Source:           row.Source,
		BundleIdentifier: row.BundleIdentifier,
		LastSeenAt:       row.LastSeenAt,
		LastOpenedAt:     row.LastOpenedAt,
	}
}

func softwareTitleFromRecord(row sqlc.ListSoftwareTitlesWithHostCountRow) SoftwareTitle {
	return SoftwareTitle{
		ID:               row.ID,
		Name:             row.Name,
		Version:          row.Version,
		Source:           row.Source,
		BundleIdentifier: row.BundleIdentifier,
		HostCount:        int(row.HostCount),
		CreatedAt:        row.CreatedAt,
	}
}
