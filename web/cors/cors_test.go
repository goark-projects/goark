package cors_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"goark.dev/arkarta/servlet"
	servletnethttp "goark.dev/arkarta/servlet/nethttp"
	"goark.dev/goark/web/cors"
)

func TestFilterAllowsPreflightRequest(t *testing.T) {
	t.Parallel()

	filter, err := cors.New(cors.Config{
		AllowedOrigins:   []string{"https://admin.example.com"},
		AllowedMethods:   []string{http.MethodGet, http.MethodPost},
		AllowedHeaders:   []string{"X-Request-ID", "Content-Type"},
		ExposedHeaders:   []string{"X-Trace-ID"},
		AllowCredentials: true,
		MaxAge:           30 * time.Minute,
	})
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	handler := servlet.ChainFilters(servlet.HandlerFunc(func(context.Context, *servlet.Request, servlet.Response) error {
		t.Fatal("preflight should not reach target handler")
		return nil
	}), filter)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodOptions, "/jobs", nil)
	request.Header.Set("Origin", "https://admin.example.com")
	request.Header.Set("Access-Control-Request-Method", http.MethodPost)
	request.Header.Set("Access-Control-Request-Headers", "x-request-id, content-type")
	servletnethttp.Handler(handler).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNoContent)
	}
	if got := recorder.Header().Get("Access-Control-Allow-Origin"); got != "https://admin.example.com" {
		t.Fatalf("allow origin = %q", got)
	}
	if got := recorder.Header().Get("Access-Control-Allow-Methods"); got != "GET, POST" {
		t.Fatalf("allow methods = %q", got)
	}
	if got := recorder.Header().Get("Access-Control-Allow-Headers"); got != "X-Request-ID, Content-Type" {
		t.Fatalf("allow headers = %q", got)
	}
	if got := recorder.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
		t.Fatalf("allow credentials = %q", got)
	}
	if got := recorder.Header().Get("Access-Control-Max-Age"); got != "1800" {
		t.Fatalf("max age = %q", got)
	}
	if got := recorder.Header().Values("Vary"); len(got) != 3 {
		t.Fatalf("vary = %#v, want 3 values", got)
	}
}

func TestFilterAppliesActualRequestHeaders(t *testing.T) {
	t.Parallel()

	filter, err := cors.New(cors.Config{
		AllowedOriginPatterns: []string{"https://*.example.com"},
		AllowedMethods:        []string{cors.AllMethods},
		AllowedHeaders:        []string{cors.AllHeaders},
		ExposedHeaders:        []string{"X-Trace-ID"},
	})
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	handler := servlet.ChainFilters(servlet.HandlerFunc(func(_ context.Context, _ *servlet.Request, res servlet.Response) error {
		res.Header().Set("X-Trace-ID", "trace-1")
		_, err := res.WriteString("ok")
		return err
	}), filter)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/jobs", nil)
	request.Header.Set("Origin", "https://admin.example.com")
	servletnethttp.Handler(handler).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if got := recorder.Header().Get("Access-Control-Allow-Origin"); got != "https://admin.example.com" {
		t.Fatalf("allow origin = %q", got)
	}
	if got := recorder.Header().Get("Access-Control-Expose-Headers"); got != "X-Trace-ID" {
		t.Fatalf("expose headers = %q", got)
	}
	if body := recorder.Body.String(); body != "ok" {
		t.Fatalf("body = %q, want ok", body)
	}
}

func TestFilterRejectsDisallowedPreflight(t *testing.T) {
	t.Parallel()

	filter, err := cors.New(cors.Config{
		AllowedOrigins: []string{"https://admin.example.com"},
		AllowedMethods: []string{http.MethodGet},
		AllowedHeaders: []string{"X-Request-ID"},
	})
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	handler := servlet.ChainFilters(servlet.HandlerFunc(func(context.Context, *servlet.Request, servlet.Response) error {
		t.Fatal("rejected preflight should not reach target handler")
		return nil
	}), filter)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodOptions, "/jobs", nil)
	request.Header.Set("Origin", "https://other.example.com")
	request.Header.Set("Access-Control-Request-Method", http.MethodPost)
	servletnethttp.Handler(handler).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusForbidden)
	}
	if got := recorder.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("allow origin = %q, want empty", got)
	}
}
