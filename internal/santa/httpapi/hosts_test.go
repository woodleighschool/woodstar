package httpapi

import (
	"log/slog"
	"net/http"
	"strconv"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
)

func TestHostSantaRulesDeclaresConflictResponse(t *testing.T) {
	humaAPI := humachi.New(chi.NewRouter(), huma.DefaultConfig("test", "test"))
	registerHostSantaRules(humaAPI, nil, nil, slog.New(slog.DiscardHandler))

	operation := humaAPI.OpenAPI().Paths["/api/hosts/{id}/santa/rules"].Get
	status := strconv.Itoa(http.StatusConflict)
	if operation.Responses[status] == nil {
		t.Fatalf("response %s is not declared", status)
	}
}
