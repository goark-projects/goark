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
