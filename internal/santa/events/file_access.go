package events

import (
	"context"
	"path/filepath"
	"time"

	"github.com/jackc/pgx/v5"
)

func insertFileAccessEvent(ctx context.Context, tx pgx.Tx, hostID int64, event FileAccessEventInput) error {
	chain := processChainColumn(processEntries(event.ProcessChain))
	primary := primaryProcess(event.ProcessChain)
	write := fileAccessEventWrite{
		HostID:                  hostID,
		RuleVersion:             event.RuleVersion,
		RuleName:                event.RuleName,
		Target:                  event.Target,
		Decision:                string(event.Decision),
		PrimaryProcessSHA256:    primary.FileSHA256,
		PrimaryProcessPath:      primary.FilePath,
		PrimaryProcessSigningID: primary.SigningID,
		PrimaryProcessTeamID:    normalizeTeamID(primary.TeamID),
		PrimaryProcessCDHash:    primary.CDHash,
		PrimaryProcessPID:       primary.PID,
		ProcessChain:            chain,
		OccurredAt:              event.OccurredAt,
	}
	_, err := tx.Exec(ctx, `
INSERT INTO santa_file_access_events (
	host_id,
	rule_version,
	rule_name,
	target,
	decision,
	primary_process_sha256,
	primary_process_path,
	primary_process_signing_id,
	primary_process_team_id,
	primary_process_cdhash,
	primary_process_pid,
	process_chain,
	occurred_at
)
VALUES (
	@host_id,
	@rule_version,
	@rule_name,
	@target,
	@decision::santa_file_access_decision,
	@primary_process_sha256,
	@primary_process_path,
	@primary_process_signing_id,
	@primary_process_team_id,
	@primary_process_cdhash,
	@primary_process_pid,
	@process_chain::jsonb,
	@occurred_at
)`, pgx.StructArgs(write))
	return err
}

func primaryProcess(chain []ProcessInput) ProcessInput {
	if len(chain) == 0 {
		return ProcessInput{}
	}
	return chain[0]
}

func processEntries(chain []ProcessInput) []Process {
	processes := make([]Process, 0, len(chain))
	for _, process := range chain {
		processes = append(processes, Process{
			PID:          process.PID,
			FilePath:     process.FilePath,
			FileName:     fileNameFromPath(process.FilePath),
			FileSHA256:   process.FileSHA256,
			SigningID:    process.SigningID,
			TeamID:       normalizeTeamID(process.TeamID),
			CDHash:       process.CDHash,
			SigningChain: signingChainOutputEntries(signingChainEntries(process.SigningChain)),
		})
	}
	return processes
}

func fileNameFromPath(path string) string {
	if path == "" {
		return ""
	}
	return filepath.Base(path)
}

type fileAccessEventWrite struct {
	HostID                  int64              `db:"host_id"`
	RuleVersion             string             `db:"rule_version"`
	RuleName                string             `db:"rule_name"`
	Target                  string             `db:"target"`
	Decision                string             `db:"decision"`
	PrimaryProcessSHA256    string             `db:"primary_process_sha256"`
	PrimaryProcessPath      string             `db:"primary_process_path"`
	PrimaryProcessSigningID string             `db:"primary_process_signing_id"`
	PrimaryProcessTeamID    string             `db:"primary_process_team_id"`
	PrimaryProcessCDHash    string             `db:"primary_process_cdhash"`
	PrimaryProcessPID       int32              `db:"primary_process_pid"`
	ProcessChain            processChainColumn `db:"process_chain"`
	OccurredAt              time.Time          `db:"occurred_at"`
}
