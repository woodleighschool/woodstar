package syncstate_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/woodleighschool/woodstar/internal/database"
	"github.com/woodleighschool/woodstar/internal/database/dbtest"
	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/hosts"
	"github.com/woodleighschool/woodstar/internal/santa/syncstate"
)

func TestPreparePendingInitialSyncIsCleanAndFreezesDesiredPayload(t *testing.T) {
	db, ctx := dbtest.Open(t)
	store := syncstate.NewStore(db)
	host := createHost(t, ctx, db, "initial")

	syncType, err := store.PreparePending(ctx, host.ID, "client-hash", []syncstate.Target{
		target("binary", "a", "blocklist", "hash-a"),
	}, syncstate.RuleCounts{}, false)
	if err != nil {
		t.Fatalf("prepare pending: %v", err)
	}
	if syncType != syncstate.SyncTypeClean {
		t.Fatalf("sync type = %q, want clean", syncType)
	}

	page, err := store.LoadPendingPayloadPage(ctx, host.ID, "", 10)
	if err != nil {
		t.Fatalf("load pending payload: %v", err)
	}
	if got := payloadSummary(page.Rules); got != "binary:a:blocklist:false" {
		t.Fatalf("payload = %q, want initial desired rule", got)
	}
	if got := countRows(t, ctx, db, "santa_sync_targets", host.ID, "phase = 'desired'"); got != 1 {
		t.Fatalf("desired rows = %d, want 1", got)
	}
}

func TestPreparePendingNormalSyncSendsChangedRulesAndRemovals(t *testing.T) {
	db, ctx := dbtest.Open(t)
	store := syncstate.NewStore(db)
	host := createHost(t, ctx, db, "delta")

	initial := []syncstate.Target{
		target("binary", "remove-me", "allowlist", "old-remove"),
		target("binary", "stay", "blocklist", "stay-hash"),
	}
	if syncType, err := store.PreparePending(ctx, host.ID, "", initial, syncstate.RuleCounts{}, false); err != nil {
		t.Fatalf("initial prepare: %v", err)
	} else if syncType != syncstate.SyncTypeClean {
		t.Fatalf("initial sync type = %q, want clean", syncType)
	}
	if err := store.PromotePending(ctx, host.ID, "applied-hash", 2, 2); err != nil {
		t.Fatalf("promote initial: %v", err)
	}

	next := []syncstate.Target{
		target("binary", "stay", "blocklist", "stay-hash"),
		target("certificate", "new-cert", "blocklist", "new-cert-hash"),
	}
	syncType, err := store.PreparePending(ctx, host.ID, "client-hash", next, syncstate.RuleCounts{
		Binary:      1,
		Certificate: 1,
	}, false)
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
		t.Fatalf("payload = %q, want remove plus new cert", got)
	}
}

func TestPreparePendingResendsChangedPayloadHash(t *testing.T) {
	db, ctx := dbtest.Open(t)
	store := syncstate.NewStore(db)
	host := createHost(t, ctx, db, "hash")

	if _, err := store.PreparePending(ctx, host.ID, "", []syncstate.Target{
		targetWithMessage("binary", "same-id", "blocklist", "old", "old-hash"),
	}, syncstate.RuleCounts{}, false); err != nil {
		t.Fatalf("initial prepare: %v", err)
	}
	if err := store.PromotePending(ctx, host.ID, "applied-hash", 1, 1); err != nil {
		t.Fatalf("promote initial: %v", err)
	}

	syncType, err := store.PreparePending(ctx, host.ID, "", []syncstate.Target{
		targetWithMessage("binary", "same-id", "blocklist", "new", "new-hash"),
	}, syncstate.RuleCounts{Binary: 1}, false)
	if err != nil {
		t.Fatalf("prepare changed payload: %v", err)
	}
	if syncType != syncstate.SyncTypeNormal {
		t.Fatalf("sync type = %q, want normal", syncType)
	}

	page, err := store.LoadPendingPayloadPage(ctx, host.ID, "", 10)
	if err != nil {
		t.Fatalf("load payload: %v", err)
	}
	if len(page.Rules) != 1 || page.Rules[0].Identifier != "same-id" || page.Rules[0].CustomMessage != "new" {
		t.Fatalf("payload = %+v, want changed same-id rule", page.Rules)
	}
}

