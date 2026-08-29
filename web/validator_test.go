package web_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	arkjson "goark.dev/arkarta/json"
	servletnethttp "goark.dev/arkarta/servlet/nethttp"
	"goark.dev/arkarta/validation"
	arkweb "goark.dev/arkarta/web"
	"goark.dev/goark/container"
	"goark.dev/goark/web"
	"goark.dev/goark/web/mvc"
)

type rejectingValidator struct{}

func (rejectingValidator) Validate(_ context.Context, _ any) (validation.Result, error) {
	return validation.NewResult(validation.NewViolation("name", "reserved", "名称不可用", nil)), nil
}

func TestRegisterValidatorContributesConfigurer(t *testing.T) {
	t.Parallel()

	beanRegistry := container.NewRegistry()
	if err := web.RegisterValidator(beanRegistry, "rejectingValidator", rejectingValidator{}); err != nil {
		t.Fatalf("RegisterValidator failed: %v", err)
	}
	resolver, err := container.New(beanRegistry)
	if err != nil {
		t.Fatalf("container.New failed: %v", err)
	}
	registry := web.NewRegistry()
	if err := web.ApplyConfigurers(t.Context(), resolver, registry); err != nil {
		t.Fatalf("ApplyConfigurers failed: %v", err)
	}
	if registry.Validator() == nil {
		t.Fatal("registry validator is nil")
	}
	if err := registry.POST("/users", mvc.BindJSON(http.StatusCreated, func(_ *arkweb.Context, input validatorRequest) (map[string]string, error) {
		return map[string]string{"name": input.Name}, nil
	})); err != nil {
		t.Fatalf("POST failed: %v", err)
	}
	router, err := registry.Router()
	if err != nil {
		t.Fatalf("Router failed: %v", err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(`{"name":"root"}`))
	request.Header.Set("Content-Type", arkjson.ContentType)
	servletnethttp.Handler(router).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422, body=%s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"code":"reserved"`) {
		t.Fatalf("body = %s, want custom validation code", recorder.Body.String())
	}
}

func TestRegisterValidatorRejectsNilValidator(t *testing.T) {
	t.Parallel()

	if err := web.RegisterValidator(container.NewRegistry(), "nilValidator", nil); !errors.Is(err, web.ErrNilValidator) {
		t.Fatalf("err = %v, want ErrNilValidator", err)
	}
}

type validatorRequest struct {
	Name string `json:"name"`
}
