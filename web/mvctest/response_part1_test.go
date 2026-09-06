package mvc_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	arkjson "goark.dev/arkarta/json"
	servletmultipart "goark.dev/arkarta/servlet/multipart"
	servletnethttp "goark.dev/arkarta/servlet/nethttp"
	arkweb "goark.dev/arkarta/web"

	"goark.dev/goark/web"
	"goark.dev/goark/web/mvc"
	mvcview "goark.dev/goark/web/mvc/view"
)

func TestControllerAdviceReturnAsRendersLogicalView(t *testing.T) {
	t.Parallel()

	resolver, err := mvcview.NewTemplateResolver(fstest.MapFS{
		"missing.html": {Data: []byte("<h1>missing</h1>")},
	})
	if err != nil {
		t.Fatalf("NewTemplateResolver failed: %v", err)
	}
	registry := web.NewRegistry()
	registry.Use(mvcview.Interceptor(resolver))
	advice := mvc.NewControllerAdvice("page-errors",
		mvc.ExceptionReturnAs[*resourceNotFoundError](http.StatusNotFound, func(_ *arkweb.Context,
			_ *resourceNotFoundError) string {
			return "missing"
		}),
	)
	if advice.Kind() != mvc.ControllerKindView {
		t.Fatalf("advice kind = %d, want view", advice.Kind())
	}
	if err := advice.ConfigureWeb(t.Context(), registry); err != nil {
		t.Fatalf("ConfigureWeb advice failed: %v", err)
	}
	configurer := mvc.NewConfigurer(mvc.NewController("pages",
		mvc.GET("/users/{id}", mvc.JSON(http.StatusOK, func(_ *arkweb.Context) (map[string]string,
			error) {
			return nil, &resourceNotFoundError{resource: "user", id: "42"}
		})),
	))
	if err := configurer.ConfigureWeb(t.Context(), registry); err != nil {
		t.Fatalf("ConfigureWeb controller failed: %v", err)
	}

	recorder := serveMVCRegistry(t, registry, http.MethodGet, "/users/42")
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404, body=%s", recorder.Code, recorder.Body.String())
	}
	if got := recorder.Header().Get("Content-Type"); got != "text/html; charset=utf-8" {
		t.Fatalf("Content-Type = %q, want html", got)
	}
	if recorder.Body.String() != "<h1>missing</h1>" {
		t.Fatalf("body = %q, want rendered view", recorder.Body.String())
	}
}

func TestExceptionHandlerAsMapsTypedErrors(t *testing.T) {
	t.Parallel()

	registry := web.NewRegistry()
	configurer := mvc.NewConfigurer(mvc.NewController("users",
		mvc.GET("/users/{id}", mvc.JSON(http.StatusOK, func(_ *arkweb.Context) (map[string]string,
			error) {
			return nil, &resourceNotFoundError{resource: "user", id: "42"}
		})),
	)).WithExceptionHandlers(mvc.ExceptionHandlerAs(func(_ *arkweb.Context,
		err *resourceNotFoundError) arkweb.Result {
		return arkweb.JSON(http.StatusNotFound, map[string]string{
			"resource": err.resource,
			"id":       err.id,
		})
	}))
	if err := configurer.ConfigureWeb(t.Context(), registry); err != nil {
		t.Fatalf("ConfigureWeb failed: %v", err)
	}

	recorder := serveMVCRegistry(t, registry, http.MethodGet, "/users/42")
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404, body=%s", recorder.Code, recorder.Body.String())
	}
	body := recorder.Body.String()
	if !strings.Contains(body, `"resource":"user"`) || !strings.Contains(body, `"id":"42"`) {
		t.Fatalf("body = %s, want typed error payload", body)
	}
}

