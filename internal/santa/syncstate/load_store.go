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

	var pendingSyncType *string
	err = s.pool.QueryRow(ctx,
		`SELECT pending_sync_type::text
		 FROM santa_sync_state WHERE host_id = $1`,
		hostID,
	).Scan(&pendingSyncType)
	if errors.Is(err, pgx.ErrNoRows) {
		return PayloadRulePage{}, nil
	}
	if err != nil {
		return PayloadRulePage{}, err
	}
	if pendingSyncType == nil {
		return PayloadRulePage{}, nil
	}
	desired, err := loadTargets(ctx, s.pool, hostID, phaseDesired)
	if err != nil {
		return PayloadRulePage{}, err
	}
	payload := cleanSyncPayload(desired)
	if SyncType(*pendingSyncType) == SyncTypeNormal {
		applied, err := loadTargets(ctx, s.pool, hostID, phaseApplied)
		if err != nil {
			return PayloadRulePage{}, err
		}
		payload = normalSyncPayload(desired, applied)
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
