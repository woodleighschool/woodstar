//go:build postgres

package syncstate_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/woodleighschool/woodstar/internal/database"
	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/hosts"
	"github.com/woodleighschool/woodstar/internal/santa/syncstate"
	"github.com/woodleighschool/woodstar/internal/testutil/testdb"
)

const (
	emptyRulesHash  = "00000000000000000000000000000000"
	syncedRulesHash = "11111111111111111111111111111111"
)

func TestPreparePendingNormalSyncSendsChangedRulesAndRemovals(t *testing.T) {
	db, ctx := testdb.Open(t)
	store := syncstate.NewStore(db)
	host := createHost(t, ctx, db, "delta")

	initial := []syncstate.Target{
		target("binary", "remove-me", "allowlist", "old-remove"),
		target("binary", "stay", "blocklist", "stay-hash"),
	}
	if syncType, err := store.PreparePending(
		ctx,
		host.ID,
		initial,
		syncstate.RuleCounts{},
		false,
		emptyRulesHash,
	); err != nil {
		t.Fatalf("initial prepare: %v", err)
	} else if syncType != syncstate.SyncTypeClean {
		t.Fatalf("initial sync type = %q, want clean", syncType)
	}
	if err := store.PromotePending(
		ctx,
		host.ID,
		2,
		2,
		syncstate.SyncTypeClean,
		syncedRulesHash,
	); err != nil {
		t.Fatalf("promote initial: %v", err)
	}

	next := []syncstate.Target{
		target("binary", "stay", "blocklist", "stay-hash"),
		target("certificate", "new-cert", "blocklist", "new-cert-hash"),
	}
	syncType, err := store.PreparePending(ctx, host.ID, next, syncstate.RuleCounts{
		Binary: 2,
	}, false, syncedRulesHash)
	if err != nil {
		t.Fatalf("prepare delta: %v", err)
	}
	if syncType != syncstate.SyncTypeNormal {
		t.Fatalf("sync type = %q, want normal", syncType)
	}

	page, err := store.LoadPendingPayloadPage(ctx, host.ID, "", 10)
	if err != nil {
		t.Fatalf("load payload: %v", err)
	}
	if got := payloadSummary(page.Rules); got != "binary:remove-me::true,certificate:new-cert:blocklist:false" {
		t.Fatalf("payload = %q, want changed rules and removals", got)
	}
}

func TestPreparePendingCleanSyncsWhenReportedCountsDrift(t *testing.T) {
	db, ctx := testdb.Open(t)
	store := syncstate.NewStore(db)
	host := createHost(t, ctx, db, "drift")
	desired := []syncstate.Target{target("binary", "known", "allowlist", "known-hash")}

	if _, err := store.PreparePending(
		ctx,
		host.ID,
		desired,
		syncstate.RuleCounts{},
		false,
		emptyRulesHash,
	); err != nil {
		t.Fatalf("initial prepare: %v", err)
	}
	if err := store.PromotePending(
		ctx,
		host.ID,
		1,
		1,
		syncstate.SyncTypeCleanAll,
		syncedRulesHash,
	); err != nil {
		t.Fatalf("promote initial: %v", err)
	}

	syncType, err := store.PreparePending(
		ctx,
		host.ID,
		desired,
		syncstate.RuleCounts{},
		false,
		syncedRulesHash,
	)
	if err != nil {
		t.Fatalf("prepare drift: %v", err)
	}
	if syncType != syncstate.SyncTypeClean {
		t.Fatalf("sync type = %q, want clean", syncType)
	}
	page, err := store.LoadPendingPayloadPage(ctx, host.ID, "", 10)
	if err != nil {
		t.Fatalf("load payload: %v", err)
	}
	if got := payloadSummary(page.Rules); got != "binary:known:allowlist:false" {
		t.Fatalf("payload = %q, want full desired clean payload", got)
	}
}

