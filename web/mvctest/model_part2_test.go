package mvc_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	arkjson "goark.dev/arkarta/json"
	servletnethttp "goark.dev/arkarta/servlet/nethttp"
	arkweb "goark.dev/arkarta/web"

	"goark.dev/goark/web"
	"goark.dev/goark/web/mvc"
	"goark.dev/goark/web/mvc/view"
)

func TestBindEntityBindsJSONAndWritesResponseEntity(t *testing.T) {
	registry := web.NewRegistry()
	configurer := mvc.NewConfigurer(mvc.NewController("jobs",
		mvc.POST("/jobs", mvc.BindEntity(func(_ *arkweb.Context, input entityCreateRequest) (
			web.ResponseEntity[map[string]string], error) {
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

func TestEntityHandlerWritesResponseEntity(t *testing.T) {
	registry := web.NewRegistry()
	configurer := mvc.NewConfigurer(mvc.NewController("jobs",
		mvc.GET("/jobs/1", mvc.Entity(func(_ *arkweb.Context) (web.ResponseEntity[map[string]string],
			error) {
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

func TestRedirectAttributesInfersAttributeName(t *testing.T) {
	t.Parallel()

	registry := web.NewRegistry()
	if err := mvc.NewController("accounts",
		mvc.GET("/accounts", mvc.Return(0, func(_ *arkweb.Context) (mvc.ModelAndView, error) {
			attributes := mvc.NewRedirectAttributes().
				AddAttribute("id", "42").
				AddAttributeValue(modelAccountSummary{Name: "Goark"})
			return mvc.Redirect("/accounts/{id}", attributes), nil
		})),
	).Register(registry); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	recorder := serveMVC(t, registry, http.MethodGet, "/accounts", "")
	if recorder.Code != http.StatusFound {
		t.Fatalf("status = %d, want 302", recorder.Code)
	}
	if got := recorder.Header().Get("Location"); got !=
		"/accounts/42?modelAccountSummary=%7BGoark%7D" {
		t.Fatalf("Location = %q, want inferred redirect attribute", got)
	}
}

func TestResponseStatusAppliesToImplicitModelView(t *testing.T) {
	t.Parallel()

	registry := web.NewRegistry()
	registry.Use(view.Interceptor(modelViewResolver(t, fstest.MapFS{
		"tasks/current.html": {Data: []byte("<h1>{{.Title}}</h1>")},
	})))
	if err := mvc.NewController("tasks",
		mvc.GET("/tasks/current", mvc.ResponseStatus(http.StatusCreated, mvc.Return(0, func(
			_ *arkweb.Context) (mvc.Model, error) {
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

func TestModelInfersAttributeNames(t *testing.T) {
	t.Parallel()

	item := modelAccountSummary{Name: "Goark"}
	model := mvc.NewModel().
		AddAttributeValue(item).
		AddAttributeValue(&item).
		AddAttributeValue([]modelAccountSummary{{Name: "A"}, {Name: "B"}})

	value, ok := model.Attribute("modelAccountSummary")
	if !ok || value != &item {
		t.Fatalf("modelAccountSummary = %#v/%v, want latest pointer value", value, ok)
	}
	list, ok := model.Attribute("modelAccountSummaryList")
	if !ok || len(list.([]modelAccountSummary)) != 2 {
		t.Fatalf("modelAccountSummaryList = %#v/%v, want inferred slice value", list, ok)
	}
}

type modelAttributeSourcesCriteria struct {
	TenantID       string `form:"tenantId" json:"tenantId"`
	UserID         int64  `form:"userId" json:"userId"`
	RequestID      string `form:"xRequestId" json:"requestId"`
	AcceptLanguage string `json:"acceptLanguage"`
	Mode           string `form:"mode" json:"mode"`
}

type entityCreateRequest struct {
	Name string `json:"name"`
}
