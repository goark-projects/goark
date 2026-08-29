package web_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	servletnethttp "goark.dev/arkarta/servlet/nethttp"
	arkweb "goark.dev/arkarta/web"
	"goark.dev/goark/web"
)

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
