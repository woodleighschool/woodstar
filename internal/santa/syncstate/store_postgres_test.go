//go:build postgres

package syncstate_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/woodleighschool/woodstar/internal/fault"
	"github.com/woodleighschool/woodstar/internal/hosts"
	"github.com/woodleighschool/woodstar/internal/labels"
	"github.com/woodleighschool/woodstar/internal/santa/syncstate"
	"github.com/woodleighschool/woodstar/internal/testutil/testdb"
)

const (
	emptyRulesHash  = "00000000000000000000000000000000"
	syncedRulesHash = "11111111111111111111111111111111"
	settingsDigest  = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	changedDigest   = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	removedDigest   = "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"
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
		settingsDigest,
		initial,
		syncstate.RuleCounts{},
		false,
		emptyRulesHash,
	); err != nil {
		t.Fatalf("initial prepare: %v", err)
	} else if syncType != syncstate.SyncTypeCleanAll {
		t.Fatalf("initial sync type = %q, want clean all", syncType)
	}
	if err := store.PromotePending(
		ctx,
		host.ID,
		2,
		2,
		syncstate.SyncTypeCleanAll,
		syncedRulesHash,
	); err != nil {
		t.Fatalf("promote initial: %v", err)
	}

	next := []syncstate.Target{
		target("binary", "stay", "blocklist", "stay-hash"),
		target("certificate", "new-cert", "blocklist", "new-cert-hash"),
	}
	syncType, err := store.PreparePending(ctx, host.ID, settingsDigest, next, syncstate.RuleCounts{
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
		settingsDigest,
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
		settingsDigest,
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

func TestPreparePendingUsesPolicyBoundaries(t *testing.T) {
	db, ctx := testdb.Open(t)
	store := syncstate.NewStore(db)
	host := createHost(t, ctx, db, "policy-boundaries")
	desired := []syncstate.Target{target("binary", "known", "allowlist", "known-hash")}

	if syncType, err := store.PreparePending(
		ctx, host.ID, settingsDigest, desired, syncstate.RuleCounts{}, false, emptyRulesHash,
	); err != nil {
		t.Fatalf("prepare unknown policy: %v", err)
	} else if syncType != syncstate.SyncTypeCleanAll {
		t.Fatalf("unknown policy sync type = %q, want clean all", syncType)
	}
	if err := store.PromotePending(
		ctx, host.ID, 1, 1, syncstate.SyncTypeCleanAll, syncedRulesHash,
	); err != nil {
		t.Fatalf("promote first policy: %v", err)
	}

	if syncType, err := store.PreparePending(
		ctx, host.ID, changedDigest, desired, syncstate.RuleCounts{Binary: 1}, false, syncedRulesHash,
	); err != nil {
		t.Fatalf("prepare changed policy: %v", err)
	} else if syncType != syncstate.SyncTypeCleanAll {
		t.Fatalf("changed policy sync type = %q, want clean all", syncType)
	}
	if err := store.PromotePending(
		ctx, host.ID, 1, 1, syncstate.SyncTypeClean, syncedRulesHash,
	); !errors.Is(err, fault.ErrInvalidInput) {
		t.Fatalf("accept clean for changed policy = %v, want invalid input", err)
	}
	if syncType, err := store.PreparePending(
		ctx, host.ID, changedDigest, desired, syncstate.RuleCounts{Binary: 1}, false, syncedRulesHash,
	); err != nil {
		t.Fatalf("retry changed policy: %v", err)
	} else if syncType != syncstate.SyncTypeCleanAll {
		t.Fatalf("retry sync type = %q, want clean all", syncType)
	}
	if err := store.PromotePending(
		ctx, host.ID, 1, 1, syncstate.SyncTypeCleanAll, syncedRulesHash,
	); err != nil {
		t.Fatalf("promote changed policy: %v", err)
	}

	if syncType, err := store.PreparePending(
		ctx, host.ID, removedDigest, desired, syncstate.RuleCounts{Binary: 1}, false, syncedRulesHash,
	); err != nil {
		t.Fatalf("prepare removed configuration: %v", err)
	} else if syncType != syncstate.SyncTypeCleanAll {
		t.Fatalf("removed configuration sync type = %q, want clean all", syncType)
	}
}

func TestPreparePendingEmptyCleanUsesTombstoneAndConverges(t *testing.T) {
	db, ctx := testdb.Open(t)
	store := syncstate.NewStore(db)
	host := createHost(t, ctx, db, "empty-clean")

	if syncType, err := store.PreparePending(
		ctx, host.ID, settingsDigest, nil, syncstate.RuleCounts{}, false, emptyRulesHash,
	); err != nil {
		t.Fatalf("prepare empty clean: %v", err)
	} else if syncType != syncstate.SyncTypeCleanAll {
		t.Fatalf("empty initial sync type = %q, want clean all", syncType)
	}
	page, err := store.LoadPendingPayloadPage(ctx, host.ID, "", 10)
	if err != nil {
		t.Fatalf("load empty clean payload: %v", err)
	}
	if len(page.Rules) != 1 || page.Rules[0].RuleType != "binary" ||
		len(page.Rules[0].Identifier) != 64 || !page.Rules[0].Removed {
		t.Fatalf("empty clean payload = %+v, want one stable binary removal", page.Rules)
	}
	if err := store.PromotePending(
		ctx, host.ID, 1, 1, syncstate.SyncTypeCleanAll, syncedRulesHash,
	); err != nil {
		t.Fatalf("promote empty clean: %v", err)
	}

	if syncType, err := store.PreparePending(
		ctx, host.ID, settingsDigest, nil, syncstate.RuleCounts{}, false, syncedRulesHash,
	); err != nil {
		t.Fatalf("prepare converged empty policy: %v", err)
	} else if syncType != syncstate.SyncTypeNormal {
		t.Fatalf("converged empty sync type = %q, want normal", syncType)
	}
	page, err = store.LoadPendingPayloadPage(ctx, host.ID, "", 10)
	if err != nil {
		t.Fatalf("load converged payload: %v", err)
	}
	if len(page.Rules) != 0 {
		t.Fatalf("converged payload = %+v, want no rules", page.Rules)
	}
}

func TestPreparePendingCleanAllWhenTransitiveAuthorityIsRemoved(t *testing.T) {
	db, ctx := testdb.Open(t)
	store := syncstate.NewStore(db)
	host := createHost(t, ctx, db, "transitive-authority")
	compiler := []syncstate.Target{
		target("binary", "compiler", "allowlist_compiler", "compiler-hash"),
	}

	if _, err := store.PreparePending(
		ctx, host.ID, settingsDigest, compiler, syncstate.RuleCounts{}, false, emptyRulesHash,
	); err != nil {
		t.Fatalf("prepare compiler authority: %v", err)
	}
	if err := store.PromotePending(
		ctx, host.ID, 1, 1, syncstate.SyncTypeCleanAll, syncedRulesHash,
	); err != nil {
		t.Fatalf("promote compiler authority: %v", err)
	}

	ordinary := []syncstate.Target{target("binary", "compiler", "allowlist", "ordinary-hash")}
	if syncType, err := store.PreparePending(
		ctx,
		host.ID,
		settingsDigest,
		ordinary,
		syncstate.RuleCounts{Binary: 1, Compiler: 1},
		false,
		syncedRulesHash,
	); err != nil {
		t.Fatalf("prepare removed compiler authority: %v", err)
	} else if syncType != syncstate.SyncTypeCleanAll {
		t.Fatalf("removed compiler authority sync type = %q, want clean all", syncType)
	}
}

func TestPreparePendingTreatsCELAsTransitiveAuthority(t *testing.T) {
	for _, tt := range []struct {
		name     string
		desired  []syncstate.Target
		wantType syncstate.SyncType
	}{
		{name: "removed", wantType: syncstate.SyncTypeCleanAll},
		{
			name: "expression changed",
			desired: []syncstate.Target{{
				RuleType:      "binary",
				Identifier:    "cel-authority",
				Policy:        "cel",
				CELExpression: "target.signing_id == 'second' ? ALLOWLIST_COMPILER : BLOCKLIST",
			}},
			wantType: syncstate.SyncTypeCleanAll,
		},
		{
			name: "payload metadata changed",
			desired: []syncstate.Target{{
				RuleType:      "binary",
				Identifier:    "cel-authority",
				Policy:        "cel",
				CELExpression: "target.signing_id == 'first' ? ALLOWLIST_COMPILER : BLOCKLIST",
				CustomMessage: "Updated message",
			}},
			wantType: syncstate.SyncTypeNormal,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			db, ctx := testdb.Open(t)
			store := syncstate.NewStore(db)
			host := createHost(t, ctx, db, "cel-"+tt.name)
			initial := []syncstate.Target{{
				RuleType:      "binary",
				Identifier:    "cel-authority",
				Policy:        "cel",
				CELExpression: "target.signing_id == 'first' ? ALLOWLIST_COMPILER : BLOCKLIST",
			}}

			if _, err := store.PreparePending(
				ctx, host.ID, settingsDigest, initial, syncstate.RuleCounts{}, false, emptyRulesHash,
			); err != nil {
				t.Fatalf("prepare initial CEL authority: %v", err)
			}
			if err := store.PromotePending(
				ctx, host.ID, 1, 1, syncstate.SyncTypeCleanAll, syncedRulesHash,
			); err != nil {
				t.Fatalf("promote initial CEL authority: %v", err)
			}

			syncType, err := store.PreparePending(
				ctx,
				host.ID,
				settingsDigest,
				tt.desired,
				syncstate.RuleCounts{Binary: 1},
				false,
				syncedRulesHash,
			)
			if err != nil {
				t.Fatalf("prepare changed CEL authority: %v", err)
			}
			if syncType != tt.wantType {
				t.Fatalf("sync type = %q, want %q", syncType, tt.wantType)
			}
		})
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
		settingsDigest,
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
		syncstate.SyncTypeCleanAll,
		syncedRulesHash,
	); err != nil {
		t.Fatalf("promote initial: %v", err)
	}

	syncType, err := store.PreparePending(ctx, host.ID, settingsDigest, desired, syncstate.RuleCounts{
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
		settingsDigest,
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

func TestPromotePendingOnlyPromotesValidatedPayload(t *testing.T) {
	db, ctx := testdb.Open(t)
	store := syncstate.NewStore(db)
	host := createHost(t, ctx, db, "promote")

	desired := []syncstate.Target{target("binary", "known", "allowlist", "known-hash")}
	if _, err := store.PreparePending(
		ctx,
		host.ID,
		settingsDigest,
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
		syncstate.SyncTypeCleanAll,
		syncedRulesHash,
	); !errors.Is(err, fault.ErrInvalidInput) {
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
	for _, invalid := range []struct {
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
			syncType:       syncstate.SyncTypeCleanAll,
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
			syncType:       syncstate.SyncTypeCleanAll,
			rulesHash:      "not-a-rules-hash",
		},
	} {
		t.Run(invalid.name, func(t *testing.T) {
			err := store.PromotePending(
				ctx,
				host.ID,
				invalid.rulesReceived,
				invalid.rulesProcessed,
				invalid.syncType,
				invalid.rulesHash,
			)
			if !errors.Is(err, fault.ErrInvalidInput) {
				t.Fatalf("promote error = %v, want invalid input", err)
			}
		})
	}

	if err := store.PromotePending(
		ctx,
		host.ID,
		1,
		1,
		syncstate.SyncTypeCleanAll,
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
		settingsDigest,
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
		syncstate.SyncTypeCleanAll,
		syncedRulesHash,
	); err != nil {
		t.Fatalf("promote initial: %v", err)
	}

	syncType, err := store.PreparePending(
		ctx,
		host.ID,
		settingsDigest,
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
		settingsDigest,
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
		syncstate.SyncTypeCleanAll,
		syncedRulesHash,
	); err != nil {
		t.Fatalf("promote initial: %v", err)
	}
	if syncType, err := store.PreparePending(
		ctx,
		host.ID,
		settingsDigest,
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
	); !errors.Is(err, fault.ErrInvalidInput) {
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
	); !errors.Is(err, fault.ErrInvalidInput) {
		t.Fatalf("postflight without pending error = %v, want invalid input", err)
	}
}

func target(ruleType string, identifier string, policy string, payloadVariant string) syncstate.Target {
	return syncstate.Target{
		RuleType:      ruleType,
		Identifier:    identifier,
		Policy:        policy,
		CustomMessage: payloadVariant,
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

func countRows(t *testing.T, ctx context.Context, db *pgxpool.Pool, table string, hostID int64, predicate string) int {
	t.Helper()

	var count int
	query := "SELECT count(*) FROM " + table + " WHERE host_id = $1 AND " + predicate
	if err := db.QueryRow(ctx, query, hostID).Scan(&count); err != nil {
		t.Fatalf("count %s: %v", table, err)
	}
	return count
}

func createHost(t *testing.T, ctx context.Context, db *pgxpool.Pool, suffix string) *hosts.Host {
	t.Helper()

	host, err := hosts.NewStore(db, labels.NewStore(db)).UpsertOnOrbitEnroll(ctx, hosts.InventoryUpdate{
		Hardware:     hosts.HostHardware{UUID: "syncstate-" + suffix + "-host"},
		OrbitNodeKey: "syncstate-" + suffix + "-orbit",
	})
	if err != nil {
		t.Fatalf("create host: %v", err)
	}
	return host
}
