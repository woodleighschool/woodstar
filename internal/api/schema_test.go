package api

import (
	"reflect"
	"testing"

	"github.com/danielgtaylor/huma/v2"
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
	if doc.Paths["/api/hosts"].Delete.Responses["403"] == nil {
		t.Fatal("ordinary mutation does not declare a 403 response")
	}
	if doc.Paths["/api/agent-secrets"].Get.Responses["403"] == nil {
		t.Fatal("sensitive read does not declare a 403 response")
	}

	if doc.Paths["/api/setup"] != nil {
		t.Fatal("removed setup endpoint is still registered")
	}
	session := doc.Paths["/api/session"]
	if len(session.Get.Security) != 0 {
		t.Fatalf("GET session security = %#v, want optional authentication", session.Get.Security)
	}
	if len(session.Post.Security) != 0 {
		t.Fatalf("POST session security = %#v, want public operation", session.Post.Security)
	}
	if session.Post.Responses["429"] == nil {
		t.Fatal("password login does not declare rate limiting")
	}
	retryAfter := session.Post.Responses["429"].Headers["Retry-After"]
	if retryAfter == nil || !retryAfter.Required ||
		retryAfter.Schema == nil || retryAfter.Schema.Type != "integer" {
		t.Fatalf("password login Retry-After header = %#v, want integer seconds", retryAfter)
	}
	if !reflect.DeepEqual(session.Delete.Security, want) {
		t.Fatalf("DELETE session security = %#v, want %#v", session.Delete.Security, want)
	}
}

func TestAdminRoutesUseResourceMethods(t *testing.T) {
	t.Parallel()
	doc := BuildSchemaAPI("test").OpenAPI()

	session := doc.Paths["/api/session"]
	if session == nil || session.Get == nil || session.Post == nil || session.Delete == nil {
		t.Fatalf("session methods = %#v, want GET, POST, and DELETE", session)
	}
	for _, oldPath := range []string{
		"/api/auth/session",
		"/api/auth/login",
		"/api/auth/logout",
	} {
		if doc.Paths[oldPath] != nil {
			t.Errorf("legacy session path %q is still registered", oldPath)
		}
	}

	liveQuery := doc.Paths["/api/live-queries/{id}"]
	if liveQuery == nil || liveQuery.Delete == nil {
		t.Fatalf("live query methods = %#v, want DELETE", liveQuery)
	}
	if doc.Paths["/api/live-queries/{id}/stop"] != nil {
		t.Error("live query stop action path is still registered")
	}

	for _, collectionPath := range []string{
		"/api/hosts",
		"/api/munki/packages",
		"/api/munki/software",
		"/api/osquery/checks",
		"/api/osquery/reports",
		"/api/santa/configurations",
		"/api/santa/rules",
	} {
		collection := doc.Paths[collectionPath]
		if collection == nil || collection.Delete == nil {
			t.Errorf("collection %q has no DELETE operation", collectionPath)
			continue
		}
		if collection.Delete.RequestBody != nil {
			t.Errorf("collection %q DELETE has a request body", collectionPath)
		}
		var idsParameter *huma.Param
		for _, parameter := range collection.Delete.Parameters {
			if parameter.In == "query" && parameter.Name == "ids" {
				idsParameter = parameter
				break
			}
		}
		if idsParameter == nil || !idsParameter.Required || idsParameter.Schema == nil ||
			idsParameter.Schema.MinItems == nil || *idsParameter.Schema.MinItems != 1 {
			t.Errorf(
				"collection %q DELETE ids parameter = %#v, want required non-empty query array",
				collectionPath,
				idsParameter,
			)
		}
		if doc.Paths[collectionPath+"/bulk-delete"] != nil {
			t.Errorf("collection %q still has a bulk-delete action path", collectionPath)
		}
	}

	if doc.Paths["/api/hosts/{id}/osquery/reports/{report_id}"] != nil {
		t.Error("unused host report results path is still registered")
	}
	if packageItem := doc.Paths["/api/munki/packages/{id}"]; packageItem == nil ||
		packageItem.Get == nil || packageItem.Put == nil || packageItem.Delete != nil {
		t.Fatalf("package item methods = %#v, want GET and PUT only", packageItem)
	}
}

