package web_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	arkjson "goark.dev/arkarta/json"
	servletnethttp "goark.dev/arkarta/servlet/nethttp"
	arkweb "goark.dev/arkarta/web"
	"goark.dev/goark/container"
	"goark.dev/goark/web"
	"goark.dev/goark/web/problem"
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
	if err := registry.GET("/errors", arkweb.HandlerFunc(func(_ *arkweb.Context) (arkweb.Result,
		error) {
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
	if err := registry.GET("/errors", arkweb.HandlerFunc(func(_ *arkweb.Context) (arkweb.Result,
		error) {
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

func TestRegistryUsesConfiguredFallbackAfterSpecificMappers(t *testing.T) {
	t.Parallel()

	specificErr := errors.New("specific")
	registry := web.NewRegistry()
	registry.UseFallbackErrorMapper(
		web.ErrorMapperFunc(func(_ *arkweb.Context, _ error) arkweb.Result {
			return arkweb.Text(http.StatusInternalServerError, "fallback")
		}),
	)
	registry.UseErrorMapper(web.ErrorMapperFunc(func(_ *arkweb.Context, err error) arkweb.Result {
		if errors.Is(err, specificErr) {
			return arkweb.Text(http.StatusNotFound, "specific")
		}
		return nil
	}))
	if err := registry.GET("/specific", arkweb.HandlerFunc(func(_ *arkweb.Context) (arkweb.Result,
		error) {
		return nil, specificErr
	})); err != nil {
		t.Fatalf("GET specific failed: %v", err)
	}
	if err := registry.GET("/fallback", arkweb.HandlerFunc(func(_ *arkweb.Context) (arkweb.Result,
		error) {
		return nil, errors.New("unknown")
	})); err != nil {
		t.Fatalf("GET fallback failed: %v", err)
	}

	specific := serveRegistry(t, registry, http.MethodGet, "/specific")
	if specific.Code != http.StatusNotFound || specific.Body.String() != "specific" {
		t.Fatalf(
			"specific response = %d %q, want 404 specific",
			specific.Code,
			specific.Body.String(),
		)
	}
	fallback := serveRegistry(t, registry, http.MethodGet, "/fallback")
	if fallback.Code != http.StatusInternalServerError || fallback.Body.String() != "fallback" {
		t.Fatalf(
			"fallback response = %d %q, want 500 fallback",
			fallback.Code,
			fallback.Body.String(),
		)
	}
}

func TestDefaultErrorMapperMapsWebStatusError(t *testing.T) {
	t.Parallel()

	cause := errors.New("internal quota bucket")
	registry := web.NewRegistry()
	if err := registry.GET("/limited", arkweb.HandlerFunc(func(_ *arkweb.Context) (arkweb.Result,
		error) {
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
	if err := registry.GET("/errors", arkweb.HandlerFunc(func(_ *arkweb.Context) (arkweb.Result,
		error) {
		return nil, errors.New("boom")
	})); err != nil {
		t.Fatalf("GET failed: %v", err)
	}

	router, err := registry.Router(
		arkweb.WithErrorMapper(
			arkweb.ErrorMapperFunc(func(_ *arkweb.Context, _ error) arkweb.Result {
				return arkweb.Text(http.StatusTeapot, "option")
			}),
		),
	)
	if err != nil {
		t.Fatalf("Router failed: %v", err)
	}
	recorder := httptest.NewRecorder()
	servletnethttp.Handler(router).
		ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/errors", nil))

	if recorder.Code != http.StatusTeapot || recorder.Body.String() != "option" {
		t.Fatalf("response = %d %q, want 418 option", recorder.Code, recorder.Body.String())
	}
}

func TestRegisterErrorMapperContributesConfigurer(t *testing.T) {
	t.Parallel()

	handledErr := errors.New("configured")
	beanRegistry := container.NewRegistry()
	if err := web.RegisterErrorMapper(beanRegistry, "configuredErrorMapper", web.ErrorMapperFunc(
		func(_ *arkweb.Context, err error) arkweb.Result {
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
	if err := registry.GET("/errors", arkweb.HandlerFunc(func(_ *arkweb.Context) (arkweb.Result,
		error) {
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

	if err := web.RegisterErrorMapper(container.NewRegistry(), "nilErrorMapper", nil); !errors.Is(
		err,
		web.ErrNilErrorMapper,
	) {
		t.Fatalf("err = %v, want ErrNilErrorMapper", err)
	}
}

func TestRegisterFallbackErrorMapperRejectsNilMapper(t *testing.T) {
	t.Parallel()

	if err := web.RegisterFallbackErrorMapper(container.NewRegistry(), "nilFallbackErrorMapper",
		nil); !errors.Is(
		err,
		web.ErrNilErrorMapper,
	) {
		t.Fatalf("err = %v, want ErrNilErrorMapper", err)
	}
}

func serveRegistry(
	t *testing.T,
	registry *web.Registry,
	method, target string,
) *httptest.ResponseRecorder {
	t.Helper()

	router, err := registry.Router()
	if err != nil {
		t.Fatalf("Router failed: %v", err)
	}
	recorder := httptest.NewRecorder()
	servletnethttp.Handler(router).ServeHTTP(recorder, httptest.NewRequest(method, target, nil))
	return recorder
}

func TestNewResponseStatusExceptionCreatesStatusError(t *testing.T) {
	t.Parallel()

	cause := errors.New("database duplicate key")
	err := web.NewResponseStatusException(http.StatusConflict, "job already exists", cause)

	var statusErr web.StatusError
	if !errors.As(err, &statusErr) {
		t.Fatalf("err = %T, want StatusError", err)
	}
	if statusErr.StatusCode() != http.StatusConflict {
		t.Fatalf("status = %d, want 409", statusErr.StatusCode())
	}
	if statusErr.PublicMessage() != "job already exists" {
		t.Fatalf("public message = %q, want job already exists", statusErr.PublicMessage())
	}
	if !errors.Is(err, cause) {
		t.Fatalf("err does not unwrap cause")
	}

	var exception *web.ResponseStatusException
	if !errors.As(err, &exception) {
		t.Fatalf("err = %T, want ResponseStatusException", err)
	}
}

func TestResponseStatusExceptionMapsToProblemDetail(t *testing.T) {
	t.Parallel()

	cause := errors.New("internal storage detail")
	registry := web.NewRegistry()
	registry.UseErrorMapper(problem.NewMapper())
	if err := registry.POST("/jobs", arkweb.HandlerFunc(func(_ *arkweb.Context) (arkweb.Result,
		error) {
		return nil, web.NewResponseStatusException(http.StatusConflict, "job already exists", cause)
	})); err != nil {
		t.Fatalf("POST failed: %v", err)
	}

	recorder := serveRegistry(t, registry, http.MethodPost, "/jobs")
	if recorder.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409, body=%s", recorder.Code, recorder.Body.String())
	}
	body := recorder.Body.String()
	if !strings.Contains(body, `"detail":"job already exists"`) ||
		!strings.Contains(body, `"error":"HTTP_409"`) {
		t.Fatalf("body = %s, want conflict problem detail", body)
	}
	if strings.Contains(body, cause.Error()) {
		t.Fatalf("body exposes cause: %s", body)
	}
}

func TestRequestLocaleReadsAcceptLanguageAndWritesContentLanguage(t *testing.T) {
	t.Parallel()

	registry := web.NewRegistry()
	if err := registry.GET("/locale", arkweb.HandlerFunc(func(ctx *arkweb.Context) (arkweb.Result,
		error) {
		locale, ok := web.RequestLocale(ctx)
		locales := web.RequestLocales(ctx)
		payload := map[string]any{
			"ok":         ok,
			"locale":     locale.Tag(),
			"language":   locale.Language(),
			"region":     locale.Region(),
			"localeSize": len(locales),
		}
		return web.OK(payload).WithContentLanguage(locale), nil
	})); err != nil {
		t.Fatalf("GET failed: %v", err)
	}
	router, err := registry.Router()
	if err != nil {
		t.Fatalf("Router failed: %v", err)
	}

	request := httptest.NewRequest(http.MethodGet, "/locale", nil)
	request.Header.Set("Accept", arkjson.ContentType)
	request.Header.Set("Accept-Language", "en-US;q=0.8, zh-CN;q=0.9")
	recorder := httptest.NewRecorder()
	servletnethttp.Handler(router).ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", recorder.Code, recorder.Body.String())
	}
	if got := recorder.Header().Get("Content-Language"); got != "zh-CN" {
		t.Fatalf("Content-Language = %q, want zh-CN", got)
	}
	var payload map[string]any
	if err := arkjson.Unmarshal(nil, recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("response json invalid: %v", err)
	}
	if payload["ok"] != true || payload["locale"] != "zh-CN" || payload["language"] != "zh" ||
		payload["region"] != "CN" ||
		payload["localeSize"] != float64(2) {
		t.Fatalf("payload = %#v, want request locale details", payload)
	}
}
