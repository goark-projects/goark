package mvc_test

import (
	"net/http"
	"testing"
	"testing/fstest"

	arkweb "goark.dev/arkarta/web"
	"goark.dev/goark/web"
	"goark.dev/goark/web/mvc"
	"goark.dev/goark/web/mvc/view"
)

func TestModelAndViewReturnRendersExplicitViewAndModel(t *testing.T) {
	t.Parallel()

	registry := web.NewRegistry()
	registry.Use(view.Interceptor(modelViewResolver(t, fstest.MapFS{
		"jobs/detail.html": {Data: []byte("<h1>{{.Title}}</h1>")},
	})))
	if err := mvc.NewRestController("jobs",
		mvc.GET("/api/jobs/42", mvc.Return(0, func(_ *arkweb.Context) (mvc.ModelAndView, error) {
			model := mvc.NewModel().AddAttribute("Title", "Goark")
			return mvc.NewModelAndView("jobs/detail", model, mvc.WithViewStatus(http.StatusAccepted)), nil
		})),
	).Register(registry); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	recorder := serveMVC(t, registry, http.MethodGet, "/api/jobs/42", "text/html")
	if recorder.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want 202", recorder.Code)
	}
	if recorder.Body.String() != "<h1>Goark</h1>" {
		t.Fatalf("body = %q, want rendered model and view", recorder.Body.String())
	}
}

func TestModelReturnInfersViewNameFromRequestPath(t *testing.T) {
	t.Parallel()

	registry := web.NewRegistry()
	registry.Use(view.Interceptor(modelViewResolver(t, fstest.MapFS{
		"reports/summary.html": {Data: []byte("<h1>{{.Title}}</h1>")},
	})))
	if err := mvc.NewController("reports",
		mvc.GET("/reports/summary.html", mvc.Return(0, func(_ *arkweb.Context) (mvc.Model, error) {
			return mvc.NewModel().AddAttribute("Title", "Summary"), nil
		})),
	).Register(registry); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	recorder := serveMVC(t, registry, http.MethodGet, "/reports/summary.html", "text/html")
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	if recorder.Body.String() != "<h1>Summary</h1>" {
		t.Fatalf("body = %q, want inferred model view", recorder.Body.String())
	}
}

func TestResponseStatusAppliesToImplicitModelView(t *testing.T) {
	t.Parallel()

	registry := web.NewRegistry()
	registry.Use(view.Interceptor(modelViewResolver(t, fstest.MapFS{
		"tasks/current.html": {Data: []byte("<h1>{{.Title}}</h1>")},
	})))
	if err := mvc.NewController("tasks",
		mvc.GET("/tasks/current", mvc.ResponseStatus(http.StatusCreated, mvc.Return(0, func(_ *arkweb.Context) (mvc.Model, error) {
			return mvc.NewModel().AddAttribute("Title", "Current"), nil
		}))),
	).Register(registry); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	recorder := serveMVC(t, registry, http.MethodGet, "/tasks/current", "text/html")
	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201", recorder.Code)
	}
	if recorder.Body.String() != "<h1>Current</h1>" {
		t.Fatalf("body = %q, want inferred model view", recorder.Body.String())
	}
}

func TestDefaultViewNameUsesIndexForRootPath(t *testing.T) {
	t.Parallel()

	registry := web.NewRegistry()
	registry.Use(view.Interceptor(modelViewResolver(t, fstest.MapFS{
		"index.html": {Data: []byte("<h1>{{.Title}}</h1>")},
	})))
	if err := mvc.NewController("home",
		mvc.GET("/", mvc.Return(0, func(_ *arkweb.Context) (mvc.Model, error) {
			return mvc.NewModel().AddAttribute("Title", "Home"), nil
		})),
	).Register(registry); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	recorder := serveMVC(t, registry, http.MethodGet, "/", "text/html")
	if recorder.Body.String() != "<h1>Home</h1>" {
		t.Fatalf("body = %q, want index view", recorder.Body.String())
	}
}

func modelViewResolver(t testing.TB, root fstest.MapFS) *view.TemplateResolver {
	t.Helper()
	resolver, err := view.NewTemplateResolver(root)
	if err != nil {
		t.Fatalf("NewTemplateResolver failed: %v", err)
	}
	return resolver
}
