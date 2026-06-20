// Package agentauth manages shared secrets accepted by agent-facing protocols.
package agentauth

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodstar/internal/database"
	"github.com/woodleighschool/woodstar/internal/dbutil"
)

// SecretVerifier reports whether a shared secret is valid for an agent.
type SecretVerifier interface {
	Verify(context.Context, Agent, string) (bool, error)
}

type Store struct {
	db *database.DB
}

func NewStore(db *database.DB) *Store {
	return &Store{db: db}
}

type agentSecretRow struct {
	ID        int64      `db:"id"`
	Agent     string     `db:"agent"`
	Value     string     `db:"value"`
	CreatedAt time.Time  `db:"created_at"`
	DeletedAt *time.Time `db:"deleted_at"`
}

func agentSecretFromRow(row agentSecretRow) AgentSecret {
	return AgentSecret{
		ID:        row.ID,
		Agent:     Agent(row.Agent),
		Value:     row.Value,
		CreatedAt: row.CreatedAt,
	}
}

const agentRecordSQL = `
SELECT id, agent, value, created_at, deleted_at
FROM agent_secrets`

func (s *Store) List(ctx context.Context) ([]AgentSecret, error) {
	qrows, err := s.db.Pool().Query(ctx, agentRecordSQL+`
WHERE deleted_at IS NULL
ORDER BY agent ASC, created_at DESC, id DESC`)
	if err != nil {
		return nil, err
	}
	rows, err := pgx.CollectRows(qrows, pgx.RowToStructByName[agentSecretRow])
	if err != nil {
		return nil, err
	}
	out := make([]AgentSecret, len(rows))
	for i, row := range rows {
		out[i] = agentSecretFromRow(row)
	}
	return out, nil
}

func (s *Store) Create(ctx context.Context, params AgentSecretCreate) (*AgentSecret, error) {
	if !params.Agent.Valid() {
		return nil, ErrInvalidAgent
	}
	row, err := dbutil.GetOne[agentSecretRow](ctx, s.db.Pool(), `
INSERT INTO agent_secrets (agent, value)
VALUES ($1::agent, $2)
RETURNING id, agent, value, created_at, deleted_at`,
		string(params.Agent), params.Value,
	)
	if err != nil {
		return nil, dbutil.MutationError(err)
	}
	out := agentSecretFromRow(row)
	return &out, nil
}

func (s *Store) Update(ctx context.Context, id int64, params AgentSecretMutation) (*AgentSecret, error) {
	row, err := dbutil.GetOne[agentSecretRow](ctx, s.db.Pool(), `
UPDATE agent_secrets
SET value = $1
WHERE id = $2
  AND deleted_at IS NULL
RETURNING id, agent, value, created_at, deleted_at`,
		params.Value, id,
	)
	if err != nil {
		return nil, dbutil.GetError(err)
	}
	out := agentSecretFromRow(row)
	return &out, nil
}

func (s *Store) Verify(ctx context.Context, agent Agent, value string) (bool, error) {
	if !agent.Valid() || value == "" {
		return false, nil
	}
	var exists bool
	err := s.db.Pool().QueryRow(ctx, `
SELECT EXISTS (
    SELECT 1
    FROM agent_secrets
    WHERE agent = $1::agent
      AND value = $2
      AND deleted_at IS NULL
)`, string(agent), value).Scan(&exists)
	return exists, err
}

func (s *Store) Delete(ctx context.Context, id int64) error {
	var deletedID int64
	err := s.db.Pool().QueryRow(ctx, `
UPDATE agent_secrets
SET deleted_at = now()
WHERE id = $1
  AND deleted_at IS NULL
RETURNING id`, id).Scan(&deletedID)
	return dbutil.GetError(err)
}
