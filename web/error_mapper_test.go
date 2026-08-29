package web_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	servletnethttp "goark.dev/arkarta/servlet/nethttp"
	arkweb "goark.dev/arkarta/web"
	"goark.dev/goark/container"
	"goark.dev/goark/web"
)

func TestRegistryUsesErrorMapperChain(t *testing.T) {
	t.Parallel()

	handledErr := errors.New("handled")
	registry := web.NewRegistry()
	firstCalled := false
	secondCalled := false
	registry.UseErrorMapper(web.ErrorMapperFunc(func(_ *arkweb.Context, _ error) arkweb.Result {
		firstCalled = true
		return nil
	}))
	registry.UseErrorMapper(web.ErrorMapperFunc(func(_ *arkweb.Context, err error) arkweb.Result {
		secondCalled = true
		if !errors.Is(err, handledErr) {
			return nil
		}
		return arkweb.Text(http.StatusConflict, "handled")
	}))
	if err := registry.GET("/errors", arkweb.HandlerFunc(func(_ *arkweb.Context) (arkweb.Result, error) {
		return nil, handledErr
	})); err != nil {
		t.Fatalf("GET failed: %v", err)
	}

	recorder := serveRegistry(t, registry, http.MethodGet, "/errors")
	if recorder.Code != http.StatusConflict || recorder.Body.String() != "handled" {
		t.Fatalf("response = %d %q, want 409 handled", recorder.Code, recorder.Body.String())
	}
	if !firstCalled || !secondCalled {
		t.Fatalf("mapper calls = first:%v second:%v, want both", firstCalled, secondCalled)
	}
}

func TestRegistryErrorMapperFallsBackToDefault(t *testing.T) {
	t.Parallel()

	registry := web.NewRegistry()
	registry.UseErrorMapper(web.ErrorMapperFunc(func(_ *arkweb.Context, _ error) arkweb.Result {
		return nil
	}))
	if err := registry.GET("/errors", arkweb.HandlerFunc(func(_ *arkweb.Context) (arkweb.Result, error) {
		return nil, errors.New("boom")
	})); err != nil {
		t.Fatalf("GET failed: %v", err)
	}

	recorder := serveRegistry(t, registry, http.MethodGet, "/errors")
	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500, body=%s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"code":"INTERNAL_ERROR"`) {
		t.Fatalf("body = %s, want default error body", recorder.Body.String())
	}
}

func TestDefaultErrorMapperMapsWebStatusError(t *testing.T) {
	t.Parallel()

	cause := errors.New("internal quota bucket")
	registry := web.NewRegistry()
	if err := registry.GET("/limited", arkweb.HandlerFunc(func(_ *arkweb.Context) (arkweb.Result, error) {
		return nil, web.NewStatusError(http.StatusTooManyRequests, "rate limited", cause)
	})); err != nil {
		t.Fatalf("GET failed: %v", err)
	}

	recorder := serveRegistry(t, registry, http.MethodGet, "/limited")
	if recorder.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want 429, body=%s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"message":"rate limited"`) ||
		strings.Contains(recorder.Body.String(), cause.Error()) {
		t.Fatalf("body = %s, want public message without cause", recorder.Body.String())
	}
}

func TestRegistryRouterOptionsOverrideErrorMappers(t *testing.T) {
	t.Parallel()

	registry := web.NewRegistry()
	registry.UseErrorMapper(web.ErrorMapperFunc(func(_ *arkweb.Context, _ error) arkweb.Result {
		return arkweb.Text(http.StatusConflict, "registry")
	}))
	if err := registry.GET("/errors", arkweb.HandlerFunc(func(_ *arkweb.Context) (arkweb.Result, error) {
		return nil, errors.New("boom")
	})); err != nil {
		t.Fatalf("GET failed: %v", err)
	}

	router, err := registry.Router(arkweb.WithErrorMapper(arkweb.ErrorMapperFunc(func(_ *arkweb.Context, _ error) arkweb.Result {
		return arkweb.Text(http.StatusTeapot, "option")
	})))
	if err != nil {
		t.Fatalf("Router failed: %v", err)
	}
	recorder := httptest.NewRecorder()
	servletnethttp.Handler(router).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/errors", nil))

	if recorder.Code != http.StatusTeapot || recorder.Body.String() != "option" {
		t.Fatalf("response = %d %q, want 418 option", recorder.Code, recorder.Body.String())
	}
}

func TestRegisterErrorMapperContributesConfigurer(t *testing.T) {
	t.Parallel()

	handledErr := errors.New("configured")
	beanRegistry := container.NewRegistry()
	if err := web.RegisterErrorMapper(beanRegistry, "configuredErrorMapper", web.ErrorMapperFunc(func(_ *arkweb.Context, err error) arkweb.Result {
		if errors.Is(err, handledErr) {
			return arkweb.Text(http.StatusNotFound, "configured")
		}
		return nil
	})); err != nil {
		t.Fatalf("RegisterErrorMapper failed: %v", err)
	}
	resolver, err := container.New(beanRegistry)
	if err != nil {
		t.Fatalf("container.New failed: %v", err)
	}

	registry := web.NewRegistry()
	if err := web.ApplyConfigurers(t.Context(), resolver, registry); err != nil {
		t.Fatalf("ApplyConfigurers failed: %v", err)
	}
	if err := registry.GET("/errors", arkweb.HandlerFunc(func(_ *arkweb.Context) (arkweb.Result, error) {
		return nil, handledErr
	})); err != nil {
		t.Fatalf("GET failed: %v", err)
	}

	recorder := serveRegistry(t, registry, http.MethodGet, "/errors")
	if recorder.Code != http.StatusNotFound || recorder.Body.String() != "configured" {
		t.Fatalf("response = %d %q, want 404 configured", recorder.Code, recorder.Body.String())
	}
}

func TestRegisterErrorMapperRejectsNilMapper(t *testing.T) {
	t.Parallel()

	if err := web.RegisterErrorMapper(container.NewRegistry(), "nilErrorMapper", nil); !errors.Is(err, web.ErrNilErrorMapper) {
		t.Fatalf("err = %v, want ErrNilErrorMapper", err)
	}
}

func serveRegistry(t *testing.T, registry *web.Registry, method, target string) *httptest.ResponseRecorder {
	t.Helper()

	router, err := registry.Router()
	if err != nil {
		t.Fatalf("Router failed: %v", err)
	}
	recorder := httptest.NewRecorder()
	servletnethttp.Handler(router).ServeHTTP(recorder, httptest.NewRequest(method, target, nil))
	return recorder
}
