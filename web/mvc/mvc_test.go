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

func TestControllerRequestMethodsCombineWithExplicitRouteMethods(t *testing.T) {
	controller := mvc.NewRestController("probe",
		mvc.GET("/probe", mvc.NoContent(func(_ *arkweb.Context) error {
			return nil
		})),
	).WithRequestMethods("post", "POST", " trace ")

	methods := controller.Methods()
	wantMethods := []string{http.MethodPost, http.MethodTrace}
	if len(methods) != len(wantMethods) {
		t.Fatalf("controller method count = %d, want %d", len(methods), len(wantMethods))
	}
	for i, method := range methods {
		if method != wantMethods[i] {
			t.Fatalf("controller method[%d] = %s, want %s", i, method, wantMethods[i])
		}
	}

	registry := web.NewRegistry()
	configurer := mvc.NewConfigurer(controller)
	if err := configurer.ConfigureWeb(t.Context(), registry); err != nil {
		t.Fatalf("ConfigureWeb failed: %v", err)
	}
	routes := registry.Routes()
	wantRegisteredMethods := []string{http.MethodGet, http.MethodPost, http.MethodTrace}
	if len(routes) != len(wantRegisteredMethods) {
		t.Fatalf("registered route count = %d, want %d", len(routes), len(wantRegisteredMethods))
	}
	for i, route := range routes {
		if route.Method != wantRegisteredMethods[i] {
			t.Fatalf("registered route[%d] method = %s, want %s", i, route.Method, wantRegisteredMethods[i])
		}
	}
}

func TestControllerRequestMethodsApplyToImplicitRequestMapping(t *testing.T) {
	controller := mvc.NewRestController("probe",
		mvc.RequestMapping("/probe", mvc.NoContent(func(_ *arkweb.Context) error {
			return nil
		}))...,
	).WithRequestMethods("post", "trace")

	registry := web.NewRegistry()
	configurer := mvc.NewConfigurer(controller)
	if err := configurer.ConfigureWeb(t.Context(), registry); err != nil {
		t.Fatalf("ConfigureWeb failed: %v", err)
	}
	routes := registry.Routes()
	wantMethods := []string{http.MethodPost, http.MethodTrace}
	if len(routes) != len(wantMethods) {
		t.Fatalf("registered route count = %d, want %d", len(routes), len(wantMethods))
	}
	for i, route := range routes {
		if route.Method != wantMethods[i] {
			t.Fatalf("registered route[%d] method = %s, want %s", i, route.Method, wantMethods[i])
		}
	}
}

func TestControllerPathPrefixesApplyToRoutePatterns(t *testing.T) {
	controller := mvc.NewRestController("api",
		mvc.GET("/users/{id}", mvc.Text(http.StatusOK, func(ctx *arkweb.Context) (string, error) {
			return mvc.PathString(ctx, "id")
		})),
	).WithPathPrefixes("/api", "v2")

	prefixes := controller.PathPrefixes()
	if len(prefixes) != 2 || prefixes[0] != "/api" || prefixes[1] != "/v2" {
		t.Fatalf("prefixes = %#v, want /api and /v2", prefixes)
	}

	registry := web.NewRegistry()
	configurer := mvc.NewConfigurer(controller)
	if err := configurer.ConfigureWeb(t.Context(), registry); err != nil {
		t.Fatalf("ConfigureWeb failed: %v", err)
	}
	routes := registry.Routes()
	if len(routes) != 2 || routes[0].Pattern != "/api/users/{id}" || routes[1].Pattern != "/v2/users/{id}" {
		t.Fatalf("registered routes = %#v, want prefixed routes", routes)
	}

	api := serveMVC(t, registry, http.MethodGet, "/api/users/42", "text/plain")
	if api.Code != http.StatusOK || api.Body.String() != "42" {
		t.Fatalf("api response = %d/%q, want 200 42", api.Code, api.Body.String())
	}
	v2 := serveMVC(t, registry, http.MethodGet, "/v2/users/84", "text/plain")
	if v2.Code != http.StatusOK || v2.Body.String() != "84" {
		t.Fatalf("v2 response = %d/%q, want 200 84", v2.Code, v2.Body.String())
	}
	plain := serveMVC(t, registry, http.MethodGet, "/users/42", "text/plain")
	if plain.Code != http.StatusNotFound {
		t.Fatalf("plain status = %d, want 404", plain.Code)
	}
}
