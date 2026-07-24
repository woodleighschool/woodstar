//go:build postgres

package protocol_test

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/go-chi/chi/v5"

	"github.com/woodleighschool/woodstar/internal/database"
	"github.com/woodleighschool/woodstar/internal/munki/mdp"
	mdpprotocol "github.com/woodleighschool/woodstar/internal/munki/mdp/protocol"
	"github.com/woodleighschool/woodstar/internal/munki/mdp/wire"
	"github.com/woodleighschool/woodstar/internal/storage"
	"github.com/woodleighschool/woodstar/internal/testutil/testdb"
)

const (
	testServerVersion = "server-test"
	testWorkerVersion = "worker-test"
)

type fakeDelivery struct{}

func (fakeDelivery) DownloadURL(
	_ context.Context,
	_ storage.Object,
	_ time.Duration,
	_ storage.DeliveryOptions,
) (string, error) {
	return "", nil
}

func discardLogger() *slog.Logger {
	return slog.New(slog.DiscardHandler)
}

// agentRouter wires the MDP protocol routes with a hub that writes the shared
// presence set. The hub is closed when the test ends.
func agentRouter(
	t *testing.T,
	store *mdp.Store,
) chi.Router {
	t.Helper()
	server, err := mdpprotocol.NewServer(
		t.Context(), store, fakeDelivery{}, testServerVersion, discardLogger(),
	)
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	t.Cleanup(server.Close)
	r := chi.NewRouter()
	server.RegisterRoutes(r, r)
	return r
}

func newStore(db *database.DB) (*mdp.Store, *mdp.Presence) {
	store := mdp.NewStore(db, storage.NewObjectStore(db, nil, discardLogger()), discardLogger())
	return store, store.Presence()
}

func pointMutation(cidrs []string) mdp.DistributionPointMutation {
	return mdp.DistributionPointMutation{
		Name:          "Melbourne",
		Enabled:       true,
		ClientCIDRs:   cidrs,
		ClientBaseURL: "https://mdp.example",
	}
}

func seedAvailablePackage(
	t *testing.T,
	ctx context.Context,
	db *database.DB,
	name string,
	sha256 string,
	size int64,
) int64 {
	t.Helper()
	var softwareID int64
	if err := db.Pool().QueryRow(ctx,
		`INSERT INTO munki_software (name, display_name) VALUES ($1, $1) RETURNING id`, name,
	).Scan(&softwareID); err != nil {
		t.Fatalf("insert software: %v", err)
	}
	var objectID int64
	if err := db.Pool().QueryRow(ctx,
		`INSERT INTO storage_objects (prefix, filename, content_type, size_bytes, sha256, available_at)
		 VALUES ('packages', $1, 'application/octet-stream', $2, $3, now()) RETURNING id`,
		name+".pkg", size, sha256,
	).Scan(&objectID); err != nil {
		t.Fatalf("insert object: %v", err)
	}
	var packageID int64
	if err := db.Pool().QueryRow(ctx,
		`INSERT INTO munki_packages (software_id, version, installer_object_id)
		 VALUES ($1, '1.0', $2) RETURNING id`,
		softwareID, objectID,
	).Scan(&packageID); err != nil {
		t.Fatalf("insert package: %v", err)
	}
	return packageID
}

func TestConnectRejectsMissingAndUnknownKey(t *testing.T) {
	db, _ := testdb.Open(t)
	store, _ := newStore(db)
	router := agentRouter(t, store)

	cases := []struct {
		name   string
		bearer string
	}{
		{name: "missing", bearer: ""},
		{name: "unknown", bearer: "Bearer not-a-real-key"}, //nolint:gosec // Invalid bearer fixture.
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/munki/distribution/connect", nil)
			if tc.bearer != "" {
				req.Header.Set("Authorization", tc.bearer)
			}
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)
			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d, want 401", rec.Code)
			}
		})
	}
}