func TestPreparePendingIgnoresClientLocalTransitiveRulesForCountDrift(t *testing.T) {
	db, ctx := testdb.Open(t)
	store := syncstate.NewStore(db)
	host := createHost(t, ctx, db, "transitive")
	desired := []syncstate.Target{
		target("binary", "compiler", "allowlist_compiler", "compiler-hash"),
		target("binary", "known", "allowlist", "known-hash"),
	}

	if _, err := store.PreparePending(
		ctx,
		host.ID,
		desired,
		syncstate.RuleCounts{},
		false,
		emptyRulesHash,
	); err != nil {
		t.Fatalf("initial prepare: %v", err)
	}
	if err := store.PromotePending(
		ctx,
		host.ID,
		2,
		2,
		syncstate.SyncTypeClean,
		syncedRulesHash,
	); err != nil {
		t.Fatalf("promote initial: %v", err)
	}

	syncType, err := store.PreparePending(ctx, host.ID, desired, syncstate.RuleCounts{
		Binary:     3,
		Compiler:   1,
		Transitive: 1,
	}, false, syncedRulesHash)
	if err != nil {
		t.Fatalf("prepare with transitive rule: %v", err)
	}
	if syncType != syncstate.SyncTypeNormal {
		t.Fatalf("sync type = %q, want normal", syncType)
	}

	page, err := store.LoadPendingPayloadPage(ctx, host.ID, "", 10)
	if err != nil {
		t.Fatalf("load payload: %v", err)
	}
	if got := payloadSummary(page.Rules); got != "" {
		t.Fatalf("payload = %q, want no unchanged rules", got)
	}
}

func TestLoadPendingPayloadPagePaginatesDeterministically(t *testing.T) {
	db, ctx := testdb.Open(t)
	store := syncstate.NewStore(db)
	host := createHost(t, ctx, db, "page")

	desired := []syncstate.Target{
		target("teamid", "e", "blocklist", "e"),
		target("binary", "b", "blocklist", "b"),
		target("cdhash", "a", "blocklist", "a"),
		target("certificate", "d", "blocklist", "d"),
		target("signingid", "c", "blocklist", "c"),
	}
	if _, err := store.PreparePending(
		ctx,
		host.ID,
		desired,
		syncstate.RuleCounts{},
		true,
		emptyRulesHash,
	); err != nil {
		t.Fatalf("prepare pending: %v", err)
	}

	first, err := store.LoadPendingPayloadPage(ctx, host.ID, "", 2)
	if err != nil {
		t.Fatalf("load first page: %v", err)
	}
	if first.Cursor == "" {
		t.Fatal("first page cursor is empty")
	}
	if got := payloadIdentifiers(first.Rules); got != "a,b" {
		t.Fatalf("first page identifiers = %q, want a,b", got)
	}

	second, err := store.LoadPendingPayloadPage(ctx, host.ID, first.Cursor, 2)
	if err != nil {
		t.Fatalf("load second page: %v", err)
	}
	if second.Cursor == "" {
		t.Fatal("second page cursor is empty")
	}
	if got := payloadIdentifiers(second.Rules); got != "c,d" {
		t.Fatalf("second page identifiers = %q, want c,d", got)
	}

	third, err := store.LoadPendingPayloadPage(ctx, host.ID, second.Cursor, 2)
	if err != nil {
		t.Fatalf("load third page: %v", err)
	}
	if third.Cursor != "" {
		t.Fatalf("third page cursor = %q, want empty", third.Cursor)
	}
	if got := payloadIdentifiers(third.Rules); got != "e" {
		t.Fatalf("third page identifiers = %q, want e", got)
	}
}

