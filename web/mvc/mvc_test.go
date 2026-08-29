package mvc_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	servletnethttp "goark.dev/arkarta/servlet/nethttp"
	arkweb "goark.dev/arkarta/web"
	"goark.dev/goark/web"
	"goark.dev/goark/web/mvc"
)

func TestConfigurerRegistersControllerRoutes(t *testing.T) {
	registry := web.NewRegistry()
	configurer := mvc.NewConfigurer(mvc.NewController("health",
		mvc.GET("/healthz", mvc.JSON(http.StatusOK, func(_ *arkweb.Context) (map[string]string, error) {
			return map[string]string{"status": "UP"}, nil
		})),
	))

	if err := configurer.ConfigureWeb(t.Context(), registry); err != nil {
		t.Fatalf("ConfigureWeb failed: %v", err)
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

func TestControllerSupportsHeadAndOptionsRoutes(t *testing.T) {
	controller := mvc.NewController("system",
		mvc.HEAD("/healthz", mvc.NoContent(func(_ *arkweb.Context) error {
			return nil
		})),
		mvc.OPTIONS("/healthz", mvc.NoContent(func(_ *arkweb.Context) error {
			return nil
		})),
	)

	routes := controller.Routes()
	if len(routes) != 2 {
		t.Fatalf("route count = %d, want 2", len(routes))
	}
	if routes[0].Method != http.MethodHead || routes[1].Method != http.MethodOptions {
		t.Fatalf("methods = %s/%s, want HEAD/OPTIONS", routes[0].Method, routes[1].Method)
	}
}

func TestControllerSupportsTraceRoute(t *testing.T) {
	controller := mvc.NewController("system",
		mvc.TRACE("/diagnostics", mvc.NoContent(func(_ *arkweb.Context) error {
			return nil
		})),
	)

	routes := controller.Routes()
	if len(routes) != 1 {
		t.Fatalf("route count = %d, want 1", len(routes))
	}
	if routes[0].Method != http.MethodTrace {
		t.Fatalf("method = %s, want TRACE", routes[0].Method)
	}
}

func TestRequestMappingCreatesDefaultMethodRoutes(t *testing.T) {
	routes := mvc.RequestMapping("/probe", mvc.NoContent(func(_ *arkweb.Context) error {
		return nil
	}))

	wantMethods := []string{
		http.MethodGet,
		http.MethodHead,
		http.MethodPost,
		http.MethodPut,
		http.MethodPatch,
		http.MethodDelete,
		http.MethodOptions,
	}
	if len(routes) != len(wantMethods) {
		t.Fatalf("route count = %d, want %d", len(routes), len(wantMethods))
	}
	for i, route := range routes {
		if route.Method != wantMethods[i] || route.Pattern != "/probe" {
			t.Fatalf("route[%d] = %s %s, want %s /probe", i, route.Method, route.Pattern, wantMethods[i])
		}
	}

	registry := web.NewRegistry()
	configurer := mvc.NewConfigurer(mvc.NewRestController("probe", routes...))
	if err := configurer.ConfigureWeb(t.Context(), registry); err != nil {
		t.Fatalf("ConfigureWeb failed: %v", err)
	}
	router, err := registry.Router()
	if err != nil {
		t.Fatalf("Router failed: %v", err)
	}
	for _, method := range []string{http.MethodGet, http.MethodPost, http.MethodOptions} {
		recorder := httptest.NewRecorder()
		servletnethttp.Handler(router).ServeHTTP(recorder, httptest.NewRequest(method, "/probe", nil))
		if recorder.Code != http.StatusNoContent {
			t.Fatalf("%s status = %d, want 204", method, recorder.Code)
		}
	}
}

func TestRequestMappingMethodsNormalizesAndDeduplicatesMethods(t *testing.T) {
	routes := mvc.RequestMappingMethods([]string{"post", "POST", " put ", "trace"}, "/jobs", mvc.NoContent(func(_ *arkweb.Context) error {
		return nil
	}))

	wantMethods := []string{http.MethodPost, http.MethodPut, http.MethodTrace}
	if len(routes) != len(wantMethods) {
		t.Fatalf("route count = %d, want %d", len(routes), len(wantMethods))
	}
	for i, route := range routes {
		if route.Method != wantMethods[i] {
			t.Fatalf("route[%d] method = %s, want %s", i, route.Method, wantMethods[i])
		}
	}
}
