package mvc_test

import (
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
	"goark.dev/goark/web/mvc/view"
)

func TestRouteConditionsRejectMismatches(t *testing.T) {
	t.Parallel()

	registry := web.NewRegistry()
	configurer := mvc.NewConfigurer(mvc.NewController("jobs",
		mvc.POST("/jobs", mvc.NoContent(func(*arkweb.Context) error {
			return nil
		}),
			mvc.WithConsumes("application/json"),
			mvc.WithProduces("application/json"),
			mvc.WithParams("mode=fast"),
			mvc.WithHeaders("X-Tenant=admin"),
		),
	))
	if err := configurer.ConfigureWeb(t.Context(), registry); err != nil {
		t.Fatalf("ConfigureWeb failed: %v", err)
	}
	router, err := registry.Router()
	if err != nil {
		t.Fatalf("Router failed: %v", err)
	}

	tests := []struct {
		name       string
		target     string
		body       string
		content    string
		accept     string
		tenant     string
		wantStatus int
	}{
		{name: "content type", target: "/jobs?mode=fast", body: "{}", content: "text/plain",
			accept: "application/json", tenant: "admin", wantStatus: http.StatusUnsupportedMediaType},
		{name: "accept", target: "/jobs?mode=fast", body: "{}", content: "application/json",
			accept: "text/plain", tenant: "admin", wantStatus: http.StatusNotAcceptable},
		{name: "param", target: "/jobs", body: "{}", content: "application/json",
			accept: "application/json", tenant: "admin", wantStatus: http.StatusBadRequest},
		{name: "header", target: "/jobs?mode=fast", body: "{}", content: "application/json",
			accept: "application/json", tenant: "user", wantStatus: http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, tt.target, strings.NewReader(tt.body))
			request.Header.Set("Content-Type", tt.content)
			request.Header.Set("Accept", tt.accept)
			request.Header.Set("X-Tenant", tt.tenant)
			recorder := httptest.NewRecorder()
			servletnethttp.Handler(router).ServeHTTP(recorder, request)
			if recorder.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d, body=%s", recorder.Code, tt.wantStatus, recorder.Body.String())
			}
		})
	}
}

func TestConfigurationAppliesMappedHandlerInterceptors(t *testing.T) {
	t.Parallel()

	mapping, err := web.NewInterceptorMapping(
		web.WithInterceptorPathPatterns("/api/**"),
		web.WithInterceptorExcludePathPatterns("/api/public/**"),
	)
	if err != nil {
		t.Fatalf("NewInterceptorMapping failed: %v", err)
	}
	beanRegistry := container.NewRegistry()
	configuration := mvc.NewConfiguration("api", mvc.NewRestController("accounts",
		mvc.GET("/api/accounts", mvc.Text(http.StatusOK, func(_ *arkweb.Context) (string, error) {
			return "ok", nil
		})),
		mvc.GET("/api/public/ping", mvc.Text(http.StatusOK, func(_ *arkweb.Context) (string, error) {
			return "ok", nil
		})),
	)).WithMappedHandlerInterceptor(mvc.HandlerInterceptorFuncs{
		PreHandleFunc: func(ctx *arkweb.Context) (bool, error) {
			ctx.Response().Header().Set("X-MVC-Configuration-Interceptor", "hit")
			return true, nil
		},
	}, mapping)
	if err := configuration.Register(t.Context(), beanRegistry); err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	resolver, err := container.New(beanRegistry)
	if err != nil {
		t.Fatalf("container.New failed: %v", err)
	}

	registry := web.NewRegistry()
	if err := web.ApplyConfigurers(t.Context(), resolver, registry); err != nil {
		t.Fatalf("ApplyConfigurers failed: %v", err)
	}
	matched := serveMVCRegistry(t, registry, http.MethodGet, "/api/accounts")
	if got := matched.Header().Get("X-MVC-Configuration-Interceptor"); got != "hit" {
		t.Fatalf("matched header = %q, want hit", got)
	}
	excluded := serveMVCRegistry(t, registry, http.MethodGet, "/api/public/ping")
	if got := excluded.Header().Get("X-MVC-Configuration-Interceptor"); got != "" {
		t.Fatalf("excluded header = %q, want empty", got)
	}
}

