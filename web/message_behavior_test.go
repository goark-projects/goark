package web_test

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"goark.dev/arkarta/servlet"
	servletnethttp "goark.dev/arkarta/servlet/nethttp"
	arkweb "goark.dev/arkarta/web"
	"goark.dev/goark/container"
	"goark.dev/goark/web"
	"goark.dev/goark/web/message"
	"goark.dev/goark/web/mvc"
)

const tokenMediaType = "application/vnd.goark.token"

type tokenRequest struct {
	Value string
}

type tokenResponse struct {
	Value string
}

type tokenConverter struct{}

func (tokenConverter) MediaTypes() []string {
	return []string{tokenMediaType}
}

func (tokenConverter) CanRead(target any, mediaType string) bool {
	_, ok := target.(*tokenRequest)
	return ok && strings.HasPrefix(mediaType, tokenMediaType)
}

func (tokenConverter) Read(ctx *arkweb.Context, target any, _ string) error {
	input := target.(*tokenRequest)
	data, err := io.ReadAll(ctx.Request().Body())
	if err != nil {
		return err
	}
	input.Value = strings.TrimPrefix(string(data), "token=")
	return nil
}

func (tokenConverter) CanWrite(value any, mediaType string) bool {
	_, ok := value.(tokenResponse)
	return ok && strings.HasPrefix(mediaType, tokenMediaType)
}

func (tokenConverter) Write(ctx *arkweb.Context, value any, mediaType string) error {
	output := value.(tokenResponse)
	if err := servlet.SetContentType(ctx.Response(), mediaType); err != nil {
		return err
	}
	_, err := ctx.Response().WriteString("issued=" + output.Value)
	return err
}

