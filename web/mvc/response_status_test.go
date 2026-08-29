package mvc_test

import (
	"net/http"
	"testing"
	"testing/fstest"

	arkjson "goark.dev/arkarta/json"
	arkweb "goark.dev/arkarta/web"
	"goark.dev/goark/web"
	"goark.dev/goark/web/mvc"
	"goark.dev/goark/web/mvc/view"
)

func TestResponseStatusAppliesToRestControllerDefaultReturnStatus(t *testing.T) {
	t.Parallel()

	registry := web.NewRegistry()
	if err := mvc.NewRestController("api",
		mvc.GET("/jobs/accepted", mvc.ResponseStatus(http.StatusAccepted, mvc.Return(0, func(_ *arkweb.Context) (string, error) {
			return "accepted", nil
		}))),
	).Register(registry); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	recorder := serveMVC(t, registry, http.MethodGet, "/jobs/accepted", "text/plain")
	if recorder.Code != http.StatusAccepted || recorder.Body.String() != "accepted" {
		t.Fatalf("response = %d/%q, want 202 accepted", recorder.Code, recorder.Body.String())
	}
}

func TestResponseStatusAppliesToControllerViewReturnStatus(t *testing.T) {
	t.Parallel()

	resolver, err := view.NewTemplateResolver(fstest.MapFS{
		"created.html": {Data: []byte("<h1>created</h1>")},
	})
	if err != nil {
		t.Fatalf("NewTemplateResolver failed: %v", err)
	}
	registry := web.NewRegistry()
	registry.Use(view.Interceptor(resolver))
	if err := mvc.NewController("pages",
		mvc.GET("/jobs/created", mvc.ResponseStatus(http.StatusCreated, mvc.Return(0, func(_ *arkweb.Context) (string, error) {
			return "created", nil
		}))),
	).Register(registry); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	recorder := serveMVC(t, registry, http.MethodGet, "/jobs/created", "text/html")
	if recorder.Code != http.StatusCreated || recorder.Body.String() != "<h1>created</h1>" {
		t.Fatalf("response = %d/%q, want 201 rendered view", recorder.Code, recorder.Body.String())
	}
}

func TestResponseStatusAppliesToNoContentHandler(t *testing.T) {
	t.Parallel()

	registry := web.NewRegistry()
	if err := mvc.NewRestController("api",
		mvc.DELETE("/jobs/1", mvc.ResponseStatus(http.StatusAccepted, mvc.NoContent(func(_ *arkweb.Context) error {
			return nil
		}))),
	).Register(registry); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	recorder := serveMVC(t, registry, http.MethodDelete, "/jobs/1", "")
	if recorder.Code != http.StatusAccepted || recorder.Body.Len() != 0 {
		t.Fatalf("response = %d/%q, want 202 without body", recorder.Code, recorder.Body.String())
	}
}

func TestResponseStatusPreservesExplicitResponseEntityStatus(t *testing.T) {
	t.Parallel()

	registry := web.NewRegistry()
	if err := mvc.NewRestController("api",
		mvc.POST("/jobs", mvc.ResponseStatus(http.StatusAccepted, mvc.Entity(func(_ *arkweb.Context) (web.ResponseEntity[map[string]string], error) {
			return web.Status(http.StatusCreated, map[string]string{"state": "created"}), nil
		}))),
	).Register(registry); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	recorder := serveMVC(t, registry, http.MethodPost, "/jobs", arkjson.ContentType)
	if recorder.Code != http.StatusCreated || recorder.Body.String() != `{"state":"created"}` {
		t.Fatalf("response = %d/%q, want explicit 201 entity", recorder.Code, recorder.Body.String())
	}
}
