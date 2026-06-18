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

// agentRouter wires the MDP protocol routes with a hub whose presence the store
// reads. The hub is closed when the test ends.
func agentRouter(t *testing.T, store *mdp.Store, presigner storage.Presigner) (chi.Router, *mdp.Hub) {
	t.Helper()
	hub := mdp.NewHub(store, discardLogger())
	store.SetPresence(hub)
	t.Cleanup(hub.Close)
	r := chi.NewRouter()
	mdp.RegisterProtocolRoutes(r, hub, store, presigner, discardLogger())
	return r, hub
}

func TestConnectRejectsMissingAndUnknownKey(t *testing.T) {
	db, _ := dbtest.Open(t)
	router, _ := agentRouter(t, mdp.NewStore(db), fakePresigner{})

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

func TestConnectDeliversHelloAndRecordsState(t *testing.T) {
	db, ctx := dbtest.Open(t)
	store := mdp.NewStore(db)
	sha := strings.Repeat("a", 64)
	pkg := seedAvailablePackage(t, db, ctx, "Chrome", sha, 4096)
	point, err := store.Create(ctx, pointMutation("Melbourne", []string{"10.0.0.0/8"}), "worker-key")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	router, hub := agentRouter(t, store, fakePresigner{})
	srv := httptest.NewServer(router)
	defer srv.Close()

	ws, _, err := websocket.Dial(ctx, wsURL(srv.URL), &websocket.DialOptions{
		HTTPHeader: http.Header{"Authorization": {"Bearer worker-key"}},
	})
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer ws.Close(websocket.StatusNormalClosure, "")

	// hello carries identity and the full desired set.
	var hello struct {
		Type              string `json:"type"`
		DistributionPoint struct {
			ID int64 `json:"id"`
		} `json:"distribution_point"`
		Packages []struct {
			PackageID int64 `json:"package_id"`
		} `json:"packages"`
	}
	readJSON(t, ctx, ws, &hello)
	if hello.Type != "hello" || hello.DistributionPoint.ID != point.ID {
		t.Fatalf("hello = %+v, want hello for point %d", hello, point.ID)
	}
	if len(hello.Packages) != 1 || hello.Packages[0].PackageID != pkg {
		t.Fatalf("hello packages = %+v, want [%d]", hello.Packages, pkg)
	}

	// The connection makes the point online for the resolver.
	eventually(t, func() bool { return hub.Online(point.ID) })

	// A reported state is byte-verified and persisted server-side.
	state := `{"type":"state","packages":[{"package_id":` +
		strconv.FormatInt(pkg, 10) + `,"sha256":"` + sha + `","status":"current"}]}`
	if err := ws.Write(ctx, websocket.MessageText, []byte(state)); err != nil {
		t.Fatalf("write state: %v", err)
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
	store := mdp.NewStore(db)
	point, err := store.Create(ctx, pointMutation("Melbourne", []string{"10.0.0.0/8"}), "worker-key")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	router, hub := agentRouter(t, store, fakePresigner{})
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
	eventually(t, func() bool { return hub.Online(point.ID) })

	if err := ws.Write(ctx, websocket.MessageText, []byte(`{"type":"not-state"}`)); err != nil {
		t.Fatalf("write unexpected message: %v", err)
	}
	eventually(t, func() bool { return !hub.Online(point.ID) })
}

func TestContentRedirectsToPresignedURL(t *testing.T) {
	db, ctx := dbtest.Open(t)
	store := mdp.NewStore(db)
	pkg := seedAvailablePackage(t, db, ctx, "Chrome", strings.Repeat("a", 64), 4096)
	if _, err := store.Create(ctx, pointMutation("Melbourne", nil), "worker-key"); err != nil {
		t.Fatalf("Create: %v", err)
	}
	router, _ := agentRouter(t, store, fakePresigner{url: "https://signed.example/object"})

	req := httptest.NewRequest(http.MethodGet,
		"/api/munki/distribution/packages/"+strconv.FormatInt(pkg, 10)+"/content", nil)
	req.Header.Set("Authorization", "Bearer worker-key")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("status = %d, want 302; body = %q", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Location"); got != "https://signed.example/object" {
		t.Fatalf("Location = %q, want presigned URL", got)
	}

	missing := httptest.NewRequest(http.MethodGet, "/api/munki/distribution/packages/999999/content", nil)
	missing.Header.Set("Authorization", "Bearer worker-key")
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, missing)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("missing package status = %d, want 404", rec.Code)
	}
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
