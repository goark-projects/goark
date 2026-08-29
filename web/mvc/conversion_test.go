package mvc_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	servletnethttp "goark.dev/arkarta/servlet/nethttp"
	arkweb "goark.dev/arkarta/web"
	"goark.dev/goark/container"
	"goark.dev/goark/core/convert"
	"goark.dev/goark/web"
	"goark.dev/goark/web/mvc"
)

type tenantID struct {
	value string
}

func TestConversionServiceAppliesToMVCParameters(t *testing.T) {
	t.Parallel()

	service, err := convert.NewService(
		convert.ConverterFunc[string, int](func(value string) (int, error) {
			return len(value) + 100, nil
		}),
		convert.ConverterFunc[string, tenantID](func(value string) (tenantID, error) {
			return tenantID{value: "tenant:" + value}, nil
		}),
	)
	if err != nil {
		t.Fatalf("NewService failed: %v", err)
	}
	beanRegistry := container.NewRegistry()
	if err := mvc.RegisterConversionService(beanRegistry, "testConversionService", service); err != nil {
		t.Fatalf("RegisterConversionService failed: %v", err)
	}
	resolver, err := container.New(beanRegistry)
	if err != nil {
		t.Fatalf("container.New failed: %v", err)
	}

	registry := web.NewRegistry()
	if err := web.ApplyConfigurers(t.Context(), resolver, registry); err != nil {
		t.Fatalf("ApplyConfigurers failed: %v", err)
	}
	if err := registry.GET("/items", mvc.JSON(http.StatusOK, func(ctx *arkweb.Context) (map[string]any, error) {
		page, err := mvc.RequestParamInt(ctx, "page")
		if err != nil {
			return nil, err
		}
		tenant, err := mvc.RequestParamAs[tenantID](ctx, "tenant")
		if err != nil {
			return nil, err
		}
		return map[string]any{
			"page":   page,
			"tenant": tenant.value,
		}, nil
	})); err != nil {
		t.Fatalf("GET failed: %v", err)
	}
	router, err := registry.Router()
	if err != nil {
		t.Fatalf("Router failed: %v", err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/items?page=abc&tenant=blue", nil)
	servletnethttp.Handler(router).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"page":103`) ||
		!strings.Contains(recorder.Body.String(), `"tenant":"tenant:blue"`) {
		t.Fatalf("body = %s, want converted parameters", recorder.Body.String())
	}
}
