// Package agentauth manages shared secrets accepted by agent-facing protocols.
package agentauth

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodstar/internal/database"
	"github.com/woodleighschool/woodstar/internal/database/sqlc"
	"github.com/woodleighschool/woodstar/internal/dbutil"
)

// SecretVerifier reports whether a shared secret is valid for an agent.
type SecretVerifier interface {
	Verify(context.Context, Agent, string) (bool, error)
}

type Store struct {
	q *sqlc.Queries
}

func NewStore(db *database.DB) *Store {
	return &Store{q: db.Queries()}
}

func (s *Store) List(ctx context.Context) ([]AgentSecret, error) {
	rows, err := s.q.ListAgentSecrets(ctx)
	if err != nil {
		return nil, err
	}

	secrets := make([]AgentSecret, 0, len(rows))
	for _, row := range rows {
		secrets = append(secrets, agentSecretFromRecord(row))
	}
	return secrets, nil
}

func (s *Store) Create(ctx context.Context, params AgentSecretCreate) (AgentSecret, error) {
	if !params.Agent.Valid() {
		return AgentSecret{}, ErrInvalidAgent
	}
	row, err := s.q.CreateAgentSecret(ctx, sqlc.CreateAgentSecretParams{
		Agent: sqlc.Agent(params.Agent),
		Value: params.Value,
	})
	if err != nil {
		return AgentSecret{}, err
	}
	return agentSecretFromRecord(row), nil
}

func (s *Store) Update(ctx context.Context, id int64, params AgentSecretMutation) (AgentSecret, error) {
	row, err := s.q.UpdateAgentSecret(ctx, sqlc.UpdateAgentSecretParams{
		ID:    id,
		Value: params.Value,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return AgentSecret{}, dbutil.ErrNotFound
	}
	if err != nil {
		return AgentSecret{}, err
	}
	return agentSecretFromRecord(row), nil
}

func (s *Store) Verify(ctx context.Context, agent Agent, value string) (bool, error) {
	if !agent.Valid() || value == "" {
		return false, nil
	}
	return s.q.HasActiveAgentSecret(ctx, sqlc.HasActiveAgentSecretParams{
		Agent: sqlc.Agent(agent),
		Value: value,
	})
}

func (s *Store) Delete(ctx context.Context, id int64) error {
	_, err := s.q.DeleteAgentSecret(ctx, sqlc.DeleteAgentSecretParams{ID: id})
	if errors.Is(err, pgx.ErrNoRows) {
		return dbutil.ErrNotFound
	}
	return err
}

func agentSecretFromRecord(row sqlc.AgentSecret) AgentSecret {
	return AgentSecret{
		ID:        row.ID,
		Agent:     Agent(row.Agent),
		Value:     row.Value,
		CreatedAt: row.CreatedAt,
	}
}
