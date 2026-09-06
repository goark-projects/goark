package mvc_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	servletnethttp "goark.dev/arkarta/servlet/nethttp"
	arkweb "goark.dev/arkarta/web"

	"goark.dev/goark/web"
	"goark.dev/goark/web/cors"
	"goark.dev/goark/web/mvc"
)

func TestControllerConditionsMergeWithRouteConditions(t *testing.T) {
	t.Parallel()

	const routeMediaType = "application/vnd.goark.job+json"
	registry := web.NewRegistry()
	controller := mvc.NewRestController("jobs",
		mvc.POST("/jobs", mvc.JSON(http.StatusCreated, func(ctx *arkweb.Context) (map[string]string,
			error) {
			produces, _ := ctx.Request().Attribute(mvc.AttributeProducesMediaType)
			return map[string]string{
				"produces": produces.(string),
			}, nil
		}),
			mvc.WithConsumes(routeMediaType),
			mvc.WithProduces(routeMediaType),
			mvc.WithParams("mode=fast"),
			mvc.WithHeaders("X-Route=enabled"),
		),
	).
		WithConsumes("application/json").
		WithProduces("application/json").
		WithParams("tenant=admin").
		WithHeaders("X-Tenant=admin")
	configurer := mvc.NewConfigurer(controller)
	if err := configurer.ConfigureWeb(t.Context(), registry); err != nil {
		t.Fatalf("ConfigureWeb failed: %v", err)
	}
	router, err := registry.Router()
	if err != nil {
		t.Fatalf("Router failed: %v", err)
	}

	request := httptest.NewRequest(http.MethodPost, "/jobs?tenant=admin&mode=fast",
		strings.NewReader("{}"))
	request.Header.Set("Content-Type", routeMediaType)
	request.Header.Set("Accept", routeMediaType)
	request.Header.Set("X-Tenant", "admin")
	request.Header.Set("X-Route", "enabled")
	recorder := httptest.NewRecorder()
	servletnethttp.Handler(router).ServeHTTP(recorder, request)
	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201, body=%s", recorder.Code, recorder.Body.String())
	}
	if got := recorder.Header().Get("Content-Type"); got != routeMediaType {
		t.Fatalf("Content-Type = %q, want route media type", got)
	}
	if !strings.Contains(recorder.Body.String(), `"produces":"`+routeMediaType+`"`) {
		t.Fatalf("body = %s, want route produces media type", recorder.Body.String())
	}

	missingControllerParam := httptest.NewRequest(http.MethodPost, "/jobs?mode=fast",
		strings.NewReader("{}"))
	missingControllerParam.Header.Set("Content-Type", routeMediaType)
	missingControllerParam.Header.Set("Accept", routeMediaType)
	missingControllerParam.Header.Set("X-Tenant", "admin")
	missingControllerParam.Header.Set("X-Route", "enabled")
	recorder = httptest.NewRecorder()
	servletnethttp.Handler(router).ServeHTTP(recorder, missingControllerParam)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("missing controller param status = %d, want 400", recorder.Code)
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
		mvc.OPTIONS("/api/options", mvc.ResponseBody(http.StatusAccepted, func(ctx *arkweb.Context) (
			string, error) {
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
	if len(routes) != 2 || routes[0].Pattern != "/api/users/{id}" || routes[1].Pattern !=
		"/v2/users/{id}" {
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
func TestControllerAdviceResponseAdviceWrapsResponseBody(t *testing.T) {
	t.Parallel()

	registry := web.NewRegistry()
	advice := mvc.NewRestControllerAdvice("api-response").WithResponseAdvice(
		web.ResponseAdviceFunc(func(ctx *arkweb.Context, _ arkweb.Result) (arkweb.Result, error) {
			ctx.Response().Header().Set("X-Response-Advice", "configured")
			return arkweb.JSON(http.StatusAccepted, map[string]string{"status": "advised"}), nil
		}),
	)
	configurer := mvc.NewConfigurer(mvc.NewRestController("users",
		mvc.GET("/users", mvc.JSON(http.StatusOK, func(_ *arkweb.Context) (map[string]string, error) {
			return map[string]string{"status": "origin"}, nil
		})),
	)).WithControllerAdvices(advice)
	if err := configurer.ConfigureWeb(t.Context(), registry); err != nil {
		t.Fatalf("ConfigureWeb failed: %v", err)
	}

	recorder := serveMVCRegistry(t, registry, http.MethodGet, "/users")
	if recorder.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want 202, body=%s", recorder.Code, recorder.Body.String())
	}
	if got := recorder.Header().Get("X-Response-Advice"); got != "configured" {
		t.Fatalf("X-Response-Advice = %q, want configured", got)
	}
	if body := recorder.Body.String(); !strings.Contains(body, `"status":"advised"`) {
		t.Fatalf("body = %s, want advised payload", body)
	}
}

func TestHandlerInterceptorAdapterRunsAfterCompletionOnHandlerError(t *testing.T) {
	t.Parallel()

	boom := errors.New("handler failed")
	var completedErr error
	registry := web.NewRegistry()
	registry.Use(mvc.HandlerInterceptorAdapter(mvc.HandlerInterceptorFuncs{
		AfterCompletionFunc: func(_ *arkweb.Context, err error) {
			completedErr = err
		},
	}))
	if err := registry.GET("/failed", mvc.Text(http.StatusOK, func(_ *arkweb.Context) (string, error) {
		return "", boom
	})); err != nil {
		t.Fatalf("GET failed: %v", err)
	}

	recorder := serveMVCRegistry(t, registry, http.MethodGet, "/failed")
	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", recorder.Code)
	}
	if !errors.Is(completedErr, boom) {
		t.Fatalf("afterCompletion err = %v, want handler error", completedErr)
	}
}

func TestControllerAdviceResponseAdviceSkipsNilValues(t *testing.T) {
	t.Parallel()

	var typedNil *panicResponseAdvice
	registry := web.NewRegistry()
	advice := mvc.NewRestControllerAdvice("api-response").WithResponseAdvice(nil, typedNil)
	configurer := mvc.NewConfigurer(mvc.NewRestController("users",
		mvc.GET("/users", mvc.Text(http.StatusOK, func(_ *arkweb.Context) (string, error) {
			return "origin", nil
		})),
	)).WithControllerAdvices(advice)
	if err := configurer.ConfigureWeb(t.Context(), registry); err != nil {
		t.Fatalf("ConfigureWeb failed: %v", err)
	}

	recorder := serveMVCRegistry(t, registry, http.MethodGet, "/users")
	if recorder.Code != http.StatusOK || recorder.Body.String() != "origin" {
		t.Fatalf("response = %d %q, want 200 origin", recorder.Code, recorder.Body.String())
	}
}

func TestControllerKindFromContextReflectsRegisteredController(t *testing.T) {
	t.Parallel()

	registry := web.NewRegistry()
	if err := mvc.NewRestController("api",
		mvc.GET("/kind", mvc.ResponseBody(http.StatusOK, func(ctx *arkweb.Context) (string, error) {
			if mvc.ControllerKindFromContext(ctx) != mvc.ControllerKindREST {
				return "wrong", nil
			}
			return "rest", nil
		})),
	).Register(registry); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	recorder := serveMVC(t, registry, http.MethodGet, "/kind", "text/plain")
	if recorder.Body.String() != "rest" {
		t.Fatalf("body = %q, want rest kind", recorder.Body.String())
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

func serveCORSRequest(router *arkweb.Router, method string, target string,
	headers map[string]string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, target, nil)
	for name, value := range headers {
		request.Header.Set(name, value)
	}
	recorder := httptest.NewRecorder()
	servletnethttp.Handler(router).ServeHTTP(recorder, request)
	return recorder
}
