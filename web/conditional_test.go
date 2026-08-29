package web_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	arkjson "goark.dev/arkarta/json"
	arkweb "goark.dev/arkarta/web"
	"goark.dev/goark/web"
	webtest "goark.dev/goark/web/test"
)

func TestCheckNotModifiedHonorsIfNoneMatch(t *testing.T) {
	t.Parallel()

	client := newConditionalClient(t, "job-1", fixedModifiedTime())
	response := client.Perform(
		t,
		http.MethodGet,
		"/jobs/1",
		webtest.WithHeader("Accept", arkjson.ContentType),
		webtest.WithHeader("If-None-Match", `"job-1"`),
	)

	response.ExpectStatus(t, http.StatusNotModified).
		ExpectHeader(t, "ETag", `"job-1"`).
		ExpectHeader(t, "Last-Modified", "Sat, 29 Aug 2026 00:30:00 GMT").
		ExpectBody(t, "")
}

func TestCheckNotModifiedUsesWeakETagComparison(t *testing.T) {
	t.Parallel()

	client := newConditionalClient(t, `W/"job-1"`, fixedModifiedTime())
	response := client.Perform(
		t,
		http.MethodHead,
		"/jobs/1",
		webtest.WithHeader("If-None-Match", `"job-1"`),
	)

	response.ExpectStatus(t, http.StatusNotModified).
		ExpectHeader(t, "ETag", `W/"job-1"`).
		ExpectBody(t, "")
}

func TestCheckNotModifiedHonorsIfModifiedSince(t *testing.T) {
	t.Parallel()

	client := newConditionalClient(t, "", fixedModifiedTime())
	response := client.Perform(
		t,
		http.MethodGet,
		"/jobs/1",
		webtest.WithHeader("Accept", arkjson.ContentType),
		webtest.WithHeader("If-Modified-Since", "Sat, 29 Aug 2026 00:30:00 GMT"),
	)

	response.ExpectStatus(t, http.StatusNotModified).
		ExpectHeader(t, "Last-Modified", "Sat, 29 Aug 2026 00:30:00 GMT").
		ExpectBody(t, "")
}

func TestCheckNotModifiedPrefersIfNoneMatch(t *testing.T) {
	t.Parallel()

	client := newConditionalClient(t, "job-1", fixedModifiedTime())
	response := client.Perform(
		t,
		http.MethodGet,
		"/jobs/1",
		webtest.WithHeader("Accept", arkjson.ContentType),
		webtest.WithHeader("If-None-Match", `"other"`),
		webtest.WithHeader("If-Modified-Since", "Sat, 29 Aug 2026 00:30:00 GMT"),
	)

	response.ExpectStatus(t, http.StatusOK).
		ExpectHeader(t, "ETag", `"job-1"`).
		ExpectHeader(t, "Last-Modified", "Sat, 29 Aug 2026 00:30:00 GMT")
	if response.BodyString() == "" {
		t.Fatalf("body is empty, want resource body")
	}
}

func TestCheckNotModifiedIgnoresUnsafeMethods(t *testing.T) {
	t.Parallel()

	registry := web.NewRegistry()
	if err := registry.POST("/jobs/1", arkweb.HandlerFunc(func(ctx *arkweb.Context) (arkweb.Result, error) {
		if web.CheckNotModified(ctx, "job-1", fixedModifiedTime()) {
			return nil, nil
		}
		return web.NoBody(http.StatusAccepted).WithETag("job-1"), nil
	})); err != nil {
		t.Fatalf("POST failed: %v", err)
	}
	client, err := webtest.NewRegistry(t.Context(), registry, web.DeploymentSpec{})
	client = webtest.Must(t, client, err)
	t.Cleanup(func() {
		if err := client.Close(context.Background()); err != nil {
			t.Fatalf("close failed: %v", err)
		}
	})

	response := client.Perform(
		t,
		http.MethodPost,
		"/jobs/1",
		webtest.WithHeader("If-None-Match", `"job-1"`),
	)

	response.ExpectStatus(t, http.StatusAccepted).
		ExpectHeader(t, "ETag", `"job-1"`)
}

func newConditionalClient(t *testing.T, etag string, lastModified time.Time) *webtest.Client {
	t.Helper()

	registry := web.NewRegistry()
	if err := registry.GET("/jobs/1", arkweb.HandlerFunc(func(ctx *arkweb.Context) (arkweb.Result, error) {
		if web.CheckNotModified(ctx, etag, lastModified) {
			return nil, nil
		}
		return web.OK(map[string]string{"id": "1"}).
			WithETag(etag).
			WithLastModified(lastModified), nil
	})); err != nil {
		t.Fatalf("GET failed: %v", err)
	}
	client, err := webtest.NewRegistry(t.Context(), registry, web.DeploymentSpec{})
	client = webtest.Must(t, client, err)
	t.Cleanup(func() {
		if err := client.Close(context.Background()); err != nil {
			t.Fatalf("close failed: %v", err)
		}
	})
	return client
}

func fixedModifiedTime() time.Time {
	return time.Date(2026, time.August, 29, 8, 30, 0, 500, time.FixedZone("CST", 8*60*60))
}
