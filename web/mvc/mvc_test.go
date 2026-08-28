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
