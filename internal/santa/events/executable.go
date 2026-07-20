package events

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
)

func upsertExecutable(ctx context.Context, tx pgx.Tx, event ExecutionEventInput) (int64, error) {
	write := executableWrite{
		SHA256:                      event.FileSHA256,
		FileName:                    event.FileName,
		FileBundleID:                event.BundleID,
		FileBundlePath:              event.BundlePath,
		FileBundleExecutableRelPath: event.BundleExecutableRelPath,
		FileBundleName:              event.BundleName,
		FileBundleVersion:           event.BundleVersion,
		FileBundleVersionString:     event.BundleVersionString,
		FileBundleHash:              event.BundleHash,
		FileBundleHashMillis:        event.BundleHashMillis,
		FileBundleBinaryCount:       event.BundleBinaryCount,
		SigningID:                   event.SigningID,
		TeamID:                      event.TeamID,
		CDHash:                      event.CDHash,
		CodesigningFlags:            event.CodesigningFlags,
		SigningStatus:               string(normalizeSigningStatus(event.SigningStatus)),
		SecureSigningTime:           timeOrNil(event.SecureSigningTime),
		SigningTime:                 timeOrNil(event.SigningTime),
		Entitlements:                executableEntitlements(event),
	}
	var id int64
	if err := tx.QueryRow(ctx, `
INSERT INTO santa_executables (
	sha256,
	file_name,
	file_bundle_id,
	file_bundle_path,
	file_bundle_executable_rel_path,
	file_bundle_name,
	file_bundle_version,
	file_bundle_version_string,
	file_bundle_hash,
	file_bundle_hash_millis,
	file_bundle_binary_count,
	signing_id,
	team_id,
	cdhash,
	codesigning_flags,
	signing_status,
	secure_signing_time,
	signing_time,
	entitlements,
	updated_at
)
VALUES (
	@sha256,
	@file_name,
	@file_bundle_id,
	@file_bundle_path,
	@file_bundle_executable_rel_path,
	@file_bundle_name,
	@file_bundle_version,
	@file_bundle_version_string,
	@file_bundle_hash,
	@file_bundle_hash_millis,
	@file_bundle_binary_count,
	@signing_id,
	@team_id,
	@cdhash,
	@codesigning_flags,
	@signing_status::santa_signing_status,
	@secure_signing_time::timestamptz,
	@signing_time::timestamptz,
	@entitlements,
	now()
)
ON CONFLICT (sha256) DO UPDATE SET
	file_name = EXCLUDED.file_name,
	file_bundle_id = EXCLUDED.file_bundle_id,
	file_bundle_path = EXCLUDED.file_bundle_path,
	file_bundle_executable_rel_path = EXCLUDED.file_bundle_executable_rel_path,
	file_bundle_name = EXCLUDED.file_bundle_name,
	file_bundle_version = EXCLUDED.file_bundle_version,
	file_bundle_version_string = EXCLUDED.file_bundle_version_string,
	file_bundle_hash = EXCLUDED.file_bundle_hash,
	file_bundle_hash_millis = EXCLUDED.file_bundle_hash_millis,
	file_bundle_binary_count = EXCLUDED.file_bundle_binary_count,
	signing_id = EXCLUDED.signing_id,
	team_id = EXCLUDED.team_id,
	cdhash = EXCLUDED.cdhash,
	codesigning_flags = EXCLUDED.codesigning_flags,
	signing_status = EXCLUDED.signing_status,
	secure_signing_time = EXCLUDED.secure_signing_time,
	signing_time = EXCLUDED.signing_time,
	entitlements = EXCLUDED.entitlements,
	updated_at = now()
RETURNING id`, pgx.StructArgs(write)).Scan(&id); err != nil {
		return 0, err
	}
	return id, nil
}

func timeOrNil(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	return &t
}

func executableEntitlements(event ExecutionEventInput) []byte {
	if len(event.Entitlements) == 0 {
		return nil
	}
	return event.Entitlements
}

type executableWrite struct {
	SHA256                      string     `db:"sha256"`
	FileName                    string     `db:"file_name"`
	FileBundleID                string     `db:"file_bundle_id"`
	FileBundlePath              string     `db:"file_bundle_path"`
	FileBundleExecutableRelPath string     `db:"file_bundle_executable_rel_path"`
	FileBundleName              string     `db:"file_bundle_name"`
	FileBundleVersion           string     `db:"file_bundle_version"`
	FileBundleVersionString     string     `db:"file_bundle_version_string"`
	FileBundleHash              string     `db:"file_bundle_hash"`
	FileBundleHashMillis        uint32     `db:"file_bundle_hash_millis"`
	FileBundleBinaryCount       uint32     `db:"file_bundle_binary_count"`
	SigningID                   string     `db:"signing_id"`
	TeamID                      string     `db:"team_id"`
	CDHash                      string     `db:"cdhash"`
	CodesigningFlags            uint32     `db:"codesigning_flags"`
	SigningStatus               string     `db:"signing_status"`
	SecureSigningTime           *time.Time `db:"secure_signing_time"`
	SigningTime                 *time.Time `db:"signing_time"`
	Entitlements                []byte     `db:"entitlements"`
}
