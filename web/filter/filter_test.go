package filter_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"goark.dev/arkarta/servlet"
	servletnethttp "goark.dev/arkarta/servlet/nethttp"
	arkweb "goark.dev/arkarta/web"
	"goark.dev/goark/web/filter"
)

func TestOnceRunsDelegateOnlyOncePerRequest(t *testing.T) {
	t.Parallel()

	calls := 0
	delegate, err := filter.Once("trace", servlet.FilterFunc(func(ctx context.Context, req *servlet.Request, res servlet.Response, chain servlet.Chain) error {
		calls++
		return chain.Next(ctx, req, res)
	}))
	if err != nil {
		t.Fatalf("Once failed: %v", err)
	}
	handler := servlet.ChainFilters(servlet.HandlerFunc(func(_ context.Context, _ *servlet.Request, res servlet.Response) error {
		_, err := res.WriteString("ok")
		return err
	}), delegate, delegate)

	recorder := httptest.NewRecorder()
	servletnethttp.Handler(handler).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/once", nil))

	if calls != 1 {
		t.Fatalf("calls = %d, want 1", calls)
	}
	if recorder.Body.String() != "ok" {
		t.Fatalf("body = %q, want ok", recorder.Body.String())
	}
}

func TestForwardedHeadersUpdatesRequestView(t *testing.T) {
	t.Parallel()

	handler := servlet.ChainFilters(servlet.HandlerFunc(func(_ context.Context, req *servlet.Request, res servlet.Response) error {
		res.Header().Set("X-Scheme", req.Scheme())
		res.Header().Set("X-Host", req.Host())
		res.Header().Set("X-Remote", req.RemoteAddr())
		res.Header().Set("X-URL", req.RequestURL())
		_, err := res.WriteString("ok")
		return err
	}), filter.ForwardedHeaders())

	request := httptest.NewRequest(http.MethodGet, "http://internal/jobs", nil)
	request.RemoteAddr = "10.0.0.2:49200"
	request.Header.Set("X-Forwarded-Proto", "https")
	request.Header.Set("X-Forwarded-Host", "api.example.com")
	request.Header.Set("X-Forwarded-For", "203.0.113.9, 10.0.0.2")
	recorder := httptest.NewRecorder()
	servletnethttp.Handler(handler).ServeHTTP(recorder, request)

	if got := recorder.Header().Get("X-Scheme"); got != "https" {
		t.Fatalf("scheme = %q", got)
	}
	if got := recorder.Header().Get("X-Host"); got != "api.example.com" {
		t.Fatalf("host = %q", got)
	}
	if got := recorder.Header().Get("X-Remote"); got != "203.0.113.9" {
		t.Fatalf("remote = %q", got)
	}
	if got := recorder.Header().Get("X-URL"); got != "https://api.example.com/jobs" {
		t.Fatalf("url = %q", got)
	}
}

func TestHiddenHTTPMethodOverridesPostFormMethod(t *testing.T) {
	t.Parallel()

	router := newHiddenMethodRouter(t)
	handler := servlet.ChainFilters(router, filter.HiddenHTTPMethod())
	body := strings.NewReader(url.Values{filter.DefaultHiddenMethodParameter: {"DELETE"}}.Encode())
	request := httptest.NewRequest(http.MethodPost, "/items/1", body)
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	recorder := httptest.NewRecorder()
	servletnethttp.Handler(handler).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	if recorder.Header().Get("X-Original-Method") != http.MethodPost {
		t.Fatalf("original method = %q, want POST", recorder.Header().Get("X-Original-Method"))
	}
	if recorder.Body.String() != http.MethodDelete {
		t.Fatalf("body = %q, want DELETE route", recorder.Body.String())
	}
}

func TestHiddenHTTPMethodIgnoresUnsupportedOverride(t *testing.T) {
	t.Parallel()

	router := newHiddenMethodRouter(t)
	handler := servlet.ChainFilters(router, filter.HiddenHTTPMethod())
	body := strings.NewReader(url.Values{filter.DefaultHiddenMethodParameter: {"GET"}}.Encode())
	request := httptest.NewRequest(http.MethodPost, "/items/1", body)
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	recorder := httptest.NewRecorder()
	servletnethttp.Handler(handler).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	if recorder.Header().Get("X-Original-Method") != "" {
		t.Fatalf("original method = %q, want empty", recorder.Header().Get("X-Original-Method"))
	}
	if recorder.Body.String() != http.MethodPost {
		t.Fatalf("body = %q, want POST route", recorder.Body.String())
	}
}

