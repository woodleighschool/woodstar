package scope

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodstar/internal/database"
	"github.com/woodleighschool/woodstar/internal/dbutil"
)

// Store persists label scopes for resources that target labels.
type Store struct {
	db *database.DB
}

// NewStore returns a label-scope store backed by db.
func NewStore(db *database.DB) *Store {
	return &Store{db: db}
}

type owner struct {
	table           string
	idColumn        string
	modeColumn      string
	joinTable       string
	joinOwnerColumn string
}

var (
	queryOwner = owner{
		table:           "queries",
		idColumn:        "id",
		modeColumn:      "label_scope_mode",
		joinTable:       "query_labels",
		joinOwnerColumn: "query_id",
	}
	checkOwner = owner{
		table:           "checks",
		idColumn:        "id",
		modeColumn:      "label_scope_mode",
		joinTable:       "check_labels",
		joinOwnerColumn: "check_id",
	}
)

// LoadQuery reads the label scope for a saved query.
func (s *Store) LoadQuery(ctx context.Context, queryID int64) (LabelScope, error) {
	return s.load(ctx, queryOwner, queryID)
}

// LoadQueries reads label scopes for saved queries keyed by query ID.
func (s *Store) LoadQueries(ctx context.Context, queryIDs []int64) (map[int64]LabelScope, error) {
	return s.loadMany(ctx, queryOwner, queryIDs)
}

// LoadCheck reads the label scope for a check.
func (s *Store) LoadCheck(ctx context.Context, checkID int64) (LabelScope, error) {
	return s.load(ctx, checkOwner, checkID)
}

// LoadChecks reads label scopes for checks keyed by check ID.
func (s *Store) LoadChecks(ctx context.Context, checkIDs []int64) (map[int64]LabelScope, error) {
	return s.loadMany(ctx, checkOwner, checkIDs)
}

// ReplaceQuery replaces the label scope for a saved query inside tx.
func (s *Store) ReplaceQuery(ctx context.Context, tx pgx.Tx, queryID int64, lscope LabelScope) error {
	return replace(ctx, tx, queryOwner, queryID, lscope)
}

// ReplaceCheck replaces the label scope for a check inside tx.
func (s *Store) ReplaceCheck(ctx context.Context, tx pgx.Tx, checkID int64, lscope LabelScope) error {
	return replace(ctx, tx, checkOwner, checkID, lscope)
}

func (s *Store) load(ctx context.Context, owner owner, ownerID int64) (LabelScope, error) {
	scopes, err := s.loadMany(ctx, owner, []int64{ownerID})
	if err != nil {
		return LabelScope{}, err
	}
	lscope, ok := scopes[ownerID]
	if !ok {
		return LabelScope{}, dbutil.ErrNotFound
	}
	return lscope, nil
}

func (s *Store) loadMany(ctx context.Context, owner owner, ownerIDs []int64) (map[int64]LabelScope, error) {
	ownerIDs = cleanPositiveIDs(ownerIDs)
	scopes := make(map[int64]LabelScope, len(ownerIDs))
	if len(ownerIDs) == 0 {
		return scopes, nil
	}

	rows, err := s.db.Pool().Query(ctx,
		fmt.Sprintf(
			"SELECT %s, %s FROM %s WHERE %s = ANY($1::bigint[])",
			owner.idColumn,
			owner.modeColumn,
			owner.table,
			owner.idColumn,
		),
		ownerIDs,
	)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var ownerID int64
		var mode LabelScopeMode
		if err := rows.Scan(&ownerID, &mode); err != nil {
			rows.Close()
			return nil, err
		}
		scopes[ownerID] = LabelScope{Mode: mode}
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	rows.Close()

	rows, err = s.db.Pool().Query(ctx,
		fmt.Sprintf(
			"SELECT %s, label_id FROM %s WHERE %s = ANY($1::bigint[]) ORDER BY %s, label_id",
			owner.joinOwnerColumn,
			owner.joinTable,
			owner.joinOwnerColumn,
			owner.joinOwnerColumn,
		),
		ownerIDs,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var ownerID int64
		var labelID int64
		if err := rows.Scan(&ownerID, &labelID); err != nil {
			return nil, err
		}
		lscope := scopes[ownerID]
		lscope.LabelIDs = append(lscope.LabelIDs, labelID)
		scopes[ownerID] = lscope
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for ownerID, lscope := range scopes {
		scopes[ownerID] = NormalizeLabelScope(lscope)
	}
	return scopes, nil
}

func replace(ctx context.Context, tx pgx.Tx, owner owner, ownerID int64, lscope LabelScope) error {
	lscope = NormalizeLabelScope(lscope)
	if _, err := tx.Exec(ctx,
		fmt.Sprintf(
			"UPDATE %s SET %s = $2 WHERE %s = $1",
			owner.table,
			owner.modeColumn,
			owner.idColumn,
		),
		ownerID, lscope.Mode,
	); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx,
		fmt.Sprintf("DELETE FROM %s WHERE %s = $1", owner.joinTable, owner.joinOwnerColumn),
		ownerID,
	); err != nil {
		return err
	}
	if len(lscope.LabelIDs) == 0 {
		return nil
	}
	_, err := tx.Exec(ctx,
		fmt.Sprintf(
			"INSERT INTO %s (%s, label_id) SELECT $1, unnest($2::bigint[])",
			owner.joinTable,
			owner.joinOwnerColumn,
		),
		ownerID,
		lscope.LabelIDs,
	)
	return err
}
