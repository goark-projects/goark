package web_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	arkjson "goark.dev/arkarta/json"
	servletnethttp "goark.dev/arkarta/servlet/nethttp"
	arkweb "goark.dev/arkarta/web"
	"goark.dev/goark/web"
	"goark.dev/goark/web/message"
	"goark.dev/goark/web/uri"
)

func TestResponseEntityWritesStatusHeadersAndJSONBody(t *testing.T) {
	registry := web.NewRegistry()
	if err := registry.GET("/jobs/1", arkweb.HandlerFunc(func(_ *arkweb.Context) (arkweb.Result, error) {
		return web.Status(http.StatusAccepted, map[string]string{"state": "queued"}).
			WithHeader("X-Job-ID", "1"), nil
	})); err != nil {
		t.Fatalf("GET failed: %v", err)
	}

	router, err := registry.Router()
	if err != nil {
		t.Fatalf("Router failed: %v", err)
	}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/jobs/1", nil)
	request.Header.Set("Accept", arkjson.ContentType)
	servletnethttp.Handler(router).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusAccepted)
	}
	if got := recorder.Header().Get("X-Job-ID"); got != "1" {
		t.Fatalf("X-Job-ID = %q, want 1", got)
	}
	if got := recorder.Header().Get("Content-Type"); got != arkjson.ContentType {
		t.Fatalf("Content-Type = %q, want %s", got, arkjson.ContentType)
	}
	var body map[string]string
	if err := arkjson.Unmarshal(nil, recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("response json invalid: %v", err)
	}
	if body["state"] != "queued" {
		t.Fatalf("state = %q, want queued", body["state"])
	}
}

func TestResponseEntityNoBodyWritesOnlyStatusAndHeaders(t *testing.T) {
	registry := web.NewRegistry()
	if err := registry.DELETE("/jobs/1", arkweb.HandlerFunc(func(_ *arkweb.Context) (arkweb.Result, error) {
		return web.NoBody(http.StatusResetContent).WithHeader("X-Deleted", "true"), nil
	})); err != nil {
		t.Fatalf("DELETE failed: %v", err)
	}

	router, err := registry.Router()
	if err != nil {
		t.Fatalf("Router failed: %v", err)
	}
	recorder := httptest.NewRecorder()
	servletnethttp.Handler(router).ServeHTTP(recorder, httptest.NewRequest(http.MethodDelete, "/jobs/1", nil))

	if recorder.Code != http.StatusResetContent {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusResetContent)
	}
	if got := recorder.Header().Get("X-Deleted"); got != "true" {
		t.Fatalf("X-Deleted = %q, want true", got)
	}
	if recorder.Body.Len() != 0 {
		t.Fatalf("body = %q, want empty", recorder.Body.String())
	}
}

func TestResponseEntitySpringStyleConstructors(t *testing.T) {
	t.Parallel()

	created := web.Created("/jobs/1", map[string]string{"state": "created"})
	if created.StatusCode() != http.StatusCreated {
		t.Fatalf("created status = %d, want 201", created.StatusCode())
	}
	if got := created.Headers().Get("Location"); got != "/jobs/1" {
		t.Fatalf("created Location = %q, want /jobs/1", got)
	}
	if body, ok := created.Body(); !ok || body["state"] != "created" {
		t.Fatalf("created body = %#v ok=%t", body, ok)
	}

	createdNoBody := web.CreatedNoBody("/jobs/1")
	if createdNoBody.StatusCode() != http.StatusCreated {
		t.Fatalf("created no body status = %d, want 201", createdNoBody.StatusCode())
	}
	if _, ok := createdNoBody.Body(); ok {
		t.Fatal("created no body should not expose a body")
	}

	if web.Accepted("queued").StatusCode() != http.StatusAccepted {
		t.Fatal("accepted should use 202")
	}
	if web.AcceptedNoBody().StatusCode() != http.StatusAccepted {
		t.Fatal("accepted no body should use 202")
	}
	if web.NoContent().StatusCode() != http.StatusNoContent {
		t.Fatal("no content should use 204")
	}
	if web.BadRequest("invalid").StatusCode() != http.StatusBadRequest {
		t.Fatal("bad request should use 400")
	}
	if web.NotFound().StatusCode() != http.StatusNotFound {
		t.Fatal("not found should use 404")
	}
}

