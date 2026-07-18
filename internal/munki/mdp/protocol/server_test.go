package protocol_test

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/go-chi/chi/v5"

	"github.com/woodleighschool/woodstar/internal/database"
	"github.com/woodleighschool/woodstar/internal/database/dbtest"
	"github.com/woodleighschool/woodstar/internal/munki/mdp"
	mdpprotocol "github.com/woodleighschool/woodstar/internal/munki/mdp/protocol"
	"github.com/woodleighschool/woodstar/internal/storage"
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
	delivery interface {
		DownloadURL(
			context.Context,
			storage.Object,
			time.Duration,
			storage.DeliveryOptions,
		) (string, error)
	},
) chi.Router {
	t.Helper()
	server := mdpprotocol.NewServer(t.Context(), store, delivery, discardLogger())
	t.Cleanup(server.Close)
	r := chi.NewRouter()
	server.RegisterRoutes(r)
	return r
}

func newStore(db *database.DB) (*mdp.Store, *mdp.Presence) {
	store := mdp.NewStore(db, discardLogger())
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
	db *database.DB,
	ctx context.Context,
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
	db, _ := dbtest.Open(t)
	store, _ := newStore(db)
	router := agentRouter(t, store, fakeDelivery{})

	cases := []struct {
		name   string
		bearer string
	}{
		{name: "missing", bearer: ""},
		{name: "unknown", bearer: "Bearer not-a-real-key"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/munki/distribution/connect", nil)
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

func TestConnectRejectsUnexpectedMessage(t *testing.T) {
	db, ctx := dbtest.Open(t)
	store, presence := newStore(db)
	point, err := store.Create(ctx, pointMutation([]string{"10.0.0.0/8"}), "worker-key")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	router := agentRouter(t, store, fakeDelivery{})
	srv := httptest.NewServer(router)
	defer srv.Close()

	ws, _, err := websocket.Dial(ctx, wsURL(srv.URL), &websocket.DialOptions{
		HTTPHeader: http.Header{"Authorization": {"Bearer worker-key"}},
	})
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer ws.Close(websocket.StatusNormalClosure, "")
	readJSON(t, ctx, ws, new(struct{}))
	eventually(t, func() bool { return presence.Online(point.ID) })

	if err := ws.Write(ctx, websocket.MessageText, []byte(`{"type":"not-an-event"}`)); err != nil {
		t.Fatalf("write unexpected message: %v", err)
	}
	eventually(t, func() bool { return !presence.Online(point.ID) })
}

func TestDisconnectDropsCurrentWorkerAndPresence(t *testing.T) {
	db, ctx := dbtest.Open(t)
	store, presence := newStore(db)
	point, err := store.Create(ctx, pointMutation(nil), "worker-key")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	protocol := mdpprotocol.NewServer(t.Context(), store, fakeDelivery{}, discardLogger())
	t.Cleanup(protocol.Close)
	router := chi.NewRouter()
	protocol.RegisterRoutes(router)
	httpServer := httptest.NewServer(router)
	defer httpServer.Close()

	ws, _, err := websocket.Dial(ctx, wsURL(httpServer.URL), &websocket.DialOptions{
		HTTPHeader: http.Header{"Authorization": {"Bearer worker-key"}},
	})
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer ws.CloseNow()
	readJSON(t, ctx, ws, new(struct{}))
	readJSON(t, ctx, ws, new(struct{}))
	eventually(t, func() bool { return presence.Online(point.ID) })

	protocol.Disconnect(point.ID)
	eventually(t, func() bool { return !presence.Online(point.ID) })
	readCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	if _, _, err := ws.Read(readCtx); websocket.CloseStatus(err) != websocket.StatusPolicyViolation {
		t.Fatalf("read after disconnect error = %v, want policy violation close", err)
	}
}

func TestDownloadURLRejectsMissingAndUnknownKey(t *testing.T) {
	db, ctx := dbtest.Open(t)
	store, _ := newStore(db)
	sha := strings.Repeat("a", 64)
	pkg := seedAvailablePackage(t, db, ctx, "Chrome", sha, 4096)
	if _, err := store.Create(ctx, pointMutation(nil), "worker-key"); err != nil {
		t.Fatalf("Create: %v", err)
	}
	router := agentRouter(t, store, fakeDelivery{})

	path := "/api/munki/distribution/packages/" + strconv.FormatInt(pkg, 10) + "/download-url"

	t.Run("missing bearer", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want 401", rec.Code)
		}
	})

	t.Run("unknown package", func(t *testing.T) {
		req := httptest.NewRequest(
			http.MethodGet,
			"/api/munki/distribution/packages/999999/download-url",
			nil,
		)
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