func TestConnectRejectsIncompatibleProtocol(t *testing.T) {
	db, ctx := testdb.Open(t)
	store, presence := newStore(db)
	point, err := store.Create(ctx, pointMutation(nil), "worker-key")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	router := agentRouter(t, store)

	cases := []struct {
		name        string
		subprotocol string
		version     string
		wantWorker  *mdp.DistributionPointWorker
	}{
		{
			name:    "missing protocol",
			version: testWorkerVersion,
			wantWorker: &mdp.DistributionPointWorker{
				BuildVersion: testWorkerVersion,
			},
		},
		{
			name:        "old protocol",
			subprotocol: "woodstar-mdp.v0",
			version:     testWorkerVersion,
			wantWorker: &mdp.DistributionPointWorker{
				ProtocolVersion: new(0),
				BuildVersion:    testWorkerVersion,
			},
		},
		{
			name:        "new protocol",
			subprotocol: "woodstar-mdp.v2",
			version:     testWorkerVersion,
			wantWorker: &mdp.DistributionPointWorker{
				ProtocolVersion: new(2),
				BuildVersion:    testWorkerVersion,
			},
		},
		{
			name:        "multiple protocols",
			subprotocol: wire.Subprotocol + ", woodstar-mdp.v2",
			version:     testWorkerVersion,
			wantWorker: &mdp.DistributionPointWorker{
				BuildVersion: testWorkerVersion,
			},
		},
		{
			name:        "missing build version",
			subprotocol: wire.Subprotocol,
			wantWorker: &mdp.DistributionPointWorker{
				ProtocolVersion: new(wire.ProtocolVersion),
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			presence.Disconnect(point.ID)
			req := httptest.NewRequestWithContext(
				t.Context(), http.MethodGet, "/api/munki/distribution/connect", nil,
			)
			req.Header.Set("Authorization", "Bearer worker-key")
			if tc.subprotocol != "" {
				req.Header.Set("Sec-WebSocket-Protocol", tc.subprotocol)
			}
			if tc.version != "" {
				req.Header.Set(wire.BuildVersionHeader, tc.version)
			}
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)
			if rec.Code != http.StatusUpgradeRequired {
				t.Fatalf("status = %d, want 426", rec.Code)
			}
			if got := rec.Header().Get(wire.ProtocolHeader); got != wire.Subprotocol {
				t.Fatalf("required protocol = %q, want %q", got, wire.Subprotocol)
			}
			if got := rec.Header().Get(wire.BuildVersionHeader); got != testServerVersion {
				t.Fatalf("server version = %q, want %q", got, testServerVersion)
			}
			worker, observed := presence.Worker(point.ID)
			if tc.wantWorker == nil {
				if observed {
					t.Fatalf("worker = %+v, want no classified worker", worker)
				}
			} else if !observed || !reflect.DeepEqual(worker, *tc.wantWorker) {
				t.Fatalf("worker = (%+v, %t), want %+v", worker, observed, *tc.wantWorker)
			}
		})
	}
}

func TestConnectRejectsUnexpectedMessage(t *testing.T) {
	db, ctx := testdb.Open(t)
	store, presence := newStore(db)
	point, err := store.Create(ctx, pointMutation([]string{"10.0.0.0/8"}), "worker-key")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	router := agentRouter(t, store)
	srv := httptest.NewServer(router)
	defer srv.Close()

	ws, response, err := dialWorker( //nolint:bodyclose // websocket.Dial always closes the handshake response body.
		ctx, wsURL(srv.URL), "worker-key",
	)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer func() { _ = ws.Close(websocket.StatusNormalClosure, "") }()
	assertNegotiated(t, ws, response)
	readJSON(t, ctx, ws, new(struct{}))
	eventually(t, func() bool { return connected(presence, point.ID) })
	worker, _ := presence.Worker(point.ID)
	if !worker.Compatible || worker.ProtocolVersion == nil ||
		*worker.ProtocolVersion != wire.ProtocolVersion ||
		worker.BuildVersion != testWorkerVersion {
		t.Fatalf(
			"worker = %+v, want connected protocol %d and build %q",
			worker,
			wire.ProtocolVersion,
			testWorkerVersion,
		)
	}

	if err := ws.Write(ctx, websocket.MessageText, []byte(`{"type":"not-an-event"}`)); err != nil {
		t.Fatalf("write unexpected message: %v", err)
	}
	eventually(t, func() bool { return !connected(presence, point.ID) })
}

