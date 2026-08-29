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

func TestModelAttributeInitializerAppliesToStringViewName(t *testing.T) {
	t.Parallel()

	registry := web.NewRegistry()
	registry.Use(view.Interceptor(modelViewResolver(t, fstest.MapFS{
		"home.html": {Data: []byte("<h1>{{.AppName}}</h1>")},
	})))
	controller := mvc.NewController("pages",
		mvc.GET("/home", mvc.Return(0, func(*arkweb.Context) (string, error) {
			return "home", nil
		})),
	).WithModelAttributes(mvc.ModelAttributeValue("AppName", func(*arkweb.Context) (string, error) {
		return "Goark", nil
	}))
	if err := controller.Register(registry); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	recorder := serveMVC(t, registry, http.MethodGet, "/home", "text/html")
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	if recorder.Body.String() != "<h1>Goark</h1>" {
		t.Fatalf("body = %q, want initialized model", recorder.Body.String())
	}
}

func TestModelAttributeInitializerMergesWithReturnedModel(t *testing.T) {
	t.Parallel()

	registry := web.NewRegistry()
	registry.Use(view.Interceptor(modelViewResolver(t, fstest.MapFS{
		"dashboard.html": {Data: []byte("<h1>{{.AppName}} {{.Title}}</h1>")},
	})))
	controller := mvc.NewController("pages",
		mvc.GET("/dashboard", mvc.Return(0, func(*arkweb.Context) (mvc.Model, error) {
			return mvc.NewModel().AddAttribute("Title", "Dashboard"), nil
		})),
	).WithModelAttributes(mvc.ModelAttributeInitializerFunc(func(_ *arkweb.Context, model mvc.Model) (mvc.Model, error) {
		return model.
			AddAttribute("AppName", "Goark").
			AddAttribute("Title", "Default"), nil
	}))
	if err := controller.Register(registry); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	recorder := serveMVC(t, registry, http.MethodGet, "/dashboard", "text/html")
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	if recorder.Body.String() != "<h1>Goark Dashboard</h1>" {
		t.Fatalf("body = %q, want merged model", recorder.Body.String())
	}
}

func TestControllerAdviceModelAttributeInitializerMergesWithReturnedModel(t *testing.T) {
	t.Parallel()

	registry := web.NewRegistry()
	registry.Use(view.Interceptor(modelViewResolver(t, fstest.MapFS{
		"dashboard.html": {Data: []byte("<h1>{{.AppName}} {{.Title}}</h1>")},
	})))
	advice := mvc.NewControllerAdvice("global-model").WithModelAttributes(
		mvc.ModelAttributeValue("AppName", func(*arkweb.Context) (string, error) {
			return "Goark", nil
		}),
		mvc.ModelAttributeInitializerFunc(func(_ *arkweb.Context, model mvc.Model) (mvc.Model, error) {
			return model.AddAttribute("Title", "Default"), nil
		}),
	)
	configurer := mvc.NewConfigurer(mvc.NewController("pages",
		mvc.GET("/dashboard", mvc.Return(0, func(_ *arkweb.Context) (mvc.ModelAndView, error) {
			return mvc.NewModelAndView("dashboard", mvc.NewModel().AddAttribute("Title", "Dashboard")), nil
		})),
	)).WithControllerAdvices(advice)
	if err := configurer.ConfigureWeb(t.Context(), registry); err != nil {
		t.Fatalf("ConfigureWeb failed: %v", err)
	}

	recorder := serveMVC(t, registry, http.MethodGet, "/dashboard", "text/html")
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	if recorder.Body.String() != "<h1>Goark Dashboard</h1>" {
		t.Fatalf("body = %q, want advice initialized model", recorder.Body.String())
	}
}
