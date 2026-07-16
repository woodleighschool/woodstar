package api

import (
	"log/slog"
	"reflect"
	"testing"
)

func TestProtectedOperationsDeclareAuthentication(t *testing.T) {
	api := BuildSchemaAPI("test", &Dependencies{Logger: slog.New(slog.DiscardHandler)})
	doc := api.OpenAPI()

	if doc.Components.SecuritySchemes["cookieAuth"] == nil {
		t.Fatal("cookieAuth security scheme is missing")
	}
	bearer := doc.Components.SecuritySchemes["bearerAuth"]
	if bearer == nil || bearer.Type != "http" || bearer.Scheme != "bearer" {
		t.Fatalf("bearerAuth = %#v, want HTTP bearer scheme", bearer)
	}

	hosts := doc.Paths["/api/hosts"].Get
	want := []map[string][]string{{"cookieAuth": {}}, {"bearerAuth": {}}}
	if !reflect.DeepEqual(hosts.Security, want) {
		t.Fatalf("hosts security = %#v, want %#v", hosts.Security, want)
	}
	if hosts.Responses["401"] == nil {
		t.Fatal("protected operation does not declare a 401 response")
	}
	if doc.Paths["/api/hosts/bulk-delete"].Post.Responses["403"] == nil {
		t.Fatal("ordinary mutation does not declare a 403 response")
	}
	if doc.Paths["/api/agent-secrets"].Get.Responses["403"] == nil {
		t.Fatal("sensitive read does not declare a 403 response")
	}

	if setup := doc.Paths["/api/setup"].Post; len(setup.Security) != 0 {
		t.Fatalf("setup security = %#v, want public operation", setup.Security)
	}
	if session := doc.Paths["/api/auth/session"].Get; len(session.Security) != 0 {
		t.Fatalf("session security = %#v, want optional authentication", session.Security)
	}
}
