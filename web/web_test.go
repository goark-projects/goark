package web_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	servletnethttp "goark.dev/arkarta/servlet/nethttp"
	arkweb "goark.dev/arkarta/web"
	"goark.dev/goark/web"
)

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

func TestRegistrySupportsHeadAndOptionsHelpers(t *testing.T) {
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

	routes := registry.Routes()
	if len(routes) != 2 {
		t.Fatalf("route count = %d, want 2", len(routes))
	}
	if routes[0].Method != http.MethodHead || routes[1].Method != http.MethodOptions {
		t.Fatalf("methods = %s/%s, want HEAD/OPTIONS", routes[0].Method, routes[1].Method)
	}
}
