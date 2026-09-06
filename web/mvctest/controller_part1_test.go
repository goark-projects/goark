package mvc_test

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"testing/fstest"

	servletnethttp "goark.dev/arkarta/servlet/nethttp"
	arkweb "goark.dev/arkarta/web"

	"goark.dev/goark/container"
	"goark.dev/goark/web"
	"goark.dev/goark/web/cors"
	"goark.dev/goark/web/message"
	"goark.dev/goark/web/mvc"
	"goark.dev/goark/web/mvc/view"
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

func TestHandlerInterceptorAdapterRunsLifecycle(t *testing.T) {
	t.Parallel()

	calls := make([]string, 0, 3)
	registry := web.NewRegistry()
	registry.Use(mvc.HandlerInterceptorAdapter(mvc.HandlerInterceptorFuncs{
		PreHandleFunc: func(ctx *arkweb.Context) (bool, error) {
			calls = append(calls, "pre")
			ctx.Response().Header().Set("X-Pre-Handle", "hit")
			return true, nil
		},
		PostHandleFunc: func(ctx *arkweb.Context, _ arkweb.Result) (arkweb.Result, error) {
			calls = append(calls, "post")
			ctx.Response().Header().Set("X-Post-Handle", "hit")
			return arkweb.Text(http.StatusAccepted, "post"), nil
		},
		AfterCompletionFunc: func(_ *arkweb.Context, err error) {
			if err != nil {
				t.Fatalf("afterCompletion err = %v, want nil", err)
			}
			calls = append(calls, "after")
		},
	}))
	if err := registry.GET("/intercepted", mvc.Text(http.StatusOK, func(_ *arkweb.Context) (
		string, error) {
		return "origin", nil
	})); err != nil {
		t.Fatalf("GET failed: %v", err)
	}

	recorder := serveMVCRegistry(t, registry, http.MethodGet, "/intercepted")
	if recorder.Code != http.StatusAccepted || recorder.Body.String() != "post" {
		t.Fatalf("response = %d %q, want 202 post", recorder.Code, recorder.Body.String())
	}
	if recorder.Header().Get("X-Pre-Handle") != "hit" || recorder.Header().Get(
		"X-Post-Handle") != "hit" {
		t.Fatalf("headers = %#v, want pre and post markers", recorder.Header())
	}
	if want := []string{"pre", "post", "after"}; !reflect.DeepEqual(calls, want) {
		t.Fatalf("calls = %#v, want %#v", calls, want)
	}
}

func TestHandlerInterceptorAdapterCanShortCircuit(t *testing.T) {
	t.Parallel()

	handlerCalled := false
	registry := web.NewRegistry()
	registry.Use(mvc.HandlerInterceptorAdapter(mvc.HandlerInterceptorFuncs{
		PreHandleFunc: func(ctx *arkweb.Context) (bool, error) {
			ctx.Response().SetStatus(http.StatusNoContent)
			ctx.Response().Header().Set("X-Pre-Handle", "stopped")
			return false, nil
		},
		PostHandleFunc: func(*arkweb.Context, arkweb.Result) (arkweb.Result, error) {
			t.Fatal("postHandle must not run after preHandle short-circuit")
			return nil, nil
		},
		AfterCompletionFunc: func(*arkweb.Context, error) {
			t.Fatal("afterCompletion must not run after preHandle short-circuit")
		},
	}))
	if err := registry.GET("/blocked", mvc.Text(http.StatusOK, func(_ *arkweb.Context) (string,
		error) {
		handlerCalled = true
		return "origin", nil
	})); err != nil {
		t.Fatalf("GET failed: %v", err)
	}

	recorder := serveMVCRegistry(t, registry, http.MethodGet, "/blocked")
	if recorder.Code != http.StatusNoContent || recorder.Body.String() != "" {
		t.Fatalf("response = %d %q, want 204 empty", recorder.Code, recorder.Body.String())
	}
	if got := recorder.Header().Get("X-Pre-Handle"); got != "stopped" {
		t.Fatalf("X-Pre-Handle = %q, want stopped", got)
	}
	if handlerCalled {
		t.Fatal("handler must not run after preHandle short-circuit")
	}
}

