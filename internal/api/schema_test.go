package api

import (
	"reflect"
	"testing"
)

func TestProtectedOperationsDeclareAuthentication(t *testing.T) {
	api := BuildSchemaAPI("test")
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

func TestTransportSpecificRoutesShareTheAppSchema(t *testing.T) {
	t.Parallel()
	doc := BuildSchemaAPI("test").OpenAPI()

	liveQueryStream := doc.Paths["/api/live-queries/{id}/stream"]
	if liveQueryStream == nil || liveQueryStream.Get == nil {
		t.Fatal("live query stream is missing")
	}
	packageInstaller := doc.Paths["/api/munki/package-installers/{id}"]
	if packageInstaller == nil || packageInstaller.Put == nil {
		t.Fatal("package installer finalization is missing")
	}
	multipartComplete := doc.Paths["/api/munki/package-installers/{id}/multipart/complete"]
	if multipartComplete == nil || multipartComplete.Post == nil {
		t.Fatal("multipart completion is missing")
	}
}