func TestRouteConditionsAllowMatchingRequest(t *testing.T) {
	t.Parallel()

	registry := web.NewRegistry()
	configurer := mvc.NewConfigurer(mvc.NewController("jobs",
		mvc.POST("/jobs", mvc.JSON(http.StatusCreated, func(ctx *arkweb.Context) (map[string]string,
			error) {
			produces, _ := ctx.Request().Attribute(mvc.AttributeProducesMediaType)
			return map[string]string{
				"produces": produces.(string),
			}, nil
		}),
			mvc.WithConsumes("application/json"),
			mvc.WithProduces("application/json"),
			mvc.WithParams("mode=fast", "!debug"),
			mvc.WithHeaders("X-Tenant=admin"),
		),
	))
	if err := configurer.ConfigureWeb(t.Context(), registry); err != nil {
		t.Fatalf("ConfigureWeb failed: %v", err)
	}
	router, err := registry.Router()
	if err != nil {
		t.Fatalf("Router failed: %v", err)
	}

	request := httptest.NewRequest(http.MethodPost, "/jobs?mode=fast", strings.NewReader("{}"))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json")
	request.Header.Set("X-Tenant", "admin")
	recorder := httptest.NewRecorder()
	servletnethttp.Handler(router).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201, body=%s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"produces":"application/json"`) {
		t.Fatalf("body = %s, want selected media type", recorder.Body.String())
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

func TestRouteProducesControlsMVCJSONContentType(t *testing.T) {
	t.Parallel()

	const mediaType = "application/vnd.goark.job+json"
	registry := web.NewRegistry()
	configurer := mvc.NewConfigurer(mvc.NewController("jobs",
		mvc.GET("/jobs/1", mvc.JSON(http.StatusOK, func(*arkweb.Context) (map[string]string, error) {
			return map[string]string{"state": "queued"}, nil
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
	if !strings.Contains(recorder.Body.String(), `"state":"queued"`) {
		t.Fatalf("body = %s, want JSON payload", recorder.Body.String())
	}
}

func TestControllerReturnTreatsStringAsViewName(t *testing.T) {
	t.Parallel()

	registry := web.NewRegistry()
	resolver := newTestTemplateResolver(t)
	registry.Use(view.Interceptor(resolver))
	if err := mvc.NewController("pages",
		mvc.GET("/home", mvc.Return(http.StatusOK, func(_ *arkweb.Context) (string, error) {
			return "home", nil
		})),
	).Register(registry); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	recorder := serveMVC(t, registry, http.MethodGet, "/home", "text/html")
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	if recorder.Header().Get("Content-Type") != "text/html; charset=utf-8" {
		t.Fatalf("content type = %q, want html", recorder.Header().Get("Content-Type"))
	}
	if recorder.Body.String() != "<h1>home</h1>" {
		t.Fatalf("body = %q, want rendered view", recorder.Body.String())
	}
}

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
	servletnethttp.Handler(router).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet,
		"/healthz", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
}

func TestRouteConditionsRejectAmbiguousCandidates(t *testing.T) {
	t.Parallel()

	registry := web.NewRegistry()
	configurer := mvc.NewConfigurer(mvc.NewRestController("jobs",
		mvc.GET("/jobs", mvc.NoContent(func(*arkweb.Context) error {
			return nil
		})),
		mvc.GET("/jobs", mvc.NoContent(func(*arkweb.Context) error {
			return nil
		})),
	))

	err := configurer.ConfigureWeb(t.Context(), registry)
	if err == nil || !strings.Contains(err.Error(), "ambiguous route conditions") {
		t.Fatalf("ConfigureWeb err = %v, want ambiguous route conditions", err)
	}
}

func TestResponseBodyIgnoresControllerViewDefault(t *testing.T) {
	t.Parallel()

	registry := web.NewRegistry()
	if err := mvc.NewController("api",
		mvc.GET("/status", mvc.ResponseBody(http.StatusOK, func(_ *arkweb.Context) (string, error) {
			return "UP", nil
		})),
	).Register(registry); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	recorder := serveMVC(t, registry, http.MethodGet, "/status", "text/plain")
	if recorder.Code != http.StatusOK || recorder.Body.String() != "UP" {
		t.Fatalf("response = %d/%q, want 200 UP", recorder.Code, recorder.Body.String())
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

func (*panicResponseAdvice) BeforeWrite(_ *arkweb.Context, _ arkweb.Result) (arkweb.Result, error) {
	panic("nil response advice must not run")
}
