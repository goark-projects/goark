package mvc_test

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	arkjson "goark.dev/arkarta/json"
	servletmultipart "goark.dev/arkarta/servlet/multipart"
	servletnethttp "goark.dev/arkarta/servlet/nethttp"
	arkweb "goark.dev/arkarta/web"

	"goark.dev/goark/web"
	"goark.dev/goark/web/mvc"
)

func TestRestControllerAdviceReturnAsWritesResponseBody(t *testing.T) {
	t.Parallel()

	registry := web.NewRegistry()
	advice := mvc.NewRestControllerAdvice("api-errors",
		mvc.ExceptionReturnAs[*resourceNotFoundError](http.StatusNotFound, func(_ *arkweb.Context,
			err *resourceNotFoundError) map[string]string {
			return map[string]string{
				"resource": err.resource,
				"id":       err.id,
			}
		}),
	)
	if advice.Kind() != mvc.ControllerKindREST {
		t.Fatalf("advice kind = %d, want REST", advice.Kind())
	}
	if err := advice.ConfigureWeb(t.Context(), registry); err != nil {
		t.Fatalf("ConfigureWeb advice failed: %v", err)
	}
	configurer := mvc.NewConfigurer(mvc.NewRestController("users",
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
	if got := recorder.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("Content-Type = %q, want application/json", got)
	}
	body := recorder.Body.String()
	if !strings.Contains(body, `"resource":"user"`) || !strings.Contains(body, `"id":"42"`) {
		t.Fatalf("body = %s, want REST advice body", body)
	}
}

func TestControllerAdviceEntityAsWritesResponseEntity(t *testing.T) {
	t.Parallel()

	registry := web.NewRegistry()
	advice := mvc.NewRestControllerAdvice("api-errors",
		mvc.ExceptionEntityAs[*resourceNotFoundError](func(_ *arkweb.Context,
			err *resourceNotFoundError) web.ResponseEntity[map[string]string] {
			return web.Status(http.StatusNotFound, map[string]string{
				"resource": err.resource,
				"id":       err.id,
			}).WithHeader("X-Error-ID", err.id)
		}),
	)
	if err := advice.ConfigureWeb(t.Context(), registry); err != nil {
		t.Fatalf("ConfigureWeb advice failed: %v", err)
	}
	configurer := mvc.NewConfigurer(mvc.NewRestController("users",
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
	if got := recorder.Header().Get("X-Error-ID"); got != "42" {
		t.Fatalf("X-Error-ID = %q, want 42", got)
	}
	body := recorder.Body.String()
	if !strings.Contains(body, `"resource":"user"`) || !strings.Contains(body, `"id":"42"`) {
		t.Fatalf("body = %s, want response entity payload", body)
	}
}

func TestBindEntityGroupsUsesExplicitValidationGroup(t *testing.T) {
	t.Parallel()

	registry := web.NewRegistry()
	configurer := mvc.NewConfigurer(mvc.NewController("jobs",
		mvc.POST("/jobs", mvc.BindEntityGroups(func(_ *arkweb.Context, input groupedCreateRequest) (
			web.ResponseEntity[map[string]string], error) {
			return web.Status(http.StatusAccepted, map[string]string{"name": input.Name}).
				WithHeader("X-Validated-Group", "create"), nil
		}, "create")),
	))
	router := configureMVC(t, registry, configurer)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/jobs", strings.NewReader(`{"name":"sync"}`))
	request.Header.Set("Content-Type", arkjson.ContentType)
	servletnethttp.Handler(router).ServeHTTP(recorder, request)
	if recorder.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want 202, body=%s", recorder.Code, recorder.Body.String())
	}
	if got := recorder.Header().Get("X-Validated-Group"); got != "create" {
		t.Fatalf("X-Validated-Group = %q, want create", got)
	}
}

func TestModelAttributeGroupsUsesExplicitValidationGroup(t *testing.T) {
	t.Parallel()

	registry := web.NewRegistry()
	configurer := mvc.NewConfigurer(mvc.NewController("users",
		mvc.GET("/users/search", mvc.JSON(http.StatusOK, func(ctx *arkweb.Context) (
			map[string]string, error) {
			input, err := mvc.ModelAttributeGroups[groupedCreateRequest](ctx, "create")
			if err != nil {
				return nil, err
			}
			return map[string]string{"name": input.Name}, nil
		})),
	))
	router := configureMVC(t, registry, configurer)

	recorder := httptest.NewRecorder()
	servletnethttp.Handler(router).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet,
		"/users/search?name=ark", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestBindJSONGroupsUsesExplicitValidationGroup(t *testing.T) {
	t.Parallel()

	registry := web.NewRegistry()
	configurer := mvc.NewConfigurer(mvc.NewController("users",
		mvc.POST("/users", mvc.BindJSONGroups(http.StatusCreated, func(_ *arkweb.Context,
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

func TestBindMultipartEntityGroupsUsesExplicitValidationGroup(t *testing.T) {
	t.Parallel()

	registry := web.NewRegistry()
	configurer := mvc.NewConfigurer(mvc.NewController("uploads",
		mvc.POST("/uploads/entity", mvc.BindMultipartEntityGroups(func(_ *arkweb.Context,
			input groupedUploadRequest) (web.ResponseEntity[map[string]string], error) {
			return web.Status(http.StatusAccepted, map[string]string{"title": input.Title}), nil
		}, []string{"create"})),
	))
	router := configureMVC(t, registry, configurer)

	request := multipartRequest(t)
	request.URL.Path = "/uploads/entity"
	recorder := httptest.NewRecorder()
	servletnethttp.Handler(router).ServeHTTP(recorder, request)
	if recorder.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want 202, body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestResponseStatusAppliesToNoContentHandler(t *testing.T) {
	t.Parallel()

	registry := web.NewRegistry()
	if err := mvc.NewRestController("api",
		mvc.DELETE("/jobs/1", mvc.ResponseStatus(http.StatusAccepted, mvc.NoContent(func(
			_ *arkweb.Context) error {
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

func serveMVCRegistry(t *testing.T, registry *web.Registry, method,
	target string) *httptest.ResponseRecorder {
	t.Helper()

	router, err := registry.Router()
	if err != nil {
		t.Fatalf("Router failed: %v", err)
	}
	recorder := httptest.NewRecorder()
	servletnethttp.Handler(router).ServeHTTP(recorder, httptest.NewRequest(method, target, nil))
	return recorder
}

func TestConfigurerRejectsNilRegistryWithExceptionHandlers(t *testing.T) {
	t.Parallel()

	configurer := mvc.NewConfigurer().WithExceptionHandlers(mvc.ExceptionHandler(func(
		_ *arkweb.Context, _ error) (arkweb.Result, bool) {
		return arkweb.NoContent(), true
	}))
	if err := configurer.ConfigureWeb(t.Context(), nil); !errors.Is(err, web.ErrNilRegistry) {
		t.Fatalf("err = %v, want ErrNilRegistry", err)
	}
}

func readMultipartBody(part servletmultipart.Part) (string, error) {
	file, err := part.Open()
	if err != nil {
		return "", err
	}
	defer file.Close()
	data, err := io.ReadAll(file)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

type groupedCreateRequest struct {
	Name string `json:"name" form:"name" arkarta:"required" arkarta-groups:"create"`
	Code string `json:"code" form:"code" arkarta:"required"`
}
