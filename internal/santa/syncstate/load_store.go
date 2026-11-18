package syncstate

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/woodleighschool/woodstar/internal/fault"
)

type PayloadRulePage struct {
	Rules  []PayloadRule
	Cursor string
}

type pageCursor struct {
	Offset int32 `json:"offset"`
}

func (s *Store) LoadPendingPayloadPage(
	ctx context.Context,
	hostID int64,
	cursor string,
	limit int32,
) (PayloadRulePage, error) {
	if limit <= 0 {
		return PayloadRulePage{}, fault.ErrInvalidInput
	}
	limitRows := int(limit)
	offset, err := decodeCursor(cursor)
	if err != nil {
		return PayloadRulePage{}, err
	}

	payload, err := s.pendingPayload(ctx, hostID)
	if err != nil {
		return PayloadRulePage{}, err
	}
	start := int(offset)
	if start >= len(payload) {
		return PayloadRulePage{}, nil
	}
	end := min(start+limitRows+1, len(payload))
	rules := payload[start:end]
	nextCursor := ""
	if len(rules) > limitRows {
		rules = rules[:limitRows]
		nextCursor, err = encodeCursor(offset + limit)
		if err != nil {
			return PayloadRulePage{}, err
		}
	}
	return PayloadRulePage{Rules: rules, Cursor: nextCursor}, nil
}

func (s *Store) pendingPayload(ctx context.Context, hostID int64) ([]PayloadRule, error) {
	// Preflight can replace the pending set and postflight can promote it during a download.
	var payload []PayloadRule
	opts := pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly}
	err := pgx.BeginTxFunc(ctx, s.pool, opts, func(tx pgx.Tx) error {
		var mode SyncType
		err := tx.QueryRow(ctx, `
SELECT pending_sync_type::text FROM santa_sync_state
WHERE host_id = $1 AND pending_sync_type IS NOT NULL`, hostID).Scan(&mode)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		desired, err := loadTargets(ctx, tx, hostID, phaseDesired)
		if err != nil {
			return err
		}
		if mode != SyncTypeNormal {
			payload = cleanSyncPayload(desired)
			return nil
		}
		applied, err := loadTargets(ctx, tx, hostID, phaseApplied)
		if err == nil {
			payload = normalSyncPayload(desired, applied)
		}
		return err
	})
	return payload, err
}

func decodeCursor(cursor string) (int32, error) {
	if cursor == "" {
		return 0, nil
	}
	payload, err := base64.RawURLEncoding.DecodeString(cursor)
	if err != nil {
		return 0, fault.ErrInvalidInput
	}
	var decoded pageCursor
	if err := json.Unmarshal(payload, &decoded); err != nil {
		return 0, fault.ErrInvalidInput
	}
	if decoded.Offset < 0 {
		return 0, fault.ErrInvalidInput
	}
	return decoded.Offset, nil
}

func encodeCursor(offset int32) (string, error) {
	payload, err := json.Marshal(pageCursor{Offset: offset})
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(payload), nil
}
