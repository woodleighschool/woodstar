package httpapi

import (
	"context"
	"fmt"

	"github.com/woodleighschool/woodstar/internal/hosts"
)

type agentVersionLoader interface {
	AgentVersions(ctx context.Context, hostIDs []int64) (map[int64]string, error)
}

func enrichHostAgents(
	ctx context.Context,
	rows []hosts.Host,
	munkiVersions agentVersionLoader,
	santaVersions agentVersionLoader,
) error {
	if len(rows) == 0 {
		return nil
	}
	hostIDs := make([]int64, len(rows))
	for i := range rows {
		hostIDs[i] = rows[i].ID
	}
	if munkiVersions != nil {
		versions, err := munkiVersions.AgentVersions(ctx, hostIDs)
		if err != nil {
			return fmt.Errorf("load Munki agent versions: %w", err)
		}
		for i := range rows {
			rows[i].Agents.Munki.Version = versions[rows[i].ID]
		}
	}
	if santaVersions != nil {
		versions, err := santaVersions.AgentVersions(ctx, hostIDs)
		if err != nil {
			return fmt.Errorf("load Santa agent versions: %w", err)
		}
		for i := range rows {
			rows[i].Agents.Santa.Version = versions[rows[i].ID]
		}
	}
	return nil
}
