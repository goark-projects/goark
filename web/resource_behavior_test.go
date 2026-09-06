package web_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	arkjson "goark.dev/arkarta/json"
	servletnethttp "goark.dev/arkarta/servlet/nethttp"
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

func TestAttachmentStreamsDownload(t *testing.T) {
	t.Parallel()

	router := arkweb.NewRouter()
	if err := router.GET("/reports/today", arkweb.HandlerFunc(func(_ *arkweb.Context) (arkweb.Result, error) {
		return web.Attachment("reports/today.csv", strings.NewReader("id,name\n1,goark\n"),
			web.WithDownloadContentType("text/csv"),
			web.WithDownloadContentLength(16),
		), nil
	})); err != nil {
		t.Fatalf("GET failed: %v", err)
	}

	recorder := httptest.NewRecorder()
	servletnethttp.Handler(router).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/reports/today", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if recorder.Body.String() != "id,name\n1,goark\n" {
		t.Fatalf("body = %q", recorder.Body.String())
	}
	if got := recorder.Header().Get("Content-Type"); got != "text/csv" {
		t.Fatalf("Content-Type = %q, want text/csv", got)
	}
	if got := recorder.Header().Get("Content-Length"); got != "16" {
		t.Fatalf("Content-Length = %q, want 16", got)
	}
	if got := recorder.Header().Get("Content-Disposition"); got != `attachment; filename=today.csv` {
		t.Fatalf("Content-Disposition = %q", got)
	}
}

func TestAttachmentDoesNotReadBodyForHead(t *testing.T) {
	t.Parallel()

	router := arkweb.NewRouter()
	if err := router.GET("/reports/today", arkweb.HandlerFunc(func(_ *arkweb.Context) (arkweb.Result, error) {
		return web.Attachment("today.csv", failReadDownload{}), nil
	})); err != nil {
		t.Fatalf("GET failed: %v", err)
	}

	recorder := httptest.NewRecorder()
	servletnethttp.Handler(router).ServeHTTP(recorder, httptest.NewRequest(http.MethodHead, "/reports/today", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if recorder.Body.Len() != 0 {
		t.Fatalf("body length = %d, want 0", recorder.Body.Len())
	}
}

type failReadDownload struct{}

func (failReadDownload) Read([]byte) (int, error) {
	return 0, errors.New("download body must not be read")
}

func TestRedirectWritesLocationAndStatus(t *testing.T) {
	t.Parallel()

	registry := web.NewRegistry()
	if err := registry.GET("/accounts", arkweb.HandlerFunc(func(*arkweb.Context) (arkweb.Result, error) {
		return web.SeeOther("/signin", web.WithRedirectHeader("X-Redirect", "yes")), nil
	})); err != nil {
		t.Fatalf("GET failed: %v", err)
	}
	router, err := registry.Router()
	if err != nil {
		t.Fatalf("Router failed: %v", err)
	}

	recorder := httptest.NewRecorder()
	servletnethttp.Handler(router).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/accounts", nil))
	if recorder.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want 303", recorder.Code)
	}
	if got := recorder.Header().Get("Location"); got != "/signin" {
		t.Fatalf("Location = %q, want /signin", got)
	}
	if got := recorder.Header().Get("X-Redirect"); got != "yes" {
		t.Fatalf("X-Redirect = %q, want yes", got)
	}
	if recorder.Body.Len() != 0 {
		t.Fatalf("body = %q, want empty", recorder.Body.String())
	}
}

func TestRedirectRejectsInvalidLocation(t *testing.T) {
	t.Parallel()

	registry := web.NewRegistry()
	if err := registry.GET("/bad", arkweb.HandlerFunc(func(*arkweb.Context) (arkweb.Result, error) {
		return web.Redirect("/signin\r\nX-Bad: yes"), nil
	})); err != nil {
		t.Fatalf("GET failed: %v", err)
	}
	router, err := registry.Router()
	if err != nil {
		t.Fatalf("Router failed: %v", err)
	}

	recorder := httptest.NewRecorder()
	servletnethttp.Handler(router).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/bad", nil))
	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", recorder.Code)
	}
	if got := recorder.Header().Get("Location"); got != "" {
		t.Fatalf("Location = %q, want empty", got)
	}
}

func TestRegistryBuildsRouter(t *testing.T) {
	registry := web.NewRegistry()
	if err := registry.GET("/healthz", arkweb.HandlerFunc(func(_ *arkweb.Context) (arkweb.Result, error) {
		return arkweb.JSON(http.StatusOK, map[string]string{"status": "UP"}), nil
	})); err != nil {
		t.Fatalf("GET failed: %v", err)
	}

	router, err := registry.Router()
	if err != nil {
		t.Fatalf("Router failed: %v", err)
	}
	recorder := httptest.NewRecorder()
	servletnethttp.Handler(router).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
}

func TestRegistrySupportsHeadOptionsAndTraceHelpers(t *testing.T) {
	registry := web.NewRegistry()
	if err := registry.HEAD("/healthz", arkweb.HandlerFunc(func(_ *arkweb.Context) (arkweb.Result, error) {
		return arkweb.Text(http.StatusOK, "UP"), nil
	})); err != nil {
		t.Fatalf("HEAD failed: %v", err)
	}
	if err := registry.OPTIONS("/healthz", arkweb.HandlerFunc(func(_ *arkweb.Context) (arkweb.Result, error) {
		return arkweb.NoContent(), nil
	})); err != nil {
		t.Fatalf("OPTIONS failed: %v", err)
	}
	if err := registry.TRACE("/healthz", arkweb.HandlerFunc(func(_ *arkweb.Context) (arkweb.Result, error) {
		return arkweb.NoContent(), nil
	})); err != nil {
		t.Fatalf("TRACE failed: %v", err)
	}

	routes := registry.Routes()
	if len(routes) != 3 {
		t.Fatalf("route count = %d, want 3", len(routes))
	}
	if routes[0].Method != http.MethodHead || routes[1].Method != http.MethodOptions || routes[2].Method != http.MethodTrace {
		t.Fatalf("methods = %s/%s/%s, want HEAD/OPTIONS/TRACE", routes[0].Method, routes[1].Method, routes[2].Method)
	}
}
