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

type signingChainWrite struct {
	SHA256  string
	Entries []signingChainEntry
}

func linkSigningChainEntry(
	ctx context.Context,
	tx pgx.Tx,
	signingChainID int64,
	position int,
	certificateID int64,
) error {
	_, err := tx.Exec(ctx, `
INSERT INTO santa_signing_chain_entries (signing_chain_id, position, certificate_id)
VALUES (@signing_chain_id, @position, @certificate_id)`,
		pgx.NamedArgs{
			"signing_chain_id": signingChainID,
			"position":         position,
			"certificate_id":   certificateID,
		})
	return err
}

func linkExecutableSigningChain(
	ctx context.Context,
	tx pgx.Tx,
	executableID int64,
	signingChainID int64,
) error {
	_, err := tx.Exec(ctx, `
INSERT INTO santa_executable_signing_chains (executable_id, signing_chain_id)
VALUES (@executable_id, @signing_chain_id)
ON CONFLICT DO NOTHING`,
		pgx.NamedArgs{
			"executable_id":    executableID,
			"signing_chain_id": signingChainID,
		})
	return err
}

const upsertCertificateSQL = `
INSERT INTO santa_certificates AS c (
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
WHERE (c.common_name, c.organization, c.organizational_unit, c.valid_from, c.valid_until)
	IS DISTINCT FROM (EXCLUDED.common_name, EXCLUDED.organization, EXCLUDED.organizational_unit,
	                  EXCLUDED.valid_from, EXCLUDED.valid_until)`

func (entry signingChainEntry) certificateArgs() pgx.NamedArgs {
	return pgx.NamedArgs{
		"sha256":              entry.SHA256,
		"common_name":         entry.CommonName,
		"organization":        entry.Org,
		"organizational_unit": entry.OU,
		"valid_from":          certificateTime(entry.ValidFrom),
		"valid_until":         certificateTime(entry.ValidUntil),
	}
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
