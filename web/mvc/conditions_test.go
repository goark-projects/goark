package mvc_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	servletnethttp "goark.dev/arkarta/servlet/nethttp"
	arkweb "goark.dev/arkarta/web"
	"goark.dev/goark/web"
	"goark.dev/goark/web/mvc"
)

func TestRouteConditionsAllowMatchingRequest(t *testing.T) {
	t.Parallel()

	registry := web.NewRegistry()
	configurer := mvc.NewConfigurer(mvc.NewController("jobs",
		mvc.POST("/jobs", mvc.JSON(http.StatusCreated, func(ctx *arkweb.Context) (map[string]string, error) {
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

func TestRouteProducesControlsMVCEntityContentType(t *testing.T) {
	t.Parallel()

	const mediaType = "application/vnd.goark.job+json"
	registry := web.NewRegistry()
	configurer := mvc.NewConfigurer(mvc.NewController("jobs",
		mvc.GET("/jobs/1", mvc.Entity(func(*arkweb.Context) (web.ResponseEntity[map[string]string], error) {
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
			servletnethttp.Handler(router).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, tt.path, nil))
			if recorder.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200, body=%s", recorder.Code, recorder.Body.String())
			}
			if !strings.Contains(recorder.Body.String(), tt.want) {
				t.Fatalf("body = %s, want %s", recorder.Body.String(), tt.want)
			}
		})
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
		{name: "content type", target: "/jobs?mode=fast", body: "{}", content: "text/plain", accept: "application/json", tenant: "admin", wantStatus: http.StatusUnsupportedMediaType},
		{name: "accept", target: "/jobs?mode=fast", body: "{}", content: "application/json", accept: "text/plain", tenant: "admin", wantStatus: http.StatusNotAcceptable},
		{name: "param", target: "/jobs", body: "{}", content: "application/json", accept: "application/json", tenant: "admin", wantStatus: http.StatusBadRequest},
		{name: "header", target: "/jobs?mode=fast", body: "{}", content: "application/json", accept: "application/json", tenant: "user", wantStatus: http.StatusBadRequest},
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

func TestControllerConditionsMergeWithRouteConditions(t *testing.T) {
	t.Parallel()

	const routeMediaType = "application/vnd.goark.job+json"
	registry := web.NewRegistry()
	controller := mvc.NewRestController("jobs",
		mvc.POST("/jobs", mvc.JSON(http.StatusCreated, func(ctx *arkweb.Context) (map[string]string, error) {
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

	request := httptest.NewRequest(http.MethodPost, "/jobs?tenant=admin&mode=fast", strings.NewReader("{}"))
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

	missingControllerParam := httptest.NewRequest(http.MethodPost, "/jobs?mode=fast", strings.NewReader("{}"))
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
