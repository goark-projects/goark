package mvc_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	servletnethttp "goark.dev/arkarta/servlet/nethttp"
	arkweb "goark.dev/arkarta/web"
	"goark.dev/goark/web"
	"goark.dev/goark/web/cors"
	"goark.dev/goark/web/mvc"
)

func TestRouteCrossOriginHandlesActualAndPreflightRequests(t *testing.T) {
	t.Parallel()

	registry := web.NewRegistry()
	if err := mvc.NewRestController("jobs",
		mvc.GET("/jobs", mvc.ResponseBody(http.StatusOK, func(ctx *arkweb.Context) (string, error) {
			ctx.Response().Header().Set("X-Trace-ID", "trace-1")
			return "ok", nil
		}), mvc.WithCrossOrigin(cors.Config{
			AllowedOrigins:   []string{"https://admin.example.com"},
			AllowedHeaders:   []string{"X-Request-ID"},
			ExposedHeaders:   []string{"X-Trace-ID"},
			AllowCredentials: true,
		})),
	).Register(registry); err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	router := mustMVCRouter(t, registry)

	actual := serveCORSRequest(router, http.MethodGet, "/jobs", map[string]string{
		"Accept": "text/plain",
		"Origin": "https://admin.example.com",
	})
	if actual.Code != http.StatusOK {
		t.Fatalf("actual status = %d, want %d", actual.Code, http.StatusOK)
	}
	if got := actual.Header().Get("Access-Control-Allow-Origin"); got != "https://admin.example.com" {
		t.Fatalf("actual allow origin = %q", got)
	}
	if got := actual.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
		t.Fatalf("actual allow credentials = %q", got)
	}
	if got := actual.Header().Get("Access-Control-Expose-Headers"); got != "X-Trace-ID" {
		t.Fatalf("actual exposed headers = %q", got)
	}

	preflight := serveCORSRequest(router, http.MethodOptions, "/jobs", map[string]string{
		"Origin":                         "https://admin.example.com",
		"Access-Control-Request-Method":  "GET",
		"Access-Control-Request-Headers": "x-request-id",
	})
	if preflight.Code != http.StatusNoContent {
		t.Fatalf("preflight status = %d, want %d", preflight.Code, http.StatusNoContent)
	}
	if got := preflight.Header().Get("Access-Control-Allow-Methods"); got != "GET, HEAD" {
		t.Fatalf("preflight allow methods = %q", got)
	}
	if got := preflight.Header().Get("Access-Control-Allow-Headers"); got != "X-Request-ID" {
		t.Fatalf("preflight allow headers = %q", got)
	}

	plainOptions := serveCORSRequest(router, http.MethodOptions, "/jobs", nil)
	if plainOptions.Code != http.StatusNoContent {
		t.Fatalf("plain OPTIONS status = %d, want %d", plainOptions.Code, http.StatusNoContent)
	}
	if got := plainOptions.Header().Get("Allow"); got != "GET, HEAD, OPTIONS" {
		t.Fatalf("plain OPTIONS allow = %q", got)
	}

	rejected := serveCORSRequest(router, http.MethodOptions, "/jobs", map[string]string{
		"Origin":                        "https://other.example.com",
		"Access-Control-Request-Method": "GET",
	})
	if rejected.Code != http.StatusForbidden {
		t.Fatalf("rejected status = %d, want %d", rejected.Code, http.StatusForbidden)
	}
}

func TestControllerCrossOriginAppliesToRoutes(t *testing.T) {
	t.Parallel()

	registry := web.NewRegistry()
	controller := mvc.NewRestController("api",
		mvc.GET("/api/users", mvc.ResponseBody(http.StatusOK, func(_ *arkweb.Context) (string, error) {
			return "list", nil
		})),
		mvc.POST("/api/users", mvc.ResponseBody(http.StatusCreated, func(_ *arkweb.Context) (string, error) {
			return "created", nil
		})),
	).WithCrossOrigin(cors.Config{
		AllowedOrigins: []string{"https://admin.example.com"},
		AllowedHeaders: []string{cors.AllHeaders},
	})
	if err := controller.Register(registry); err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	router := mustMVCRouter(t, registry)

	preflight := serveCORSRequest(router, http.MethodOptions, "/api/users", map[string]string{
		"Origin":                         "https://admin.example.com",
		"Access-Control-Request-Method":  "POST",
		"Access-Control-Request-Headers": "x-request-id, content-type",
	})
	if preflight.Code != http.StatusNoContent {
		t.Fatalf("preflight status = %d, want %d", preflight.Code, http.StatusNoContent)
	}
	if got := preflight.Header().Get("Access-Control-Allow-Origin"); got != "https://admin.example.com" {
		t.Fatalf("preflight allow origin = %q", got)
	}
	if got := preflight.Header().Get("Access-Control-Allow-Methods"); got != "POST" {
		t.Fatalf("preflight allow methods = %q", got)
	}
	if got := preflight.Header().Get("Access-Control-Allow-Headers"); got != "X-Request-Id, Content-Type" {
		t.Fatalf("preflight allow headers = %q", got)
	}
}

