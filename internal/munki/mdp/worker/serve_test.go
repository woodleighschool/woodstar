package worker

import (
	"crypto/sha256"
	"encoding/hex"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/woodleighschool/woodstar/internal/munki/mdp/grant"
)

func discardLogger() *slog.Logger {
	return slog.New(slog.DiscardHandler)
}

func sha256Hex(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func TestServeNodeAppliesGrantAndIntegrityChecks(t *testing.T) {
	dir := t.TempDir()
	mirror, err := loadMirror(dir)
	if err != nil {
		t.Fatalf("loadMirror: %v", err)
	}
	content := []byte("installer-bytes-0123456789")
	sha := sha256Hex(content)
	size := int64(len(content))
	if err := os.WriteFile(mirror.localPath(7, "Chrome.pkg"), content, 0o600); err != nil {
		t.Fatalf("write mirror file: %v", err)
	}
	mirror.put(7, packageState{Filename: "Chrome.pkg", SHA256: sha, SizeBytes: size})
	// A package known to the mirror but whose file is gone.
	mirror.put(9, packageState{Filename: "Gone.pkg", SHA256: sha, SizeBytes: size})

	key := []byte("dp-key")
	handler := (&server{mirror: mirror, key: key, logger: discardLogger()}).handler()
	now := time.Now()
	token := func(packageID int64, sha string, size int64, exp time.Time) string {
		tok, err := grant.Sign(key, grant.Claims{
			Exp:       exp.Unix(),
			PackageID: packageID,
			SHA256:    sha,
			SizeBytes: size,
		})
		if err != nil {
			t.Fatalf("sign grant: %v", err)
		}
		return tok
	}

	t.Run("valid full", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(
			http.MethodGet,
			"/munki-distribution/packages/7?cap="+token(7, sha, size, now.Add(time.Minute)),
			nil,
		)
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rec.Code)
		}
		if rec.Body.String() != string(content) {
			t.Fatalf("body = %q, want full installer", rec.Body.String())
		}
	})

	t.Run("range", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(
			http.MethodGet,
			"/munki-distribution/packages/7?cap="+token(7, sha, size, now.Add(time.Minute)),
			nil,
		)
		req.Header.Set("Range", "bytes=0-3")
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusPartialContent {
			t.Fatalf("status = %d, want 206", rec.Code)
		}
		if rec.Body.String() != "inst" {
			t.Fatalf("range body = %q, want inst", rec.Body.String())
		}
	})

	cases := []struct {
		name string
		cap  string
		want int
	}{
		{name: "invalid grant", cap: "garbage", want: http.StatusUnauthorized},
		{name: "expired grant", cap: token(7, sha, size, now.Add(-time.Second)), want: http.StatusGone},
		{name: "wrong package", cap: token(8, sha, size, now.Add(time.Minute)), want: http.StatusUnauthorized},
		{
			name: "stale sha",
			cap:  token(7, sha256Hex([]byte("other")), size, now.Add(time.Minute)),
			want: http.StatusConflict,
		},
		{name: "stale size", cap: token(7, sha, size+1, now.Add(time.Minute)), want: http.StatusConflict},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/munki-distribution/packages/7?cap="+tc.cap, nil)
			handler.ServeHTTP(rec, req)
			if rec.Code != tc.want {
				t.Fatalf("status = %d, want %d", rec.Code, tc.want)
			}
		})
	}

	t.Run("missing file", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(
			http.MethodGet,
			"/munki-distribution/packages/9?cap="+token(9, sha, size, now.Add(time.Minute)),
			nil,
		)
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want 404", rec.Code)
		}
	})
}
