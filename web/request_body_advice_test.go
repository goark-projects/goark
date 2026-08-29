package web_test

import (
	"errors"
	"testing"

	arkweb "goark.dev/arkarta/web"
	"goark.dev/goark/container"
	"goark.dev/goark/web"
)

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
