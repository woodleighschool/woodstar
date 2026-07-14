package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"

	"github.com/woodleighschool/woodstar/internal/database/dbtest"
	"github.com/woodleighschool/woodstar/internal/santa/references"
)

func TestSoftwareSantaReferenceEndpoint(t *testing.T) {
	db, ctx := dbtest.Open(t)
	router := chi.NewRouter()
	api := humachi.New(router, huma.DefaultConfig("test", "test"))
	registerSoftwareSantaReference(api, references.NewStore(db), discardLogger())

	var titleID int64
	if err := db.Pool().QueryRow(ctx, `
		INSERT INTO software_titles (name, source, bundle_identifier)
		VALUES ('Reference Endpoint', 'apps', 'com.example.reference-endpoint')
		RETURNING id
	`).Scan(&titleID); err != nil {
		t.Fatalf("insert software title: %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(
		context.Background(),
		http.MethodGet,
		fmt.Sprintf("/api/software/%d/santa", titleID),
		http.NoBody,
	)
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %q", rec.Code, http.StatusOK, rec.Body.String())
	}
	var body references.SoftwareReference
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode software reference: %v", err)
	}
	if body.ExecutionCount != 0 || body.BlockCount != 0 {
		t.Fatalf("software reference = %+v, want empty counts", body)
	}
}
