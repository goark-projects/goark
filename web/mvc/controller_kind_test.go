package mvc_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"

	servletnethttp "goark.dev/arkarta/servlet/nethttp"
	arkweb "goark.dev/arkarta/web"
	"goark.dev/goark/web"
	"goark.dev/goark/web/message"
	"goark.dev/goark/web/mvc"
	"goark.dev/goark/web/mvc/view"
)

func TestControllerReturnTreatsStringAsViewName(t *testing.T) {
	t.Parallel()

	registry := web.NewRegistry()
	resolver := newTestTemplateResolver(t)
	registry.Use(view.Interceptor(resolver))
	if err := mvc.NewController("pages",
		mvc.GET("/home", mvc.Return(http.StatusOK, func(_ *arkweb.Context) (string, error) {
			return "home", nil
		})),
	).Register(registry); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	recorder := serveMVC(t, registry, http.MethodGet, "/home", "text/html")
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	if recorder.Header().Get("Content-Type") != "text/html; charset=utf-8" {
		t.Fatalf("content type = %q, want html", recorder.Header().Get("Content-Type"))
	}
	if recorder.Body.String() != "<h1>home</h1>" {
		t.Fatalf("body = %q, want rendered view", recorder.Body.String())
	}
}

func TestRestControllerReturnTreatsStringAsResponseBody(t *testing.T) {
	t.Parallel()

	registry := web.NewRegistry()
	if err := mvc.NewRestController("api",
		mvc.GET("/status", mvc.Return(http.StatusOK, func(_ *arkweb.Context) (string, error) {
			return "UP", nil
		})),
	).Register(registry); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	recorder := serveMVC(t, registry, http.MethodGet, "/status", "text/plain")
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	if recorder.Header().Get("Content-Type") != message.MediaTypeTextPlain {
		t.Fatalf("content type = %q, want text/plain", recorder.Header().Get("Content-Type"))
	}
	if recorder.Body.String() != "UP" {
		t.Fatalf("body = %q, want raw response body", recorder.Body.String())
	}
}

func TestResponseBodyIgnoresControllerViewDefault(t *testing.T) {
	t.Parallel()

	registry := web.NewRegistry()
	if err := mvc.NewController("api",
		mvc.GET("/status", mvc.ResponseBody(http.StatusOK, func(_ *arkweb.Context) (string, error) {
			return "UP", nil
		})),
	).Register(registry); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	recorder := serveMVC(t, registry, http.MethodGet, "/status", "text/plain")
	if recorder.Code != http.StatusOK || recorder.Body.String() != "UP" {
		t.Fatalf("response = %d/%q, want 200 UP", recorder.Code, recorder.Body.String())
	}
}

func TestRestControllerReturnWritesStructuredValuesWithMessageConverters(t *testing.T) {
	t.Parallel()

	type payload struct {
		Status string `json:"status"`
	}
	registry := web.NewRegistry()
	if err := mvc.NewRestController("api",
		mvc.GET("/status", mvc.Return(http.StatusCreated, func(_ *arkweb.Context) (payload, error) {
			return payload{Status: "UP"}, nil
		})),
	).Register(registry); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	recorder := serveMVC(t, registry, http.MethodGet, "/status", "application/json")
	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201", recorder.Code)
	}
	if recorder.Header().Get("Content-Type") != "application/json" {
		t.Fatalf("content type = %q, want json", recorder.Header().Get("Content-Type"))
	}
	if recorder.Body.String() != `{"status":"UP"}` {
		t.Fatalf("body = %q, want json payload", recorder.Body.String())
	}
}

func TestControllerKindFromContextReflectsRegisteredController(t *testing.T) {
	t.Parallel()

	registry := web.NewRegistry()
	if err := mvc.NewRestController("api",
		mvc.GET("/kind", mvc.ResponseBody(http.StatusOK, func(ctx *arkweb.Context) (string, error) {
			if mvc.ControllerKindFromContext(ctx) != mvc.ControllerKindREST {
				return "wrong", nil
			}
			return "rest", nil
		})),
	).Register(registry); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	recorder := serveMVC(t, registry, http.MethodGet, "/kind", "text/plain")
	if recorder.Body.String() != "rest" {
		t.Fatalf("body = %q, want rest kind", recorder.Body.String())
	}
}

func TestRestControllerPreservesInvalidRouteValidation(t *testing.T) {
	t.Parallel()

	registry := web.NewRegistry()
	err := mvc.NewRestController("api", mvc.GET("/bad", nil)).Register(registry)
	if !errors.Is(err, web.ErrInvalidRoute) {
		t.Fatalf("err = %v, want ErrInvalidRoute", err)
	}
}

func newTestTemplateResolver(t testing.TB) *view.TemplateResolver {
	t.Helper()
	resolver, err := view.NewTemplateResolver(fstest.MapFS{
		"home.html": &fstest.MapFile{Data: []byte("<h1>home</h1>")},
	})
	if err != nil {
		t.Fatalf("NewTemplateResolver failed: %v", err)
	}
	return resolver
}

func serveMVC(t testing.TB, registry *web.Registry, method string, target string, accept string) *httptest.ResponseRecorder {
	t.Helper()
	router, err := registry.Router()
	if err != nil {
		t.Fatalf("Router failed: %v", err)
	}
	request := httptest.NewRequest(method, target, nil)
	if accept != "" {
		request.Header.Set("Accept", accept)
	}
	recorder := httptest.NewRecorder()
	servletnethttp.Handler(router).ServeHTTP(recorder, request)
	return recorder
}