func TestPreparePendingCleanSyncsWhenReportedCountsDrift(t *testing.T) {
	db, ctx := dbtest.Open(t)
	store := syncstate.NewStore(db)
	host := createHost(t, ctx, db, "drift")
	desired := []syncstate.Target{target("binary", "known", "allowlist", "known-hash")}

	if _, err := store.PreparePending(ctx, host.ID, "", desired, syncstate.RuleCounts{}, false); err != nil {
		t.Fatalf("initial prepare: %v", err)
	}
	if err := store.PromotePending(ctx, host.ID, "applied-hash", 1, 1); err != nil {
		t.Fatalf("promote initial: %v", err)
	}

	syncType, err := store.PreparePending(ctx, host.ID, "", desired, syncstate.RuleCounts{}, false)
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

func TestLoadPendingPayloadPagePaginatesDeterministically(t *testing.T) {
	db, ctx := dbtest.Open(t)
	store := syncstate.NewStore(db)
	host := createHost(t, ctx, db, "page")

	desired := []syncstate.Target{
		target("teamid", "e", "blocklist", "e"),
		target("binary", "b", "blocklist", "b"),
		target("cdhash", "a", "blocklist", "a"),
		target("certificate", "d", "blocklist", "d"),
		target("signingid", "c", "blocklist", "c"),
	}
	if _, err := store.PreparePending(ctx, host.ID, "", desired, syncstate.RuleCounts{}, true); err != nil {
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

func TestLoadPendingPayloadPageRejectsInvalidCursor(t *testing.T) {
	db, ctx := dbtest.Open(t)
	store := syncstate.NewStore(db)

	_, err := store.LoadPendingPayloadPage(ctx, 1, "not-base64", 2)
	if !errors.Is(err, dbutil.ErrInvalidInput) {
		t.Fatalf("error = %v, want ErrInvalidInput", err)
	}
}

func TestPromotePendingRecordsAttemptsAndOnlyPromotesProcessedPayload(t *testing.T) {
	db, ctx := dbtest.Open(t)
	store := syncstate.NewStore(db)
	host := createHost(t, ctx, db, "promote")

	desired := []syncstate.Target{target("binary", "known", "allowlist", "known-hash")}
	if _, err := store.PreparePending(ctx, host.ID, "", desired, syncstate.RuleCounts{}, false); err != nil {
		t.Fatalf("prepare pending: %v", err)
	}
	if err := store.PromotePending(ctx, host.ID, "mismatch-hash", 1, 0); err != nil {
		t.Fatalf("mismatch promote: %v", err)
	}
	if got := countRows(t, ctx, db, "santa_sync_targets", host.ID, "phase = 'applied'"); got != 0 {
		t.Fatalf("applied rows after mismatch = %d, want 0", got)
	}
	if got := countRows(t, ctx, db, "santa_sync_pending_rules", host.ID, "true"); got != 1 {
		t.Fatalf("pending rows after mismatch = %d, want 1", got)
	}

	if err := store.PromotePending(ctx, host.ID, "success-hash", 1, 1); err != nil {
		t.Fatalf("successful promote: %v", err)
	}
	if got := countRows(t, ctx, db, "santa_sync_targets", host.ID, "phase = 'applied'"); got != 1 {
		t.Fatalf("applied rows after success = %d, want 1", got)
	}
	if got := countRows(t, ctx, db, "santa_sync_pending_rules", host.ID, "true"); got != 0 {
		t.Fatalf("pending rows after success = %d, want 0", got)
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

func targetWithMessage(
	ruleType string,
	identifier string,
	policy string,
	message string,
	payloadHash string,
) syncstate.Target {
	target := target(ruleType, identifier, policy, payloadHash)
	target.CustomMessage = message
	return target
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