func TestLiveQueryStreamUsesTheAppSchema(t *testing.T) {
	t.Parallel()
	doc := BuildSchemaAPI("test").OpenAPI()

	liveQueryStream := doc.Paths["/api/live-queries/{id}/stream"]
	if liveQueryStream == nil || liveQueryStream.Get == nil {
		t.Fatal("live query stream is missing")
	}
	streamSchema := liveQueryStream.Get.Responses["200"].Content["text/event-stream"].Schema
	if streamSchema == nil || streamSchema.Type == huma.TypeArray || len(streamSchema.OneOf) != 3 {
		t.Fatalf("live query stream schema = %#v, want three yielded payload variants", streamSchema)
	}
}

func TestMunkiUploadStrategySchema(t *testing.T) {
	t.Parallel()
	doc := BuildSchemaAPI("test").OpenAPI()

	packageInstaller := doc.Paths["/api/munki/package-installers/{id}"]
	if packageInstaller == nil || packageInstaller.Put == nil {
		t.Fatal("package installer finalization is missing")
	}
	multipartComplete := doc.Paths["/api/munki/package-installers/{id}/multipart/complete"]
	if multipartComplete == nil || multipartComplete.Post == nil {
		t.Fatal("multipart completion is missing")
	}

	packageTarget := doc.Components.Schemas.Map()["MunkiPackageInstallerUploadTarget"]
	if packageTarget == nil {
		t.Fatal("MunkiPackageInstallerUploadTarget schema is missing")
	}
	upload := packageTarget.Properties["upload"]
	if upload == nil || len(upload.OneOf) != 2 || upload.Discriminator == nil ||
		upload.Discriminator.PropertyName != "strategy" ||
		!reflect.DeepEqual(upload.Discriminator.Mapping, map[string]string{
			"direct-put": "#/components/schemas/MunkiDirectUploadAction",
			"multipart":  "#/components/schemas/MunkiMultipartUploadAction",
		}) {
		t.Fatalf("upload action = %#v, want a strategy-discriminated two-variant union", upload)
	}

	direct := doc.Components.Schemas.Map()["MunkiDirectUploadAction"]
	if direct == nil || !reflect.DeepEqual(direct.Properties["strategy"].Enum, []any{"direct-put"}) ||
		direct.Properties["url"] == nil ||
		!reflect.DeepEqual(direct.Properties["method"].Enum, []any{"PUT"}) {
		t.Fatalf("direct upload action = %#v, want strategy, URL, and method", direct)
	}
	multipart := doc.Components.Schemas.Map()["MunkiMultipartUploadAction"]
	if multipart == nil || !reflect.DeepEqual(multipart.Properties["strategy"].Enum, []any{"multipart"}) ||
		multipart.Properties["url"] != nil || multipart.Properties["method"] != nil {
		t.Fatalf("multipart upload action = %#v, want only the multipart strategy", multipart)
	}
	multipartPart := doc.Components.Schemas.Map()["MunkiMultipartPartTarget"]
	if multipartPart == nil || multipartPart.Properties["upload_url"] == nil ||
		!reflect.DeepEqual(multipartPart.Properties["method"].Enum, []any{"PUT"}) {
		t.Fatalf("multipart part target = %#v, want upload URL and PUT method", multipartPart)
	}

	for path, wantRef := range map[string]string{
		"/api/munki/package-installers":      "#/components/schemas/MunkiPackageInstallerUploadTarget",
		"/api/munki/client-resources/banner": "#/components/schemas/MunkiDirectUploadTarget",
		"/api/munki/software/{id}/icon":      "#/components/schemas/MunkiDirectUploadTarget",
	} {
		operation := doc.Paths[path].Post
		got := operation.Responses["201"].Content["application/json"].Schema.Ref
		if got != wantRef {
			t.Errorf("POST %s response schema = %q, want %q", path, got, wantRef)
		}
	}
}