func TestPromotePendingRecordsAttemptsAndOnlyPromotesProcessedPayload(t *testing.T) {
	db, ctx := testdb.Open(t)
	store := syncstate.NewStore(db)
	host := createHost(t, ctx, db, "promote")

	desired := []syncstate.Target{target("binary", "known", "allowlist", "known-hash")}
	if _, err := store.PreparePending(
		ctx,
		host.ID,
		desired,
		syncstate.RuleCounts{},
		false,
		emptyRulesHash,
	); err != nil {
		t.Fatalf("prepare pending: %v", err)
	}
	if err := store.PromotePending(
		ctx,
		host.ID,
		1,
		0,
		syncstate.SyncTypeClean,
		syncedRulesHash,
	); !errors.Is(err, dbutil.ErrInvalidInput) {
		t.Fatalf("mismatch promote error = %v, want invalid input", err)
	}
	if got := countRows(t, ctx, db, "santa_sync_targets", host.ID, "phase = 'applied'"); got != 0 {
		t.Fatalf("applied rows after mismatch = %d, want 0", got)
	}
	page, err := store.LoadPendingPayloadPage(ctx, host.ID, "", 10)
	if err != nil {
		t.Fatalf("load pending after mismatch: %v", err)
	}
	if got := payloadSummary(page.Rules); got != "binary:known:allowlist:false" {
		t.Fatalf("pending payload after mismatch = %q, want desired rule", got)
	}
	for _, attempt := range []struct {
		name           string
		rulesReceived  uint32
		rulesProcessed uint32
		syncType       syncstate.SyncType
		rulesHash      string
	}{
		{
			name:           "received count",
			rulesReceived:  0,
			rulesProcessed: 1,
			syncType:       syncstate.SyncTypeClean,
			rulesHash:      syncedRulesHash,
		},
		{
			name:           "sync type",
			rulesReceived:  1,
			rulesProcessed: 1,
			syncType:       syncstate.SyncTypeNormal,
			rulesHash:      syncedRulesHash,
		},
		{
			name:           "rules hash",
			rulesReceived:  1,
			rulesProcessed: 1,
			syncType:       syncstate.SyncTypeClean,
			rulesHash:      "not-a-rules-hash",
		},
	} {
		t.Run(attempt.name, func(t *testing.T) {
			err := store.PromotePending(
				ctx,
				host.ID,
				attempt.rulesReceived,
				attempt.rulesProcessed,
				attempt.syncType,
				attempt.rulesHash,
			)
			if !errors.Is(err, dbutil.ErrInvalidInput) {
				t.Fatalf("promote error = %v, want invalid input", err)
			}
		})
	}

	if err := store.PromotePending(
		ctx,
		host.ID,
		1,
		1,
		syncstate.SyncTypeClean,
		syncedRulesHash,
	); err != nil {
		t.Fatalf("successful promote: %v", err)
	}
	if got := countRows(t, ctx, db, "santa_sync_targets", host.ID, "phase = 'applied'"); got != 1 {
		t.Fatalf("applied rows after success = %d, want 1", got)
	}
	page, err = store.LoadPendingPayloadPage(ctx, host.ID, "", 10)
	if err != nil {
		t.Fatalf("load pending after success: %v", err)
	}
	if got := payloadSummary(page.Rules); got != "" {
		t.Fatalf("pending payload after success = %q, want empty", got)
	}
}

func TestPreparePendingUsesRulesHashToDetectClientDrift(t *testing.T) {
	db, ctx := testdb.Open(t)
	store := syncstate.NewStore(db)
	host := createHost(t, ctx, db, "hash-drift")
	desired := []syncstate.Target{target("binary", "known", "allowlist", "known-hash")}

	if _, err := store.PreparePending(
		ctx,
		host.ID,
		desired,
		syncstate.RuleCounts{},
		false,
		emptyRulesHash,
	); err != nil {
		t.Fatalf("prepare initial: %v", err)
	}
	if err := store.PromotePending(
		ctx,
		host.ID,
		1,
		1,
		syncstate.SyncTypeClean,
		syncedRulesHash,
	); err != nil {
		t.Fatalf("promote initial: %v", err)
	}

	syncType, err := store.PreparePending(
		ctx,
		host.ID,
		desired,
		syncstate.RuleCounts{Binary: 1},
		false,
		"22222222222222222222222222222222",
	)
	if err != nil {
		t.Fatalf("prepare drift: %v", err)
	}
	if syncType != syncstate.SyncTypeClean {
		t.Fatalf("sync type = %q, want clean", syncType)
	}
}