func TestResponseEntityCreatedFromCurrentRequest(t *testing.T) {
	t.Parallel()

	registry := web.NewRegistry()
	if err := registry.POST("/jobs", arkweb.HandlerFunc(func(ctx *arkweb.Context) (arkweb.Result, error) {
		return web.CreatedFromCurrentRequest(ctx, "/{id}", map[string]string{"id": "a/b"}, map[string]string{"state": "created"})
	})); err != nil {
		t.Fatalf("POST failed: %v", err)
	}

	router, err := registry.Router()
	if err != nil {
		t.Fatalf("Router failed: %v", err)
	}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "https://api.example.com/jobs?draft=true", nil)
	request.Header.Set("Accept", arkjson.ContentType)
	servletnethttp.Handler(router).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201", recorder.Code)
	}
	if got := recorder.Header().Get("Location"); got != "https://api.example.com/jobs/a%2Fb" {
		t.Fatalf("Location = %q, want current request created URI", got)
	}
	var body map[string]string
	if err := arkjson.Unmarshal(nil, recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("response json invalid: %v", err)
	}
	if body["state"] != "created" {
		t.Fatalf("state = %q, want created", body["state"])
	}
}

func TestResponseEntityCreatedFromCurrentRequestReturnsURIErrors(t *testing.T) {
	t.Parallel()

	registry := web.NewRegistry()
	if err := registry.POST("/jobs", arkweb.HandlerFunc(func(ctx *arkweb.Context) (arkweb.Result, error) {
		entity, err := web.CreatedNoBodyFromCurrentRequest(ctx, "/{id}", nil)
		if !errors.Is(err, uri.ErrMissingPathVariable) {
			t.Fatalf("error = %v, want ErrMissingPathVariable", err)
		}
		return entity, err
	})); err != nil {
		t.Fatalf("POST failed: %v", err)
	}

	router, err := registry.Router()
	if err != nil {
		t.Fatalf("Router failed: %v", err)
	}
	recorder := httptest.NewRecorder()
	servletnethttp.Handler(router).ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "https://api.example.com/jobs", nil))

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", recorder.Code)
	}
}

func TestResponseEntityWritesConfiguredMediaType(t *testing.T) {
	t.Parallel()

	registry := web.NewRegistry()
	if err := registry.GET("/entity", arkweb.HandlerFunc(func(_ *arkweb.Context) (arkweb.Result, error) {
		return web.Status(http.StatusOK, "plain").WithContentType(message.MediaTypeTextPlain), nil
	})); err != nil {
		t.Fatalf("GET failed: %v", err)
	}

	router, err := registry.Router()
	if err != nil {
		t.Fatalf("Router failed: %v", err)
	}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/entity", nil)
	request.Header.Set("Accept", "text/plain")
	servletnethttp.Handler(router).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	if recorder.Header().Get("Content-Type") != message.MediaTypeTextPlain {
		t.Fatalf("content type = %q, want text/plain", recorder.Header().Get("Content-Type"))
	}
	if recorder.Body.String() != "plain" {
		t.Fatalf("body = %q, want plain", recorder.Body.String())
	}
}

