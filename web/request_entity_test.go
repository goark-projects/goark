package web_test

import (
	"net/http"
	"testing"

	"goark.dev/goark/web"
)

func TestRequestEntityCapturesMetadataAndBody(t *testing.T) {
	t.Parallel()

	headers := http.Header{
		"X-Trace-Id": {"trace-1"},
	}
	entity := web.NewRequestEntity(web.RequestMetadata{
		Method:        "post",
		URL:           "https://api.example.com/jobs/42?trace=1",
		RequestURI:    "/jobs/42",
		Path:          "/jobs/42",
		Headers:       headers,
		ContentLength: 15,
	}, map[string]string{"name": "sync"}, true)
	headers.Set("X-Trace-Id", "mutated")

	if entity.Method() != http.MethodPost {
		t.Fatalf("method = %q, want POST", entity.Method())
	}
	if entity.URL() != "https://api.example.com/jobs/42?trace=1" {
		t.Fatalf("url = %q, want full request URL", entity.URL())
	}
	if entity.RequestURI() != "/jobs/42" {
		t.Fatalf("request uri = %q, want /jobs/42", entity.RequestURI())
	}
	if entity.Path() != "/jobs/42" {
		t.Fatalf("path = %q, want /jobs/42", entity.Path())
	}
	if entity.ContentLength() != 15 {
		t.Fatalf("content length = %d, want 15", entity.ContentLength())
	}
	if got, ok := entity.HeaderValue("x-trace-id"); !ok || got != "trace-1" {
		t.Fatalf("header = %q ok=%t, want trace-1", got, ok)
	}
	if values := entity.HeaderValues("X-Trace-Id"); len(values) != 1 || values[0] != "trace-1" {
		t.Fatalf("header values = %#v, want trace-1", values)
	}
	if body, ok := entity.Body(); !ok || body["name"] != "sync" {
		t.Fatalf("body = %#v ok=%t, want explicit body", body, ok)
	}
}

func TestRequestEntityHeadersReturnsImmutableSnapshot(t *testing.T) {
	t.Parallel()

	entity := web.NewRequestEntity(web.RequestMetadata{
		Method:  http.MethodGet,
		Headers: http.Header{"X-Role": {"admin"}},
	}, struct{}{}, false)

	headers := entity.Headers()
	headers.Set("X-Role", "operator")

	if got, ok := entity.HeaderValue("X-Role"); !ok || got != "admin" {
		t.Fatalf("header = %q ok=%t, want immutable snapshot", got, ok)
	}
	if _, ok := entity.Body(); ok {
		t.Fatal("empty request entity should not expose a body")
	}
	if entity.HasBody() {
		t.Fatal("empty request entity should report no body")
	}
}
