package agentauth

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/woodleighschool/woodstar/internal/postgres"
)

// SecretVerifier reports whether a shared secret is valid for an agent.
type SecretVerifier interface {
	Verify(ctx context.Context, agent Agent, value string) (bool, error)
}

type Store struct {
	pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
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

func (s *Store) List(ctx context.Context) ([]AgentSecret, error) {
	rows, err := postgres.GetAll[agentSecretRow](ctx, s.pool, `
SELECT id, agent, value, created_at, deleted_at
FROM agent_secrets
WHERE deleted_at IS NULL
ORDER BY agent ASC, created_at DESC, id DESC`)
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
	params.normalize()
	if err := params.validate(); err != nil {
		return nil, err
	}
	row, err := postgres.GetOne[agentSecretRow](ctx, s.pool, `
INSERT INTO agent_secrets (agent, value)
VALUES ($1::agent, $2)
RETURNING id, agent, value, created_at, deleted_at`,
		string(params.Agent), params.Value,
	)
	if err != nil {
		return nil, postgres.MutationError(err)
	}
	out := agentSecretFromRow(row)
	return &out, nil
}

func (s *Store) Update(ctx context.Context, id int64, params AgentSecretMutation) (*AgentSecret, error) {
	if err := params.validate(); err != nil {
		return nil, err
	}
	row, err := postgres.GetOne[agentSecretRow](ctx, s.pool, `
UPDATE agent_secrets
SET value = $1
WHERE id = $2
  AND deleted_at IS NULL
RETURNING id, agent, value, created_at, deleted_at`,
		params.Value, id,
	)
	if err != nil {
		return nil, postgres.GetError(err)
	}
	out := agentSecretFromRow(row)
	return &out, nil
}

func (s *Store) Verify(ctx context.Context, agent Agent, value string) (bool, error) {
	if !agent.Valid() || value == "" {
		return false, nil
	}
	var exists bool
	err := s.pool.QueryRow(ctx, `
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
	err := s.pool.QueryRow(ctx, `
UPDATE agent_secrets
SET deleted_at = now()
WHERE id = $1
  AND deleted_at IS NULL
RETURNING id`, id).Scan(&deletedID)
	return postgres.GetError(err)
}
