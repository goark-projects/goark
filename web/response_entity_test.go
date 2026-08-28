package web_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	arkjson "goark.dev/arkarta/json"
	servletnethttp "goark.dev/arkarta/servlet/nethttp"
	arkweb "goark.dev/arkarta/web"
	"goark.dev/goark/web"
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
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
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