func TestRouteCrossOriginOverridesControllerCrossOrigin(t *testing.T) {
	t.Parallel()

	registry := web.NewRegistry()
	controller := mvc.NewRestController("api",
		mvc.GET("/api/status", mvc.ResponseBody(http.StatusOK, func(_ *arkweb.Context) (string, error) {
			return "ok", nil
		}), mvc.WithCrossOrigin(cors.Config{
			AllowedOrigins: []string{"https://route.example.com"},
		})),
	).WithCrossOrigin(cors.Config{
		AllowedOrigins: []string{"https://controller.example.com"},
	})
	if err := controller.Register(registry); err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	router := mustMVCRouter(t, registry)

	allowed := serveCORSRequest(router, http.MethodGet, "/api/status", map[string]string{
		"Accept": "text/plain",
		"Origin": "https://route.example.com",
	})
	if allowed.Code != http.StatusOK {
		t.Fatalf("allowed status = %d, want %d", allowed.Code, http.StatusOK)
	}
	if got := allowed.Header().Get("Access-Control-Allow-Origin"); got != "https://route.example.com" {
		t.Fatalf("allowed origin = %q", got)
	}

	rejected := serveCORSRequest(router, http.MethodGet, "/api/status", map[string]string{
		"Accept": "text/plain",
		"Origin": "https://controller.example.com",
	})
	if rejected.Code != http.StatusForbidden {
		t.Fatalf("rejected status = %d, want %d", rejected.Code, http.StatusForbidden)
	}
}

func TestCrossOriginPreflightCoexistsWithExplicitOptionsRoute(t *testing.T) {
	t.Parallel()

	registry := web.NewRegistry()
	if err := mvc.NewRestController("api",
		mvc.GET("/api/options", mvc.ResponseBody(http.StatusOK, func(_ *arkweb.Context) (string, error) {
			return "ok", nil
		}), mvc.WithCrossOrigin(cors.Config{
			AllowedOrigins: []string{"https://admin.example.com"},
		})),
		mvc.OPTIONS("/api/options", mvc.ResponseBody(http.StatusAccepted, func(ctx *arkweb.Context) (string, error) {
			ctx.Response().Header().Set("X-Options-Handler", "true")
			return "custom", nil
		})),
	).Register(registry); err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	router := mustMVCRouter(t, registry)

	preflight := serveCORSRequest(router, http.MethodOptions, "/api/options", map[string]string{
		"Origin":                        "https://admin.example.com",
		"Access-Control-Request-Method": "GET",
	})
	if preflight.Code != http.StatusNoContent {
		t.Fatalf("preflight status = %d, want %d", preflight.Code, http.StatusNoContent)
	}
	if got := preflight.Header().Get("X-Options-Handler"); got != "" {
		t.Fatalf("preflight options handler header = %q, want empty", got)
	}

	plainOptions := serveCORSRequest(router, http.MethodOptions, "/api/options", map[string]string{
		"Accept": "text/plain",
	})
	if plainOptions.Code != http.StatusAccepted {
		t.Fatalf("plain OPTIONS status = %d, want %d", plainOptions.Code, http.StatusAccepted)
	}
	if got := plainOptions.Header().Get("X-Options-Handler"); got != "true" {
		t.Fatalf("plain OPTIONS handler header = %q", got)
	}
	if body := plainOptions.Body.String(); body != "custom" {
		t.Fatalf("plain OPTIONS body = %q, want custom", body)
	}
}

func mustMVCRouter(t testing.TB, registry *web.Registry) *arkweb.Router {
	t.Helper()
	router, err := registry.Router()
	if err != nil {
		t.Fatalf("Router failed: %v", err)
	}
	return router
}

func serveCORSRequest(router *arkweb.Router, method string, target string, headers map[string]string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, target, nil)
	for name, value := range headers {
		request.Header.Set(name, value)
	}
	recorder := httptest.NewRecorder()
	servletnethttp.Handler(router).ServeHTTP(recorder, request)
	return recorder
}
