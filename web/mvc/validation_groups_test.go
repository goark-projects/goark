package mvc_test

import (
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

type groupedCreateRequest struct {
	Name string `json:"name" form:"name" arkarta:"required" arkarta-groups:"create"`
	Code string `json:"code" form:"code" arkarta:"required"`
}

type groupedUploadRequest struct {
	Title string                `form:"title" arkarta:"required" arkarta-groups:"create"`
	Token string                `form:"token" arkarta:"required"`
	File  servletmultipart.Part `multipart:"file"`
}

func TestBindJSONGroupsUsesExplicitValidationGroup(t *testing.T) {
	t.Parallel()

	registry := web.NewRegistry()
	configurer := mvc.NewConfigurer(mvc.NewController("users",
		mvc.POST("/users", mvc.BindJSONGroups(http.StatusCreated, func(_ *arkweb.Context, input groupedCreateRequest) (map[string]string, error) {
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

func TestBindBodyGroupsUsesExplicitValidationGroup(t *testing.T) {
	t.Parallel()

	registry := web.NewRegistry()
	configurer := mvc.NewConfigurer(mvc.NewController("users",
		mvc.POST("/users", mvc.BindBodyGroups(http.StatusCreated, func(_ *arkweb.Context, input groupedCreateRequest) (map[string]string, error) {
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

func TestBindEntityGroupsUsesExplicitValidationGroup(t *testing.T) {
	t.Parallel()

	registry := web.NewRegistry()
	configurer := mvc.NewConfigurer(mvc.NewController("jobs",
		mvc.POST("/jobs", mvc.BindEntityGroups(func(_ *arkweb.Context, input groupedCreateRequest) (web.ResponseEntity[map[string]string], error) {
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
		mvc.GET("/users/search", mvc.JSON(http.StatusOK, func(ctx *arkweb.Context) (map[string]string, error) {
			input, err := mvc.ModelAttributeGroups[groupedCreateRequest](ctx, "create")
			if err != nil {
				return nil, err
			}
			return map[string]string{"name": input.Name}, nil
		})),
	))
	router := configureMVC(t, registry, configurer)

	recorder := httptest.NewRecorder()
	servletnethttp.Handler(router).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/users/search?name=ark", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestBindMultipartGroupsUsesExplicitValidationGroup(t *testing.T) {
	t.Parallel()

	registry := web.NewRegistry()
	configurer := mvc.NewConfigurer(mvc.NewController("uploads",
		mvc.POST("/uploads", mvc.BindMultipartGroups(http.StatusCreated, func(_ *arkweb.Context, input groupedUploadRequest) (map[string]string, error) {
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

func TestBindMultipartEntityGroupsUsesExplicitValidationGroup(t *testing.T) {
	t.Parallel()

	registry := web.NewRegistry()
	configurer := mvc.NewConfigurer(mvc.NewController("uploads",
		mvc.POST("/uploads/entity", mvc.BindMultipartEntityGroups(func(_ *arkweb.Context, input groupedUploadRequest) (web.ResponseEntity[map[string]string], error) {
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
