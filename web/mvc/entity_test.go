package mvc_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	arkjson "goark.dev/arkarta/json"
	servletnethttp "goark.dev/arkarta/servlet/nethttp"
	arkweb "goark.dev/arkarta/web"
	"goark.dev/goark/web"
	"goark.dev/goark/web/mvc"
)

type entityCreateRequest struct {
	Name string `json:"name"`
}

func TestEntityHandlerWritesResponseEntity(t *testing.T) {
	registry := web.NewRegistry()
	configurer := mvc.NewConfigurer(mvc.NewController("jobs",
		mvc.GET("/jobs/1", mvc.Entity(func(_ *arkweb.Context) (web.ResponseEntity[map[string]string], error) {
			return web.Status(http.StatusAccepted, map[string]string{"state": "queued"}).
				WithHeader("X-MVC", "entity"), nil
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
	request := httptest.NewRequest(http.MethodGet, "/jobs/1", nil)
	request.Header.Set("Accept", arkjson.ContentType)
	servletnethttp.Handler(router).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusAccepted)
	}
	if got := recorder.Header().Get("X-MVC"); got != "entity" {
		t.Fatalf("X-MVC = %q, want entity", got)
	}
	var body map[string]string
	if err := arkjson.Unmarshal(nil, recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("response json invalid: %v", err)
	}
	if body["state"] != "queued" {
		t.Fatalf("state = %q, want queued", body["state"])
	}
}

func TestBindEntityBindsJSONAndWritesResponseEntity(t *testing.T) {
	registry := web.NewRegistry()
	configurer := mvc.NewConfigurer(mvc.NewController("jobs",
		mvc.POST("/jobs", mvc.BindEntity(func(_ *arkweb.Context, input entityCreateRequest) (web.ResponseEntity[map[string]string], error) {
			return web.Status(http.StatusCreated, map[string]string{"name": input.Name}).
				WithHeader("Location", "/jobs/1"), nil
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
	request := httptest.NewRequest(http.MethodPost, "/jobs", strings.NewReader(`{"name":"sync"}`))
	request.Header.Set("Accept", arkjson.ContentType)
	request.Header.Set("Content-Type", arkjson.ContentType)
	servletnethttp.Handler(router).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusCreated)
	}
	if got := recorder.Header().Get("Location"); got != "/jobs/1" {
		t.Fatalf("Location = %q, want /jobs/1", got)
	}
	var body map[string]string
	if err := arkjson.Unmarshal(nil, recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("response json invalid: %v", err)
	}
	if body["name"] != "sync" {
		t.Fatalf("name = %q, want sync", body["name"])
	}
}
