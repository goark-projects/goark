package web_test

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"goark.dev/arkarta/servlet"
	servletnethttp "goark.dev/arkarta/servlet/nethttp"
	arkweb "goark.dev/arkarta/web"
	"goark.dev/goark/container"
	"goark.dev/goark/web"
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
			return tokenResponse{Value: input.Value}, nil
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
