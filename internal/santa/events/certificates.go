package events

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodstar/internal/santa/payloadhash"
)

type signingChainEntry struct {
	SHA256     string `json:"sha256"`
	CommonName string `json:"common_name,omitempty"`
	Org        string `json:"org,omitempty"`
	OU         string `json:"ou,omitempty"`
	ValidFrom  uint32 `json:"valid_from,omitempty"`
	ValidUntil uint32 `json:"valid_until,omitempty"`
}

func upsertSigningChain(ctx context.Context, tx pgx.Tx, executableID int64, chain []CertificateInput) error {
	entries := signingChainEntries(chain)
	if len(entries) == 0 {
		return nil
	}
	var chainID int64
	if err := tx.QueryRow(ctx, `
INSERT INTO santa_signing_chains (sha256)
VALUES ($1)
ON CONFLICT (sha256) DO UPDATE SET sha256 = EXCLUDED.sha256
RETURNING id`,
		signingChainHash(entries),
	).Scan(&chainID); err != nil {
		return err
	}
	for position, entry := range entries {
		certificateID, err := upsertCertificate(ctx, tx, entry)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `
INSERT INTO santa_signing_chain_entries (signing_chain_id, position, certificate_id)
VALUES (@signing_chain_id, @position, @certificate_id)
ON CONFLICT (signing_chain_id, position) DO UPDATE SET certificate_id = EXCLUDED.certificate_id`,
			pgx.NamedArgs{
				"signing_chain_id": chainID,
				"position":         int32(position),
				"certificate_id":   certificateID,
			}); err != nil {
			return err
		}
	}
	_, err := tx.Exec(ctx, `
INSERT INTO santa_executable_signing_chains (executable_id, signing_chain_id)
VALUES (@executable_id, @signing_chain_id)
ON CONFLICT DO NOTHING`,
		pgx.NamedArgs{
			"executable_id":    executableID,
			"signing_chain_id": chainID,
		})
	return err
}

func upsertCertificate(ctx context.Context, tx pgx.Tx, entry signingChainEntry) (int64, error) {
	var id int64
	err := tx.QueryRow(ctx, `
INSERT INTO santa_certificates (
	sha256,
	common_name,
	organization,
	organizational_unit,
	valid_from,
	valid_until,
	updated_at
)
VALUES (
	@sha256,
	@common_name,
	@organization,
	@organizational_unit,
	@valid_from::timestamptz,
	@valid_until::timestamptz,
	now()
)
ON CONFLICT (sha256) DO UPDATE SET
	common_name = EXCLUDED.common_name,
	organization = EXCLUDED.organization,
	organizational_unit = EXCLUDED.organizational_unit,
	valid_from = EXCLUDED.valid_from,
	valid_until = EXCLUDED.valid_until,
	updated_at = now()
RETURNING id`,
		pgx.NamedArgs{
			"sha256":              entry.SHA256,
			"common_name":         entry.CommonName,
			"organization":        entry.Org,
			"organizational_unit": entry.OU,
			"valid_from":          certificateTime(entry.ValidFrom),
			"valid_until":         certificateTime(entry.ValidUntil),
		}).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func signingChainEntries(chain []CertificateInput) []signingChainEntry {
	entries := make([]signingChainEntry, 0, len(chain))
	for _, cert := range chain {
		entry := signingChainEntry(cert)
		if entry.SHA256 == "" {
			continue
		}
		entries = append(entries, entry)
	}
	return entries
}

func signingChainOutputEntries(entries []signingChainEntry) []SigningChainEntry {
	out := make([]SigningChainEntry, 0, len(entries))
	for _, entry := range entries {
		out = append(out, SigningChainEntry{
			SHA256:             entry.SHA256,
			CommonName:         entry.CommonName,
			Organization:       entry.Org,
			OrganizationalUnit: entry.OU,
			ValidFrom:          certificateTime(entry.ValidFrom),
			ValidUntil:         certificateTime(entry.ValidUntil),
		})
	}
	return out
}

func certificateTime(seconds uint32) *time.Time {
	if seconds == 0 {
		return nil
	}
	t := time.Unix(int64(seconds), 0).UTC()
	return &t
}

func signingChainHash(entries []signingChainEntry) string {
	fields := make([]string, len(entries))
	for i, entry := range entries {
		fields[i] = entry.SHA256
	}
	return payloadhash.Hash(fields...)
}