func TestRegisterHandlerInterceptorContributesConfigurer(t *testing.T) {
	t.Parallel()

	beanRegistry := container.NewRegistry()
	if err := mvc.RegisterHandlerInterceptor(beanRegistry, "traceHandlerInterceptor",
		mvc.HandlerInterceptorFuncs{
			PreHandleFunc: func(ctx *arkweb.Context) (bool, error) {
				ctx.Response().Header().Set("X-Handler-Interceptor", "hit")
				return true, nil
			},
		}); err != nil {
		t.Fatalf("RegisterHandlerInterceptor failed: %v", err)
	}
	resolver, err := container.New(beanRegistry)
	if err != nil {
		t.Fatalf("container.New failed: %v", err)
	}

	registry := web.NewRegistry()
	if err := web.ApplyConfigurers(t.Context(), resolver, registry); err != nil {
		t.Fatalf("ApplyConfigurers failed: %v", err)
	}
	if err := registry.GET("/registered", mvc.Text(http.StatusOK, func(_ *arkweb.Context) (string,
		error) {
		return "ok", nil
	})); err != nil {
		t.Fatalf("GET failed: %v", err)
	}

	recorder := serveMVCRegistry(t, registry, http.MethodGet, "/registered")
	if got := recorder.Header().Get("X-Handler-Interceptor"); got != "hit" {
		t.Fatalf("X-Handler-Interceptor = %q, want hit", got)
	}
}

func TestRouteProducesControlsMVCEntityContentType(t *testing.T) {
	t.Parallel()

	const mediaType = "application/vnd.goark.job+json"
	registry := web.NewRegistry()
	configurer := mvc.NewConfigurer(mvc.NewController("jobs",
		mvc.GET("/jobs/1", mvc.Entity(func(*arkweb.Context) (web.ResponseEntity[map[string]string],
			error) {
			return web.OK(map[string]string{"state": "queued"}), nil
		}), mvc.WithProduces(mediaType)),
	))
	if err := configurer.ConfigureWeb(t.Context(), registry); err != nil {
		t.Fatalf("ConfigureWeb failed: %v", err)
	}
	router, err := registry.Router()
	if err != nil {
		t.Fatalf("Router failed: %v", err)
	}

	request := httptest.NewRequest(http.MethodGet, "/jobs/1", nil)
	request.Header.Set("Accept", mediaType)
	recorder := httptest.NewRecorder()
	servletnethttp.Handler(router).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", recorder.Code, recorder.Body.String())
	}
	if got := recorder.Header().Get("Content-Type"); got != mediaType {
		t.Fatalf("Content-Type = %q, want %s", got, mediaType)
	}
}

func TestRestControllerReturnTreatsStringAsResponseBody(t *testing.T) {
	t.Parallel()

	registry := web.NewRegistry()
	if err := mvc.NewRestController("api",
		mvc.GET("/status", mvc.Return(http.StatusOK, func(_ *arkweb.Context) (string, error) {
			return "UP", nil
		})),
	).Register(registry); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	recorder := serveMVC(t, registry, http.MethodGet, "/status", "text/plain")
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	if recorder.Header().Get("Content-Type") != message.MediaTypeTextPlain {
		t.Fatalf("content type = %q, want text/plain", recorder.Header().Get("Content-Type"))
	}
	if recorder.Body.String() != "UP" {
		t.Fatalf("body = %q, want raw response body", recorder.Body.String())
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

func serveMVC(t testing.TB, registry *web.Registry, method string, target string,
	accept string) *httptest.ResponseRecorder {
	t.Helper()
	router, err := registry.Router()
	if err != nil {
		t.Fatalf("Router failed: %v", err)
	}
	request := httptest.NewRequest(method, target, nil)
	if accept != "" {
		request.Header.Set("Accept", accept)
	}
	recorder := httptest.NewRecorder()
	servletnethttp.Handler(router).ServeHTTP(recorder, request)
	return recorder
}

func newTestTemplateResolver(t testing.TB) *view.TemplateResolver {
	t.Helper()
	resolver, err := view.NewTemplateResolver(fstest.MapFS{
		"home.html": &fstest.MapFile{Data: []byte("<h1>home</h1>")},
	})
	if err != nil {
		t.Fatalf("NewTemplateResolver failed: %v", err)
	}
	return resolver
}