func TestPromotePendingValidatesEmptySyncHashAndPendingState(t *testing.T) {
	db, ctx := testdb.Open(t)
	store := syncstate.NewStore(db)
	host := createHost(t, ctx, db, "empty-postflight")
	desired := []syncstate.Target{target("binary", "known", "allowlist", "known-hash")}

	if _, err := store.PreparePending(
		ctx,
		host.ID,
		desired,
		syncstate.RuleCounts{},
		false,
		emptyRulesHash,
	); err != nil {
		t.Fatalf("prepare initial: %v", err)
	}
	if err := store.PromotePending(
		ctx,
		host.ID,
		1,
		1,
		syncstate.SyncTypeClean,
		syncedRulesHash,
	); err != nil {
		t.Fatalf("promote initial: %v", err)
	}
	if syncType, err := store.PreparePending(
		ctx,
		host.ID,
		desired,
		syncstate.RuleCounts{Binary: 1},
		false,
		syncedRulesHash,
	); err != nil {
		t.Fatalf("prepare empty sync: %v", err)
	} else if syncType != syncstate.SyncTypeNormal {
		t.Fatalf("sync type = %q, want normal", syncType)
	}

	if err := store.PromotePending(
		ctx,
		host.ID,
		0,
		0,
		syncstate.SyncTypeNormal,
		"22222222222222222222222222222222",
	); !errors.Is(err, dbutil.ErrInvalidInput) {
		t.Fatalf("changed hash error = %v, want invalid input", err)
	}
	if err := store.PromotePending(
		ctx,
		host.ID,
		0,
		0,
		syncstate.SyncTypeNormal,
		syncedRulesHash,
	); err != nil {
		t.Fatalf("promote empty sync: %v", err)
	}
	if err := store.PromotePending(
		ctx,
		host.ID,
		0,
		0,
		syncstate.SyncTypeNormal,
		syncedRulesHash,
	); !errors.Is(err, dbutil.ErrInvalidInput) {
		t.Fatalf("postflight without pending error = %v, want invalid input", err)
	}
}

func target(ruleType string, identifier string, policy string, payloadHash string) syncstate.Target {
	return syncstate.Target{
		RuleType:    ruleType,
		Identifier:  identifier,
		Policy:      policy,
		PayloadHash: payloadHash,
	}
}

func payloadSummary(rules []syncstate.PayloadRule) string {
	parts := make([]string, 0, len(rules))
	for _, rule := range rules {
		parts = append(parts, rule.RuleType+":"+rule.Identifier+":"+rule.Policy+":"+boolString(rule.Removed))
	}
	return strings.Join(parts, ",")
}

func payloadIdentifiers(rules []syncstate.PayloadRule) string {
	parts := make([]string, 0, len(rules))
	for _, rule := range rules {
		parts = append(parts, rule.Identifier)
	}
	return strings.Join(parts, ",")
}

func boolString(value bool) string {
	if value {
		return "true"
	}
	return "false"
}

func countRows(t *testing.T, ctx context.Context, db *database.DB, table string, hostID int64, predicate string) int {
	t.Helper()

	var count int
	query := "SELECT count(*) FROM " + table + " WHERE host_id = $1 AND " + predicate
	if err := db.Pool().QueryRow(ctx, query, hostID).Scan(&count); err != nil {
		t.Fatalf("count %s: %v", table, err)
	}
	return count
}

func createHost(t *testing.T, ctx context.Context, db *database.DB, suffix string) *hosts.Host {
	t.Helper()

	host, err := hosts.NewStore(db).UpsertOnOrbitEnroll(ctx, hosts.InventoryUpdate{
		Hardware:     hosts.HostHardware{UUID: "syncstate-" + suffix + "-host"},
		OrbitNodeKey: "syncstate-" + suffix + "-orbit",
	})
	if err != nil {
		t.Fatalf("create host: %v", err)
	}
	return host
}
