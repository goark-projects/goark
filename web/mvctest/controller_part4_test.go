package mvc_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	servletnethttp "goark.dev/arkarta/servlet/nethttp"
	arkweb "goark.dev/arkarta/web"

	"goark.dev/goark/container"
	"goark.dev/goark/web"
	"goark.dev/goark/web/cors"
	"goark.dev/goark/web/mvc"
)

func TestRegisterMappedHandlerInterceptorContributesScopedConfigurer(t *testing.T) {
	t.Parallel()

	mapping, err := web.NewInterceptorMapping(
		web.WithInterceptorPathPatterns("/api/**"),
		web.WithInterceptorExcludePathPatterns("/api/public/**"),
	)
	if err != nil {
		t.Fatalf("NewInterceptorMapping failed: %v", err)
	}
	beanRegistry := container.NewRegistry()
	if err := mvc.RegisterMappedHandlerInterceptor(beanRegistry, "apiHandlerInterceptor",
		mvc.HandlerInterceptorFuncs{
			PreHandleFunc: func(ctx *arkweb.Context) (bool, error) {
				ctx.Response().Header().Set("X-Mapped-Handler-Interceptor", "hit")
				return true, nil
			},
		}, mapping); err != nil {
		t.Fatalf("RegisterMappedHandlerInterceptor failed: %v", err)
	}
	resolver, err := container.New(beanRegistry)
	if err != nil {
		t.Fatalf("container.New failed: %v", err)
	}

	registry := web.NewRegistry()
	if err := web.ApplyConfigurers(t.Context(), resolver, registry); err != nil {
		t.Fatalf("ApplyConfigurers failed: %v", err)
	}
	for _, target := range []string{"/api/users", "/api/public/ping", "/admin"} {
		if err := registry.GET(target, mvc.Text(http.StatusOK, func(_ *arkweb.Context) (string, error) {
			return "ok", nil
		})); err != nil {
			t.Fatalf("GET %s failed: %v", target, err)
		}
	}

	matched := serveMVCRegistry(t, registry, http.MethodGet, "/api/users")
	if got := matched.Header().Get("X-Mapped-Handler-Interceptor"); got != "hit" {
		t.Fatalf("matched header = %q, want hit", got)
	}
	excluded := serveMVCRegistry(t, registry, http.MethodGet, "/api/public/ping")
	if got := excluded.Header().Get("X-Mapped-Handler-Interceptor"); got != "" {
		t.Fatalf("excluded header = %q, want empty", got)
	}
	unmatched := serveMVCRegistry(t, registry, http.MethodGet, "/admin")
	if got := unmatched.Header().Get("X-Mapped-Handler-Interceptor"); got != "" {
		t.Fatalf("unmatched header = %q, want empty", got)
	}
}

