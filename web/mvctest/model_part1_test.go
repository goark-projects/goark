package mvc_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"

	arkjson "goark.dev/arkarta/json"
	servletnethttp "goark.dev/arkarta/servlet/nethttp"
	arkweb "goark.dev/arkarta/web"

	"goark.dev/goark/web"
	"goark.dev/goark/web/mvc"
	"goark.dev/goark/web/mvc/view"
)

func TestModelAttributeFallbackSourcesDoNotSuppressUnrelatedHeaders(t *testing.T) {
	t.Parallel()

	registry := web.NewRegistry()
	controller := mvc.NewRestController("suppressedSources",
		mvc.GET("/users/{userId}", mvc.JSON(http.StatusOK, func(ctx *arkweb.Context) ([]string, error) {
			_, result, err := mvc.ModelAttributeResult[binderSuppressedInput](ctx)
			if err != nil {
				return nil, err
			}
			return result.SuppressedFields(), nil
		})),
	).WithInitBinders(mvc.BinderInitializerFunc(func(_ *arkweb.Context, binder *mvc.DataBinder) error {
		return binder.SetAllowedFields("name")
	}))
	if err := controller.Register(registry); err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	router, err := registry.Router()
	if err != nil {
		t.Fatalf("Router failed: %v", err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/users/42?name=ada&admin=true", nil)
	request.Header.Set("User-Agent", "goark-test")
	request.Header.Set("X-Trace-Id", "trace-1")
	servletnethttp.Handler(router).ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", recorder.Code, recorder.Body.String())
	}
	var got []string
	if err := arkjson.Unmarshal(nil, recorder.Body.Bytes(), &got); err != nil {
		t.Fatalf("response json invalid: %v", err)
	}
	if len(got) != 1 || got[0] != "admin" {
		t.Fatalf("suppressed fields = %#v, want only bound request fields", got)
	}
}

func TestModelAttributeBindsPathVariablesAndHeadersWithRequestParameterPriority(t *testing.T) {
	t.Parallel()

	router := arkweb.NewRouter()
	err := router.Handle(http.MethodGet, "/tenants/{tenantId}/users/{userId}", mvc.JSON(
		http.StatusOK, func(ctx *arkweb.Context) (modelAttributeSourcesCriteria, error) {
			return mvc.ModelAttribute[modelAttributeSourcesCriteria](ctx)
		}))
	if err != nil {
		t.Fatalf("Handle failed: %v", err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet,
		"/tenants/core;scope=internal/users/42;role=admin?tenantId=query&mode=query", nil)
	request.Header.Set("X-Request-Id", "req-1")
	request.Header.Set("Accept-Language", "zh-CN")
	request.Header.Set("Mode", "header")
	servletnethttp.Handler(router).ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", recorder.Code, recorder.Body.String())
	}

	var got modelAttributeSourcesCriteria
	if err := arkjson.Unmarshal(nil, recorder.Body.Bytes(), &got); err != nil {
		t.Fatalf("response json invalid: %v", err)
	}
	if got.TenantID != "query" ||
		got.UserID != 42 ||
		got.RequestID != "req-1" ||
		got.AcceptLanguage != "zh-CN" ||
		got.Mode != "query" {
		t.Fatalf("criteria = %#v, want model attribute sources with request parameter priority", got)
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
	).WithModelAttributes(mvc.ModelAttributeInitializerFunc(func(_ *arkweb.Context,
		model mvc.Model) (mvc.Model, error) {
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

func TestModelAndViewInfersModelObjectName(t *testing.T) {
	t.Parallel()

	registry := web.NewRegistry()
	registry.Use(view.Interceptor(modelViewResolver(t, fstest.MapFS{
		"accounts/detail.html": {Data: []byte("<h1>{{.modelAccountSummary.Name}}</h1>")},
	})))
	if err := mvc.NewController("accounts",
		mvc.GET("/accounts/detail", mvc.Return(0, func(_ *arkweb.Context) (mvc.ModelAndView, error) {
			return mvc.NewModelAndView("accounts/detail", modelAccountSummary{Name: "Goark"}), nil
		})),
	).Register(registry); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	recorder := serveMVC(t, registry, http.MethodGet, "/accounts/detail", "text/html")
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	if recorder.Body.String() != "<h1>Goark</h1>" {
		t.Fatalf("body = %q, want inferred model object", recorder.Body.String())
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

type modelAccountSummary struct {
	Name string
}
