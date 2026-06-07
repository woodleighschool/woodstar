package inventory

import (
	"context"

	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodstar/internal/database/sqlc"
)

// ReplaceHostSoftware replaces a host's software snapshot in one transaction.
func (s *Store) ReplaceHostSoftware(ctx context.Context, hostID int64, entries []HostSoftwareEntry) error {
	return s.db.WithTx(ctx, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		if err := resetHostSoftware(ctx, q, hostID); err != nil {
			return err
		}
		for _, entry := range entries {
			if err := replaceHostSoftwareEntry(ctx, q, hostID, entry); err != nil {
				return err
			}
		}
		return nil
	})
}

func resetHostSoftware(ctx context.Context, q *sqlc.Queries, hostID int64) error {
	if err := q.DeleteHostSoftwarePaths(ctx, sqlc.DeleteHostSoftwarePathsParams{HostID: hostID}); err != nil {
		return err
	}
	return q.DeleteHostSoftware(ctx, sqlc.DeleteHostSoftwareParams{HostID: hostID})
}

func replaceHostSoftwareEntry(ctx context.Context, q *sqlc.Queries, hostID int64, entry HostSoftwareEntry) error {
	if entry.Name == "" || entry.Source == "" {
		return nil
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
	if entry.InstalledPath == "" {
		return nil
	}
	return q.InsertHostSoftwareInstalledPath(ctx, sqlc.InsertHostSoftwareInstalledPathParams{
		HostID:           hostID,
		SoftwareID:       softwareID,
		InstalledPath:    entry.InstalledPath,
		TeamIdentifier:   entry.TeamIdentifier,
		CdhashSha256:     entry.CDHashSHA256,
		ExecutableSha256: entry.ExecutableSHA256,
		ExecutablePath:   entry.ExecutablePath,
	})
}

func softwareIDFor(ctx context.Context, q *sqlc.Queries, entry HostSoftwareEntry) (int64, error) {
	titleID, err := softwareTitleIDFor(ctx, q, entry)
	if err != nil {
		return 0, err
	}
	row, err := q.UpsertSoftware(ctx, sqlc.UpsertSoftwareParams{
		TitleID:          titleID,
		Name:             entry.Name,
		Version:          entry.Version,
		Source:           entry.Source,
		BundleIdentifier: entry.BundleIdentifier,
		ExtensionID:      entry.ExtensionID,
		ExtensionFor:     entry.ExtensionFor,
		Vendor:           entry.Vendor,
		Arch:             entry.Arch,
		Release:          entry.Release,
	})
	if err != nil {
		return 0, err
	}
	return row.ID, nil
}

func softwareTitleIDFor(ctx context.Context, q *sqlc.Queries, entry HostSoftwareEntry) (int64, error) {
	params := softwareTitleParams{
		Name:             entry.Name,
		DisplayName:      entry.Name,
		Source:           entry.Source,
		ExtensionFor:     entry.ExtensionFor,
		BundleIdentifier: entry.BundleIdentifier,
		Vendor:           entry.Vendor,
	}
	if entry.BundleIdentifier != "" {
		return upsertSoftwareTitleByBundle(ctx, q, params)
	}
	return upsertSoftwareTitleByName(ctx, q, params)
}

type softwareTitleParams struct {
	Name             string
	DisplayName      string
	Source           string
	ExtensionFor     string
	BundleIdentifier string
	Vendor           string
}

func upsertSoftwareTitleByBundle(ctx context.Context, q *sqlc.Queries, params softwareTitleParams) (int64, error) {
	row, err := q.UpsertSoftwareTitleByBundle(ctx, sqlc.UpsertSoftwareTitleByBundleParams{
		Name:             params.Name,
		DisplayName:      params.DisplayName,
		Source:           params.Source,
		ExtensionFor:     params.ExtensionFor,
		BundleIdentifier: params.BundleIdentifier,
		Vendor:           params.Vendor,
	})
	if err != nil {
		return 0, err
	}
	return row.ID, nil
}

func upsertSoftwareTitleByName(ctx context.Context, q *sqlc.Queries, params softwareTitleParams) (int64, error) {
	row, err := q.UpsertSoftwareTitleByName(ctx, sqlc.UpsertSoftwareTitleByNameParams{
		Name:             params.Name,
		DisplayName:      params.DisplayName,
		Source:           params.Source,
		ExtensionFor:     params.ExtensionFor,
		BundleIdentifier: params.BundleIdentifier,
		Vendor:           params.Vendor,
	})
	if err != nil {
		return 0, err
	}
	return row.ID, nil
}