func TestDisconnectDropsCurrentWorkerAndPresence(t *testing.T) {
	db, ctx := testdb.Open(t)
	store, presence := newStore(db)
	point, err := store.Create(ctx, pointMutation(nil), "worker-key")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	protocol, err := mdpprotocol.NewServer(
		t.Context(), store, fakeDelivery{}, testServerVersion, discardLogger(),
	)
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	t.Cleanup(protocol.Close)
	router := chi.NewRouter()
	protocol.RegisterRoutes(router, router)
	httpServer := httptest.NewServer(router)
	defer httpServer.Close()

	ws, response, err := dialWorker( //nolint:bodyclose // websocket.Dial always closes the handshake response body.
		ctx, wsURL(httpServer.URL), "worker-key",
	)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer func() { _ = ws.CloseNow() }()
	assertNegotiated(t, ws, response)
	readJSON(t, ctx, ws, new(struct{}))
	readJSON(t, ctx, ws, new(struct{}))
	eventually(t, func() bool { return connected(presence, point.ID) })

	protocol.Disconnect(point.ID)
	eventually(t, func() bool { return !connected(presence, point.ID) })
	readCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	if _, _, err := ws.Read(readCtx); websocket.CloseStatus(err) != websocket.StatusPolicyViolation {
		t.Fatalf("read after disconnect error = %v, want policy violation close", err)
	}
}

func TestDownloadURLRejectsMissingAndUnknownKey(t *testing.T) {
	db, ctx := testdb.Open(t)
	store, _ := newStore(db)
	sha := strings.Repeat("a", 64)
	pkg := seedAvailablePackage(t, ctx, db, "Chrome", sha, 4096)
	var nopkgID int64
	if err := db.Pool().QueryRow(ctx, `
WITH software AS (
	INSERT INTO munki_software (name, display_name)
	VALUES ('Configuration', 'Configuration')
	RETURNING id
)
INSERT INTO munki_packages (software_id, version, installer_type)
SELECT id, '1.0', 'nopkg' FROM software
RETURNING id`).Scan(&nopkgID); err != nil {
		t.Fatalf("insert nopkg package: %v", err)
	}
	if _, err := store.Create(ctx, pointMutation(nil), "worker-key"); err != nil {
		t.Fatalf("Create: %v", err)
	}
	router := agentRouter(t, store)

	path := "/api/munki/distribution/packages/" + strconv.FormatInt(pkg, 10) + "/download-url"

	t.Run("missing bearer", func(t *testing.T) {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want 401", rec.Code)
		}
	})

	t.Run("unknown package", func(t *testing.T) {
		req := httptest.NewRequestWithContext(t.Context(),
			http.MethodGet,
			"/api/munki/distribution/packages/999999/download-url",
			nil)

		req.Header.Set("Authorization", "Bearer worker-key")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want 404", rec.Code)
		}
	})

	t.Run("package without installer", func(t *testing.T) {
		req := httptest.NewRequestWithContext(t.Context(),
			http.MethodGet,
			"/api/munki/distribution/packages/"+strconv.FormatInt(nopkgID, 10)+"/download-url",
			nil)

		req.Header.Set("Authorization", "Bearer worker-key")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want 404", rec.Code)
		}
	})
}

func wsURL(httpURL string) string {
	return "ws" + strings.TrimPrefix(httpURL, "http") + "/api/munki/distribution/connect"
}

func dialWorker(
	ctx context.Context,
	url string,
	key string,
) (*websocket.Conn, *http.Response, error) {
	return websocket.Dial(ctx, url, &websocket.DialOptions{
		HTTPHeader: http.Header{
			"Authorization":         {"Bearer " + key},
			wire.BuildVersionHeader: {testWorkerVersion},
		},
		Subprotocols: []string{wire.Subprotocol},
	})
}

func assertNegotiated(t *testing.T, ws *websocket.Conn, response *http.Response) {
	t.Helper()
	if got := ws.Subprotocol(); got != wire.Subprotocol {
		t.Fatalf("selected protocol = %q, want %q", got, wire.Subprotocol)
	}
	if got := response.Header.Get(wire.BuildVersionHeader); got != testServerVersion {
		t.Fatalf("server version = %q, want %q", got, testServerVersion)
	}
}

func connected(presence *mdp.Presence, id int64) bool {
	worker, ok := presence.Worker(id)
	return ok && worker.Compatible
}

func readJSON(t *testing.T, ctx context.Context, ws *websocket.Conn, v any) {
	t.Helper()
	readCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	_, data, err := ws.Read(readCtx)
	if err != nil {
		t.Fatalf("ws read: %v", err)
	}
	if err := json.Unmarshal(data, v); err != nil {
		t.Fatalf("decode message: %v", err)
	}
}

// eventually waits until cond holds or a short deadline passes. The server
// records state asynchronously off the connection, so there is no return signal
// to wait on deterministically.
func eventually(t *testing.T, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("condition not met within timeout")
}
