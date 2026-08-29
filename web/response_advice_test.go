package web_test

import (
	"errors"
	"net/http"
	"testing"

	arkweb "goark.dev/arkarta/web"
	"goark.dev/goark/container"
	"goark.dev/goark/web"
)

func TestRegisterResponseAdviceContributesConfigurer(t *testing.T) {
	t.Parallel()

	beanRegistry := container.NewRegistry()
	if err := web.RegisterResponseAdvice(beanRegistry, "configuredAdvice", web.ResponseAdviceFunc(func(ctx *arkweb.Context, _ arkweb.Result) (arkweb.Result, error) {
		ctx.Response().Header().Set("X-Advice", "configured")
		return arkweb.Text(http.StatusAccepted, "advised"), nil
	})); err != nil {
		t.Fatalf("RegisterResponseAdvice failed: %v", err)
	}
	resolver, err := container.New(beanRegistry)
	if err != nil {
		t.Fatalf("container.New failed: %v", err)
	}

	registry := web.NewRegistry()
	if err := web.ApplyConfigurers(t.Context(), resolver, registry); err != nil {
		t.Fatalf("ApplyConfigurers failed: %v", err)
	}
	if err := registry.GET("/advice", arkweb.HandlerFunc(func(_ *arkweb.Context) (arkweb.Result, error) {
		return arkweb.Text(http.StatusOK, "origin"), nil
	})); err != nil {
		t.Fatalf("GET failed: %v", err)
	}

	recorder := serveRegistry(t, registry, http.MethodGet, "/advice")
	if recorder.Code != http.StatusAccepted || recorder.Body.String() != "advised" {
		t.Fatalf("response = %d %q, want 202 advised", recorder.Code, recorder.Body.String())
	}
	if got := recorder.Header().Get("X-Advice"); got != "configured" {
		t.Fatalf("X-Advice = %q, want configured", got)
	}
}

func TestRegisterResponseAdviceRejectsNilAdvice(t *testing.T) {
	t.Parallel()

	if err := web.RegisterResponseAdvice(container.NewRegistry(), "nilAdvice", nil); !errors.Is(err, web.ErrNilResponseAdvice) {
		t.Fatalf("err = %v, want ErrNilResponseAdvice", err)
	}
}
