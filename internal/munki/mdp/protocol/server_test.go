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

type fakePresigner struct {
	url     string
	options *storage.GetOptions
}

func (f fakePresigner) PresignGet(
	_ context.Context,
	_ string,
	_ time.Duration,
	options storage.GetOptions,
) (string, error) {
	if f.options != nil {
		*f.options = options
	}
	return f.url, nil
}

func discardLogger() *slog.Logger {
	return slog.New(slog.DiscardHandler)
}

// agentRouter wires the MDP protocol routes with a hub that writes the shared
// presence set. The hub is closed when the test ends.
func agentRouter(
	t *testing.T,
	store *mdp.Store,
	presigner storage.Presigner,
) chi.Router {
	t.Helper()
	server := mdpprotocol.NewServer(t.Context(), store, presigner, discardLogger())
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
	router := agentRouter(t, store, fakePresigner{})

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

func TestConnectDeliversIdentityAndDesiredSetThenRecordsState(t *testing.T) {
	db, ctx := dbtest.Open(t)
	store, presence := newStore(db)
	sha := strings.Repeat("a", 64)
	pkg := seedAvailablePackage(t, db, ctx, "Chrome", sha, 4096)
	point, err := store.Create(ctx, pointMutation([]string{"10.0.0.0/8"}), "worker-key")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	router := agentRouter(t, store, fakePresigner{})
	srv := httptest.NewServer(router)
	defer srv.Close()

	ws, _, err := websocket.Dial(ctx, wsURL(srv.URL), &websocket.DialOptions{
		HTTPHeader: http.Header{"Authorization": {"Bearer worker-key"}},
	})
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer ws.Close(websocket.StatusNormalClosure, "")

	// hello carries identity only; the desired set follows in its own message.
	var hello struct {
		Type              string `json:"type"`
		DistributionPoint struct {
			ID int64 `json:"id"`
		} `json:"distribution_point"`
	}
	readJSON(t, ctx, ws, &hello)
	if hello.Type != "hello" || hello.DistributionPoint.ID != point.ID {
		t.Fatalf("hello = %+v, want hello for point %d", hello, point.ID)
	}

	var desired struct {
		Type     string `json:"type"`
		Packages []struct {
			PackageID int64  `json:"package_id"`
			Filename  string `json:"filename"`
			SHA256    string `json:"sha256"`
			SizeBytes int64  `json:"size_bytes"`
		} `json:"packages"`
	}
	readJSON(t, ctx, ws, &desired)
	if desired.Type != "desired_set" || len(desired.Packages) != 1 {
		t.Fatalf("desired_set = %+v, want one package", desired)
	}
	got := desired.Packages[0]
	if got.PackageID != pkg || got.Filename != "Chrome.pkg" || got.SHA256 != sha || got.SizeBytes != 4096 {
		t.Fatalf("desired package = %+v, want mirror bytes for %d", got, pkg)
	}

	// The connection makes the point online for the resolver.
	eventually(t, func() bool { return presence.Online(point.ID) })

	// A package_current event is recorded server-side and gates eligibility.
	current := `{"type":"package_current","package_id":` +
		strconv.FormatInt(pkg, 10) + `,"sha256":"` + sha + `"}`
	if err := ws.Write(ctx, websocket.MessageText, []byte(current)); err != nil {
		t.Fatalf("write package_current: %v", err)
	}
	eventually(t, func() bool {
		detail, err := store.GetByID(ctx, point.ID)
		return err == nil &&
			len(detail.Packages) == 1 &&
			detail.Packages[0].Status == mdp.PackageStatusCurrent
	})
}

func TestConnectRejectsUnexpectedMessage(t *testing.T) {
	db, ctx := dbtest.Open(t)
	store, presence := newStore(db)
	point, err := store.Create(ctx, pointMutation([]string{"10.0.0.0/8"}), "worker-key")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	router := agentRouter(t, store, fakePresigner{})
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

	protocol := mdpprotocol.NewServer(t.Context(), store, fakePresigner{}, discardLogger())
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

func TestDownloadURLMintsPresignedURLForWorker(t *testing.T) {
	db, ctx := dbtest.Open(t)
	store, _ := newStore(db)
	sha := strings.Repeat("a", 64)
	pkg := seedAvailablePackage(t, db, ctx, "Chrome", sha, 4096)
	if _, err := store.Create(ctx, pointMutation(nil), "worker-key"); err != nil {
		t.Fatalf("Create: %v", err)
	}
	const presigned = "https://storage.example/packages/1/Chrome.pkg?cap=signed"
	var options storage.GetOptions
	router := agentRouter(t, store, fakePresigner{url: presigned, options: &options})

	path := "/api/munki/distribution/packages/" + strconv.FormatInt(pkg, 10) + "/download-url"

	t.Run("authorized", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.Header.Set("Authorization", "Bearer worker-key")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rec.Code)
		}
		var body struct {
			DownloadURL string `json:"download_url"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if body.DownloadURL != presigned {
			t.Fatalf("download_url = %q, want %q", body.DownloadURL, presigned)
		}
		if options.ContentType != "application/octet-stream" {
			t.Fatalf("download content type = %q", options.ContentType)
		}
	})

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