func TestResponseEntityWritesHTTPMetadata(t *testing.T) {
	t.Parallel()

	modified := time.Date(2026, time.August, 29, 8, 30, 0, 0, time.FixedZone("CST", 8*60*60))
	registry := web.NewRegistry()
	if err := registry.POST("/jobs", arkweb.HandlerFunc(func(_ *arkweb.Context) (arkweb.Result, error) {
		return web.NoBody(http.StatusCreated).
			WithLocation("/jobs/1").
			WithCookie(&http.Cookie{Name: "sid", Value: "abc", HttpOnly: true}).
			WithCacheControl(web.MaxAge(time.Minute).Public().NoTransform()).
			WithETag("job-1").
			WithLastModified(modified).
			WithAllow(http.MethodGet, http.MethodHead, http.MethodGet).
			WithVary("origin", "Accept-Encoding").
			WithContentLength(0), nil
	})); err != nil {
		t.Fatalf("POST failed: %v", err)
	}

	router, err := registry.Router()
	if err != nil {
		t.Fatalf("Router failed: %v", err)
	}
	recorder := httptest.NewRecorder()
	servletnethttp.Handler(router).ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/jobs", nil))

	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201", recorder.Code)
	}
	if got := recorder.Header().Get("Location"); got != "/jobs/1" {
		t.Fatalf("Location = %q, want /jobs/1", got)
	}
	if got := recorder.Header().Get("Set-Cookie"); got != "sid=abc; HttpOnly" {
		t.Fatalf("Set-Cookie = %q, want sid cookie", got)
	}
	if got := recorder.Header().Get("Cache-Control"); got != "max-age=60, public, no-transform" {
		t.Fatalf("Cache-Control = %q, want cache directives", got)
	}
	if got := recorder.Header().Get("ETag"); got != `"job-1"` {
		t.Fatalf("ETag = %q, want quoted tag", got)
	}
	if got := recorder.Header().Get("Last-Modified"); got != "Sat, 29 Aug 2026 00:30:00 GMT" {
		t.Fatalf("Last-Modified = %q, want UTC HTTP date", got)
	}
	if got := recorder.Header().Get("Allow"); got != "GET, HEAD" {
		t.Fatalf("Allow = %q, want deduplicated methods", got)
	}
	if got := recorder.Header().Get("Vary"); got != "Origin, Accept-Encoding" {
		t.Fatalf("Vary = %q, want canonical names", got)
	}
	if got := recorder.Header().Get("Content-Length"); got != "0" {
		t.Fatalf("Content-Length = %q, want 0", got)
	}
}

func TestResponseEntityIgnoresInvalidHTTPMetadata(t *testing.T) {
	t.Parallel()

	entity := web.NoBody(http.StatusNoContent).
		WithHeader("X-Bad", "a\r\nb").
		WithAddedHeader("Bad Header", "ok").
		WithHeaders(http.Header{"X-Good": {"ok"}, "X-Bad-2": {"bad\nvalue"}}).
		WithCookie(nil).
		WithCacheControlValue("private\r\nSet-Cookie: sid=bad").
		WithETag(`bad"tag`).
		WithWeakETag("weak").
		WithLastModified(time.Time{}).
		WithAllow("GET", "BAD METHOD").
		WithVary("Accept", "Bad Header").
		WithContentLength(-1)

	headers := entity.Headers()
	if got := headers.Get("X-Bad"); got != "" {
		t.Fatalf("X-Bad = %q, want empty", got)
	}
	if got := headers.Get("Bad Header"); got != "" {
		t.Fatalf("Bad Header = %q, want empty", got)
	}
	if got := headers.Get("X-Good"); got != "ok" {
		t.Fatalf("X-Good = %q, want ok", got)
	}
	if got := headers.Get("Cache-Control"); got != "" {
		t.Fatalf("Cache-Control = %q, want empty", got)
	}
	if got := headers.Get("ETag"); got != `W/"weak"` {
		t.Fatalf("ETag = %q, want weak tag", got)
	}
	if got := headers.Get("Allow"); got != "GET" {
		t.Fatalf("Allow = %q, want GET", got)
	}
	if got := headers.Get("Vary"); got != "Accept" {
		t.Fatalf("Vary = %q, want Accept", got)
	}
	if got := headers.Get("Content-Length"); got != "" {
		t.Fatalf("Content-Length = %q, want empty", got)
	}
}
