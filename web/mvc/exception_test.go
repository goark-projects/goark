package mvc_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	servletnethttp "goark.dev/arkarta/servlet/nethttp"
	arkweb "goark.dev/arkarta/web"
	"goark.dev/goark/web"
	"goark.dev/goark/web/mvc"
)

type resourceNotFoundError struct {
	resource string
	id       string
}

func (e *resourceNotFoundError) Error() string {
	return e.resource + " " + e.id + " not found"
}

func TestExceptionHandlerAsMapsTypedErrors(t *testing.T) {
	t.Parallel()

	registry := web.NewRegistry()
	configurer := mvc.NewConfigurer(mvc.NewController("users",
		mvc.GET("/users/{id}", mvc.JSON(http.StatusOK, func(_ *arkweb.Context) (map[string]string, error) {
			return nil, &resourceNotFoundError{resource: "user", id: "42"}
		})),
	)).WithExceptionHandlers(mvc.ExceptionHandlerAs(func(_ *arkweb.Context, err *resourceNotFoundError) arkweb.Result {
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

func TestConfigurerRejectsNilRegistryWithExceptionHandlers(t *testing.T) {
	t.Parallel()

	configurer := mvc.NewConfigurer().WithExceptionHandlers(mvc.ExceptionHandler(func(_ *arkweb.Context, _ error) (arkweb.Result, bool) {
		return arkweb.NoContent(), true
	}))
	if err := configurer.ConfigureWeb(t.Context(), nil); !errors.Is(err, web.ErrNilRegistry) {
		t.Fatalf("err = %v, want ErrNilRegistry", err)
	}
}

func serveMVCRegistry(t *testing.T, registry *web.Registry, method, target string) *httptest.ResponseRecorder {
	t.Helper()

	router, err := registry.Router()
	if err != nil {
		t.Fatalf("Router failed: %v", err)
	}
	recorder := httptest.NewRecorder()
	servletnethttp.Handler(router).ServeHTTP(recorder, httptest.NewRequest(method, target, nil))
	return recorder
}
