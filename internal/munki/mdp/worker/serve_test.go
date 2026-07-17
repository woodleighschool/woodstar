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
	const installerItemLocation = "packages/7/installer/Chrome.pkg"
	token := func(packageID int64, location string, sha string, size int64, exp time.Time) string {
		tok, err := grant.Sign(key, grant.Claims{
			Exp:                   exp.Unix(),
			PackageID:             packageID,
			InstallerItemLocation: location,
			SHA256:                sha,
			SizeBytes:             size,
		})
		if err != nil {
			t.Fatalf("sign grant: %v", err)
		}
		return tok
	}

	cases := []struct {
		name string
		cap  string
		want int
	}{
		{name: "invalid grant", cap: "garbage", want: http.StatusUnauthorized},
		{
			name: "expired grant",
			cap:  token(7, installerItemLocation, sha, size, now.Add(-time.Second)),
			want: http.StatusGone,
		},
		{
			name: "wrong path",
			cap:  token(7, "packages/8/installer/Chrome.pkg", sha, size, now.Add(time.Minute)),
			want: http.StatusUnauthorized,
		},
		{
			name: "wrong package",
			cap:  token(8, installerItemLocation, sha, size, now.Add(time.Minute)),
			want: http.StatusNotFound,
		},
		{
			name: "stale sha",
			cap:  token(7, installerItemLocation, sha256Hex([]byte("other")), size, now.Add(time.Minute)),
			want: http.StatusConflict,
		},
		{
			name: "stale size",
			cap:  token(7, installerItemLocation, sha, size+1, now.Add(time.Minute)),
			want: http.StatusConflict,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/munki/pkgs/"+installerItemLocation+"?cap="+tc.cap, nil)
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
			"/munki/pkgs/packages/9/installer/Gone.pkg?cap="+
				token(9, "packages/9/installer/Gone.pkg", sha, size, now.Add(time.Minute)),
			nil,
		)
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want 404", rec.Code)
		}
	})
}