func TestRouteConditionsDispatchMatchingCandidate(t *testing.T) {
	t.Parallel()

	registry := web.NewRegistry()
	configurer := mvc.NewConfigurer(mvc.NewRestController("jobs",
		mvc.GET("/jobs", mvc.JSON(http.StatusOK, func(*arkweb.Context) (map[string]string, error) {
			return map[string]string{"mode": "fast"}, nil
		}), mvc.WithParams("mode=fast")),
		mvc.GET("/jobs", mvc.JSON(http.StatusOK, func(*arkweb.Context) (map[string]string, error) {
			return map[string]string{"mode": "slow"}, nil
		}), mvc.WithParams("mode=slow")),
		mvc.GET("/jobs", mvc.JSON(http.StatusOK, func(*arkweb.Context) (map[string]string, error) {
			return map[string]string{"mode": "default"}, nil
		})),
	))
	if err := configurer.ConfigureWeb(t.Context(), registry); err != nil {
		t.Fatalf("ConfigureWeb failed: %v", err)
	}
	router, err := registry.Router()
	if err != nil {
		t.Fatalf("Router failed: %v", err)
	}

	tests := []struct {
		name string
		path string
		want string
	}{
		{name: "fast", path: "/jobs?mode=fast", want: `"mode":"fast"`},
		{name: "slow", path: "/jobs?mode=slow", want: `"mode":"slow"`},
		{name: "fallback", path: "/jobs", want: `"mode":"default"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			servletnethttp.Handler(router).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet,
				tt.path, nil))
			if recorder.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200, body=%s", recorder.Code, recorder.Body.String())
			}
			if !strings.Contains(recorder.Body.String(), tt.want) {
				t.Fatalf("body = %s, want %s", recorder.Body.String(), tt.want)
			}
		})
	}
}

func TestControllerCrossOriginAppliesToRoutes(t *testing.T) {
	t.Parallel()

	registry := web.NewRegistry()
	controller := mvc.NewRestController("api",
		mvc.GET("/api/users", mvc.ResponseBody(http.StatusOK, func(_ *arkweb.Context) (string, error) {
			return "list", nil
		})),
		mvc.POST("/api/users", mvc.ResponseBody(http.StatusCreated, func(_ *arkweb.Context) (string,
			error) {
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
	if got := preflight.Header().Get("Access-Control-Allow-Origin"); got !=
		"https://admin.example.com" {
		t.Fatalf("preflight allow origin = %q", got)
	}
	if got := preflight.Header().Get("Access-Control-Allow-Methods"); got != "POST" {
		t.Fatalf("preflight allow methods = %q", got)
	}
	if got := preflight.Header().Get("Access-Control-Allow-Headers"); got !=
		"X-Request-Id, Content-Type" {
		t.Fatalf("preflight allow headers = %q", got)
	}
}

func TestRouteConditionsDispatchPreservesSelectedProduces(t *testing.T) {
	t.Parallel()

	registry := web.NewRegistry()
	configurer := mvc.NewConfigurer(mvc.NewRestController("reports",
		mvc.GET("/reports", mvc.JSON(http.StatusOK, func(ctx *arkweb.Context) (map[string]string, error) {
			produces, _ := ctx.Request().Attribute(mvc.AttributeProducesMediaType)
			return map[string]string{"produces": produces.(string)}, nil
		}), mvc.WithProduces("application/json")),
		mvc.GET("/reports", mvc.JSON(http.StatusOK, func(ctx *arkweb.Context) (map[string]string, error) {
			produces, _ := ctx.Request().Attribute(mvc.AttributeProducesMediaType)
			return map[string]string{"produces": produces.(string)}, nil
		}), mvc.WithProduces("text/plain")),
	))
	if err := configurer.ConfigureWeb(t.Context(), registry); err != nil {
		t.Fatalf("ConfigureWeb failed: %v", err)
	}
	router, err := registry.Router()
	if err != nil {
		t.Fatalf("Router failed: %v", err)
	}

	request := httptest.NewRequest(http.MethodGet, "/reports", nil)
	request.Header.Set("Accept", "application/json, text/plain")
	recorder := httptest.NewRecorder()
	servletnethttp.Handler(router).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", recorder.Code, recorder.Body.String())
	}
	if got := recorder.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("Content-Type = %q, want application/json", got)
	}
	if !strings.Contains(recorder.Body.String(), `"produces":"application/json"`) {
		t.Fatalf("body = %s, want selected JSON produces", recorder.Body.String())
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

func TestRestControllerReturnWritesStructuredValuesWithMessageConverters(t *testing.T) {
	t.Parallel()

	type payload struct {
		Status string `json:"status"`
	}
	registry := web.NewRegistry()
	if err := mvc.NewRestController("api",
		mvc.GET("/status", mvc.Return(http.StatusCreated, func(_ *arkweb.Context) (payload, error) {
			return payload{Status: "UP"}, nil
		})),
	).Register(registry); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	recorder := serveMVC(t, registry, http.MethodGet, "/status", "application/json")
	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201", recorder.Code)
	}
	if recorder.Header().Get("Content-Type") != "application/json" {
		t.Fatalf("content type = %q, want json", recorder.Header().Get("Content-Type"))
	}
	if recorder.Body.String() != `{"status":"UP"}` {
		t.Fatalf("body = %q, want json payload", recorder.Body.String())
	}
}

func TestConfigurerAppliesHandlerInterceptors(t *testing.T) {
	t.Parallel()

	registry := web.NewRegistry()
	configurer := mvc.NewConfigurer(mvc.NewRestController("accounts",
		mvc.GET("/accounts", mvc.Text(http.StatusOK, func(_ *arkweb.Context) (string, error) {
			return "ok", nil
		})),
	)).WithHandlerInterceptors(mvc.HandlerInterceptorFuncs{
		PreHandleFunc: func(ctx *arkweb.Context) (bool, error) {
			ctx.Response().Header().Set("X-MVC-Configurer-Interceptor", "hit")
			return true, nil
		},
	})
	if err := configurer.ConfigureWeb(t.Context(), registry); err != nil {
		t.Fatalf("ConfigureWeb failed: %v", err)
	}

	recorder := serveMVCRegistry(t, registry, http.MethodGet, "/accounts")
	if got := recorder.Header().Get("X-MVC-Configurer-Interceptor"); got != "hit" {
		t.Fatalf("X-MVC-Configurer-Interceptor = %q, want hit", got)
	}
}

func TestRouteConditionsRejectAmbiguousCandidatesWithDifferentExpressionOrder(t *testing.T) {
	t.Parallel()

	registry := web.NewRegistry()
	configurer := mvc.NewConfigurer(mvc.NewRestController("jobs",
		mvc.GET("/jobs", mvc.NoContent(func(*arkweb.Context) error {
			return nil
		}), mvc.WithParams("tenant=admin", "mode=fast")),
		mvc.GET("/jobs", mvc.NoContent(func(*arkweb.Context) error {
			return nil
		}), mvc.WithParams("mode=fast", "tenant=admin")),
	))

	err := configurer.ConfigureWeb(t.Context(), registry)
	if err == nil || !strings.Contains(err.Error(), "ambiguous route conditions") {
		t.Fatalf("ConfigureWeb err = %v, want ambiguous route conditions", err)
	}
}

func TestRequestMappingMethodsNormalizesAndDeduplicatesMethods(t *testing.T) {
	routes := mvc.RequestMappingMethods([]string{"post", "POST", " put ", "trace"}, "/jobs",
		mvc.NoContent(func(_ *arkweb.Context) error {
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

func TestRestControllerPreservesInvalidRouteValidation(t *testing.T) {
	t.Parallel()

	registry := web.NewRegistry()
	err := mvc.NewRestController("api", mvc.GET("/bad", nil)).Register(registry)
	if !errors.Is(err, web.ErrInvalidRoute) {
		t.Fatalf("err = %v, want ErrInvalidRoute", err)
	}
}

type panicResponseAdvice struct{}