func TestRegisterMessageConverterContributesReadAndWritePipeline(t *testing.T) {
	t.Parallel()

	beanRegistry := container.NewRegistry()
	if err := web.RegisterMessageConverter(beanRegistry, "tokenConverter", tokenConverter{}); err != nil {
		t.Fatalf("RegisterMessageConverter failed: %v", err)
	}
	resolver, err := container.New(beanRegistry)
	if err != nil {
		t.Fatalf("container.New failed: %v", err)
	}

	registry := web.NewRegistry()
	if err := web.ApplyConfigurers(t.Context(), resolver, registry); err != nil {
		t.Fatalf("ApplyConfigurers failed: %v", err)
	}
	configurer := mvc.NewConfigurer(mvc.NewController("tokens",
		mvc.POST("/tokens", mvc.BindBody(http.StatusCreated, func(_ *arkweb.Context, input tokenRequest) (tokenResponse, error) {
			return tokenResponse(input), nil
		}), mvc.WithConsumes(tokenMediaType), mvc.WithProduces(tokenMediaType)),
	))
	if err := configurer.ConfigureWeb(t.Context(), registry); err != nil {
		t.Fatalf("ConfigureWeb failed: %v", err)
	}
	router, err := registry.Router()
	if err != nil {
		t.Fatalf("Router failed: %v", err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/tokens", strings.NewReader("token=abc"))
	request.Header.Set("Content-Type", tokenMediaType)
	request.Header.Set("Accept", tokenMediaType)
	servletnethttp.Handler(router).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201, body=%s", recorder.Code, recorder.Body.String())
	}
	if got := recorder.Header().Get("Content-Type"); got != tokenMediaType {
		t.Fatalf("Content-Type = %q, want %s", got, tokenMediaType)
	}
	if recorder.Body.String() != "issued=abc" {
		t.Fatalf("body = %q, want custom converter output", recorder.Body.String())
	}
}

func TestRegisterMessageConverterRejectsNilConverter(t *testing.T) {
	t.Parallel()

	if err := web.RegisterMessageConverter(container.NewRegistry(), "nilMessageConverter", nil); !errors.Is(err, web.ErrNilMessageConverter) {
		t.Fatalf("err = %v, want ErrNilMessageConverter", err)
	}
}

func TestMessageResultWritesNegotiatedString(t *testing.T) {
	t.Parallel()

	router := arkweb.NewRouter()
	if err := router.Handle(http.MethodGet, "/messages", arkweb.HandlerFunc(func(_ *arkweb.Context) (arkweb.Result, error) {
		return web.Message(http.StatusOK, "hello"), nil
	})); err != nil {
		t.Fatalf("Handle failed: %v", err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/messages", nil)
	request.Header.Set("Accept", "text/plain")
	servletnethttp.Handler(router).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	if recorder.Header().Get("Content-Type") != message.MediaTypeTextPlain {
		t.Fatalf("content type = %q, want text/plain", recorder.Header().Get("Content-Type"))
	}
	if recorder.Body.String() != "hello" {
		t.Fatalf("body = %q, want hello", recorder.Body.String())
	}
}

func TestRegisterRequestBodyAdviceContributesConfigurer(t *testing.T) {
	t.Parallel()

	advice := web.RequestBodyAdviceFunc{
		Before: func(*arkweb.Context, web.RequestBodyAdviceContext) error {
			return nil
		},
	}
	beanRegistry := container.NewRegistry()
	if err := web.RegisterRequestBodyAdvice(beanRegistry, "configuredRequestBodyAdvice", advice); err != nil {
		t.Fatalf("RegisterRequestBodyAdvice failed: %v", err)
	}
	resolver, err := container.New(beanRegistry)
	if err != nil {
		t.Fatalf("container.New failed: %v", err)
	}

	registry := web.NewRegistry()
	if err := web.ApplyConfigurers(t.Context(), resolver, registry); err != nil {
		t.Fatalf("ApplyConfigurers failed: %v", err)
	}
	if got := len(registry.RequestBodyAdvice()); got != 1 {
		t.Fatalf("request body advice count = %d, want 1", got)
	}
}

func TestRegisterRequestBodyAdviceRejectsNilAdvice(t *testing.T) {
	t.Parallel()

	if err := web.RegisterRequestBodyAdvice(container.NewRegistry(), "nilRequestBodyAdvice", nil); !errors.Is(err, web.ErrNilRequestBodyAdvice) {
		t.Fatalf("err = %v, want ErrNilRequestBodyAdvice", err)
	}
}

func TestRequestEntityCapturesMetadataAndBody(t *testing.T) {
	t.Parallel()

	headers := http.Header{
		"X-Trace-Id": {"trace-1"},
	}
	entity := web.NewRequestEntity(web.RequestMetadata{
		Method:        "post",
		URL:           "https://api.example.com/jobs/42?trace=1",
		RequestURI:    "/jobs/42",
		Path:          "/jobs/42",
		Headers:       headers,
		ContentLength: 15,
	}, map[string]string{"name": "sync"}, true)
	headers.Set("X-Trace-Id", "mutated")

	if entity.Method() != http.MethodPost {
		t.Fatalf("method = %q, want POST", entity.Method())
	}
	if entity.URL() != "https://api.example.com/jobs/42?trace=1" {
		t.Fatalf("url = %q, want full request URL", entity.URL())
	}
	if entity.RequestURI() != "/jobs/42" {
		t.Fatalf("request uri = %q, want /jobs/42", entity.RequestURI())
	}
	if entity.Path() != "/jobs/42" {
		t.Fatalf("path = %q, want /jobs/42", entity.Path())
	}
	if entity.ContentLength() != 15 {
		t.Fatalf("content length = %d, want 15", entity.ContentLength())
	}
	if got, ok := entity.HeaderValue("x-trace-id"); !ok || got != "trace-1" {
		t.Fatalf("header = %q ok=%t, want trace-1", got, ok)
	}
	if values := entity.HeaderValues("X-Trace-Id"); len(values) != 1 || values[0] != "trace-1" {
		t.Fatalf("header values = %#v, want trace-1", values)
	}
	if body, ok := entity.Body(); !ok || body["name"] != "sync" {
		t.Fatalf("body = %#v ok=%t, want explicit body", body, ok)
	}
}

func TestRequestEntityHeadersReturnsImmutableSnapshot(t *testing.T) {
	t.Parallel()

	entity := web.NewRequestEntity(web.RequestMetadata{
		Method:  http.MethodGet,
		Headers: http.Header{"X-Role": {"admin"}},
	}, struct{}{}, false)

	headers := entity.Headers()
	headers.Set("X-Role", "operator")

	if got, ok := entity.HeaderValue("X-Role"); !ok || got != "admin" {
		t.Fatalf("header = %q ok=%t, want immutable snapshot", got, ok)
	}
	if _, ok := entity.Body(); ok {
		t.Fatal("empty request entity should not expose a body")
	}
	if entity.HasBody() {
		t.Fatal("empty request entity should report no body")
	}
}

func TestResponseCookieBuildsSetCookieValue(t *testing.T) {
	t.Parallel()

	cookie := web.NewResponseCookie("sid", "abc").
		WithPath("/").
		WithDomain("example.com").
		WithMaxAge(time.Minute).
		WithSecure(true).
		WithHTTPOnly(true).
		WithSameSite(http.SameSiteLaxMode)

	if got := cookie.String(); got != "sid=abc; Path=/; Domain=example.com; Max-Age=60; HttpOnly; Secure; SameSite=Lax" {
		t.Fatalf("cookie = %q", got)
	}
}

func TestResponseCookieRejectsInvalidValues(t *testing.T) {
	t.Parallel()

	tests := []web.ResponseCookie{
		web.NewResponseCookie("", "abc"),
		web.NewResponseCookie("bad name", "abc"),
		web.NewResponseCookie("sid", "a\nb"),
		web.NewResponseCookie("sid", "abc").WithPath("/bad\npath"),
		web.NewResponseCookie("sid", "abc").WithDomain("bad domain"),
	}
	for _, cookie := range tests {
		if got := cookie.String(); got != "" {
			t.Fatalf("invalid cookie string = %q, want empty", got)
		}
	}
}

func TestResponseEntityAddsResponseCookie(t *testing.T) {
	t.Parallel()

	entity := web.NoBody(http.StatusNoContent).
		WithResponseCookie(web.NewResponseCookie("sid", "abc").WithHTTPOnly(true)).
		WithResponseCookie(web.ResponseCookie{})

	if got := entity.Headers().Values("Set-Cookie"); len(got) != 1 || got[0] != "sid=abc; HttpOnly" {
		t.Fatalf("Set-Cookie = %#v", got)
	}
}

func TestResponseCookieReturnsDefensiveCopy(t *testing.T) {
	t.Parallel()

	cookie := web.NewResponseCookie("sid", "abc").WithPath("/")
	raw := cookie.Cookie()
	raw.Path = "/mutated"

	if got := cookie.String(); got != "sid=abc; Path=/" {
		t.Fatalf("cookie mutated through copy: %q", got)
	}
}