func TestResponseStatusAppliesToControllerViewReturnStatus(t *testing.T) {
	t.Parallel()

	resolver, err := mvcview.NewTemplateResolver(fstest.MapFS{
		"created.html": {Data: []byte("<h1>created</h1>")},
	})
	if err != nil {
		t.Fatalf("NewTemplateResolver failed: %v", err)
	}
	registry := web.NewRegistry()
	registry.Use(mvcview.Interceptor(resolver))
	if err := mvc.NewController("pages",
		mvc.GET("/jobs/created", mvc.ResponseStatus(http.StatusCreated, mvc.Return(0, func(
			_ *arkweb.Context) (string, error) {
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

func TestExceptionHandlerIfMapsSentinelErrors(t *testing.T) {
	t.Parallel()

	accessDenied := errors.New("access denied")
	registry := web.NewRegistry()
	configurer := mvc.NewConfigurer(mvc.NewController("users",
		mvc.GET("/admin", mvc.NoContent(func(_ *arkweb.Context) error {
			return errors.Join(errors.New("policy rejected"), accessDenied)
		})),
	)).WithExceptionHandlers(mvc.ExceptionHandlerIf(func(err error) bool {
		return errors.Is(err, accessDenied)
	}, func(_ *arkweb.Context, _ error) arkweb.Result {
		return arkweb.Text(http.StatusForbidden, "denied")
	}))
	if err := configurer.ConfigureWeb(t.Context(), registry); err != nil {
		t.Fatalf("ConfigureWeb failed: %v", err)
	}

	recorder := serveMVCRegistry(t, registry, http.MethodGet, "/admin")
	if recorder.Code != http.StatusForbidden || recorder.Body.String() != "denied" {
		t.Fatalf("response = %d %q, want 403 denied", recorder.Code, recorder.Body.String())
	}
}

func TestBindMultipartGroupsUsesExplicitValidationGroup(t *testing.T) {
	t.Parallel()

	registry := web.NewRegistry()
	configurer := mvc.NewConfigurer(mvc.NewController("uploads",
		mvc.POST("/uploads", mvc.BindMultipartGroups(http.StatusCreated, func(_ *arkweb.Context,
			input groupedUploadRequest) (map[string]string, error) {
			body, err := readMultipartBody(input.File)
			if err != nil {
				return nil, err
			}
			return map[string]string{"title": input.Title, "body": body}, nil
		}, []string{"create"})),
	))
	router := configureMVC(t, registry, configurer)

	recorder := httptest.NewRecorder()
	servletnethttp.Handler(router).ServeHTTP(recorder, multipartRequest(t))
	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201, body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestBindBodyGroupsUsesExplicitValidationGroup(t *testing.T) {
	t.Parallel()

	registry := web.NewRegistry()
	configurer := mvc.NewConfigurer(mvc.NewController("users",
		mvc.POST("/users", mvc.BindBodyGroups(http.StatusCreated, func(_ *arkweb.Context,
			input groupedCreateRequest) (map[string]string, error) {
			return map[string]string{"name": input.Name}, nil
		}, "create")),
	))
	router := configureMVC(t, registry, configurer)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(`{"name":"arkarta"}`))
	request.Header.Set("Content-Type", arkjson.ContentType)
	servletnethttp.Handler(router).ServeHTTP(recorder, request)
	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201, body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestResponseStatusAppliesToRestControllerDefaultReturnStatus(t *testing.T) {
	t.Parallel()

	registry := web.NewRegistry()
	if err := mvc.NewRestController("api",
		mvc.GET("/jobs/accepted", mvc.ResponseStatus(http.StatusAccepted, mvc.Return(0, func(
			_ *arkweb.Context) (string, error) {
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

func TestResponseStatusPreservesExplicitResponseEntityStatus(t *testing.T) {
	t.Parallel()

	registry := web.NewRegistry()
	if err := mvc.NewRestController("api",
		mvc.POST("/jobs", mvc.ResponseStatus(http.StatusAccepted, mvc.Entity(func(
			_ *arkweb.Context) (web.ResponseEntity[map[string]string], error) {
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

func configureMVC(t testing.TB, registry *web.Registry, configurer web.Configurer) *arkweb.Router {
	t.Helper()
	if err := configurer.ConfigureWeb(t.Context(), registry); err != nil {
		t.Fatalf("ConfigureWeb failed: %v", err)
	}
	router, err := registry.Router()
	if err != nil {
		t.Fatalf("Router failed: %v", err)
	}
	return router
}

type groupedUploadRequest struct {
	Title string                `form:"title" arkarta:"required" arkarta-groups:"create"`
	Token string                `form:"token" arkarta:"required"`
	File  servletmultipart.Part `multipart:"file"`
}

type resourceNotFoundError struct {
	resource string
	id       string
}

func (e *resourceNotFoundError) Error() string {
	return e.resource + " " + e.id + " not found"
}
