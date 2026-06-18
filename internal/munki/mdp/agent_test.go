package mdp_test

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

	"github.com/woodleighschool/woodstar/internal/database/dbtest"
	"github.com/woodleighschool/woodstar/internal/munki/mdp"
	"github.com/woodleighschool/woodstar/internal/storage"
)

type fakePresigner struct {
	url string
}

func (f fakePresigner) PresignGet(context.Context, string, time.Duration, storage.GetOptions) (string, error) {
	return f.url, nil
}

func (f fakePresigner) PresignPut(
	context.Context,
	string,
	time.Duration,
	storage.PutOptions,
) (storage.UploadTarget, error) {
	return storage.UploadTarget{}, nil
}

func discardLogger() *slog.Logger {
	return slog.New(slog.DiscardHandler)
}

// agentRouter wires the MDP protocol routes with a hub that writes the shared
// presence set. The hub is closed when the test ends.
func agentRouter(
	t *testing.T,
	store *mdp.Store,
	presence *mdp.Presence,
	presigner storage.Presigner,
) chi.Router {
	t.Helper()
	hub := mdp.NewHub(store, presence, discardLogger())
	t.Cleanup(hub.Close)
	r := chi.NewRouter()
	mdp.RegisterProtocolRoutes(r, hub, store, presigner, discardLogger())
	return r
}

func TestConnectRejectsMissingAndUnknownKey(t *testing.T) {
	db, _ := dbtest.Open(t)
	store, presence := newStore(db)
	router := agentRouter(t, store, presence, fakePresigner{})

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
	point, err := store.Create(ctx, pointMutation("Melbourne", []string{"10.0.0.0/8"}), "worker-key")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	router := agentRouter(t, store, presence, fakePresigner{})
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
	point, err := store.Create(ctx, pointMutation("Melbourne", []string{"10.0.0.0/8"}), "worker-key")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	router := agentRouter(t, store, presence, fakePresigner{})
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

func TestDownloadURLMintsPresignedURLForWorker(t *testing.T) {
	db, ctx := dbtest.Open(t)
	store, presence := newStore(db)
	sha := strings.Repeat("a", 64)
	pkg := seedAvailablePackage(t, db, ctx, "Chrome", sha, 4096)
	if _, err := store.Create(ctx, pointMutation("Melbourne", nil), "worker-key"); err != nil {
		t.Fatalf("Create: %v", err)
	}
	const presigned = "https://storage.example/packages/1/Chrome.pkg?cap=signed"
	router := agentRouter(t, store, presence, fakePresigner{url: presigned})

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