func TestHiddenHTTPMethodUsesCustomParameterAndAllowedMethods(t *testing.T) {
	t.Parallel()

	router := newHiddenMethodRouter(t)
	handler := servlet.ChainFilters(router, filter.HiddenHTTPMethod(
		filter.WithHiddenMethodParameter("http_method"),
		filter.WithHiddenMethodAllowedMethods(http.MethodPost),
	))
	body := strings.NewReader(url.Values{"http_method": {"post"}}.Encode())
	request := httptest.NewRequest(http.MethodPost, "/items/1", body)
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	recorder := httptest.NewRecorder()
	servletnethttp.Handler(handler).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	if recorder.Body.String() != http.MethodPost {
		t.Fatalf("body = %q, want POST route", recorder.Body.String())
	}
}

func TestFormContentCachesDeleteFormAndPreservesBody(t *testing.T) {
	t.Parallel()

	handler := servlet.ChainFilters(servlet.HandlerFunc(func(_ context.Context, req *servlet.Request, res servlet.Response) error {
		value, ok := filter.FormContentValue(req, "name")
		if !ok {
			res.SetStatus(http.StatusInternalServerError)
			return nil
		}
		body, err := io.ReadAll(req.Body())
		if err != nil {
			return err
		}
		res.Header().Set("X-Form-Name", value)
		_, err = res.Write(body)
		return err
	}), filter.FormContent())

	form := url.Values{"name": {"goark"}}
	request := httptest.NewRequest(http.MethodDelete, "/items/1", strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=utf-8")
	recorder := httptest.NewRecorder()
	servletnethttp.Handler(handler).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	if recorder.Header().Get("X-Form-Name") != "goark" {
		t.Fatalf("form name = %q, want goark", recorder.Header().Get("X-Form-Name"))
	}
	if recorder.Body.String() != form.Encode() {
		t.Fatalf("body = %q, want original form body", recorder.Body.String())
	}
}

func TestFormContentRejectsOversizedBody(t *testing.T) {
	t.Parallel()

	called := false
	handler := servlet.ChainFilters(servlet.HandlerFunc(func(_ context.Context, _ *servlet.Request, _ servlet.Response) error {
		called = true
		return nil
	}), filter.FormContent(filter.WithFormContentMaxBodyBytes(4)))

	request := httptest.NewRequest(http.MethodDelete, "/items/1", strings.NewReader("name=goark"))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	recorder := httptest.NewRecorder()
	servletnethttp.Handler(handler).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want 413", recorder.Code)
	}
	if called {
		t.Fatal("chain should not run when form body is oversized")
	}
}

func TestShallowETagWritesValidatorAndHonorsIfNoneMatch(t *testing.T) {
	t.Parallel()

	handler := servlet.ChainFilters(servlet.HandlerFunc(func(_ context.Context, _ *servlet.Request, res servlet.Response) error {
		res.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, err := res.WriteString("hello")
		return err
	}), filter.ShallowETag())

	first := httptest.NewRecorder()
	servletnethttp.Handler(handler).ServeHTTP(first, httptest.NewRequest(http.MethodGet, "/etag", nil))
	etag := first.Header().Get("ETag")
	if first.Code != http.StatusOK || first.Body.String() != "hello" || etag == "" {
		t.Fatalf("first response = %d/%q/%q", first.Code, first.Body.String(), etag)
	}

	second := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/etag", nil)
	request.Header.Set("If-None-Match", etag)
	servletnethttp.Handler(handler).ServeHTTP(second, request)
	if second.Code != http.StatusNotModified {
		t.Fatalf("status = %d, want %d", second.Code, http.StatusNotModified)
	}
	if second.Body.Len() != 0 {
		t.Fatalf("body len = %d, want 0", second.Body.Len())
	}
}

func newHiddenMethodRouter(t testing.TB) *arkweb.Router {
	t.Helper()

	router := arkweb.NewRouter()
	for _, method := range []string{http.MethodPost, http.MethodDelete} {
		method := method
		if err := router.Handle(method, "/items/1", arkweb.HandlerFunc(func(ctx *arkweb.Context) (arkweb.Result, error) {
			if original, ok := ctx.Request().Attribute(filter.AttributeOriginalMethod); ok {
				ctx.Response().Header().Set("X-Original-Method", original.(string))
			}
			return arkweb.Text(http.StatusOK, method), nil
		})); err != nil {
			t.Fatalf("Handle %s failed: %v", method, err)
		}
	}
	return router
}
