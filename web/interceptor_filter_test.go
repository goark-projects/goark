package web_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"goark.dev/arkarta/servlet"
	servletnethttp "goark.dev/arkarta/servlet/nethttp"
	arkweb "goark.dev/arkarta/web"
	"goark.dev/goark/container"
	"goark.dev/goark/web"
)

func TestRegisterInterceptorContributesConfigurer(t *testing.T) {
	t.Parallel()

	beanRegistry := container.NewRegistry()
	if err := web.RegisterInterceptor(beanRegistry, "traceInterceptor", arkweb.InterceptorFunc(func(ctx *arkweb.Context, next arkweb.Handler) (arkweb.Result, error) {
		ctx.Response().Header().Set("X-Interceptor", "before")
		result, err := next.Handle(ctx)
		ctx.Response().Header().Set("X-Interceptor-After", "after")
		return result, err
	})); err != nil {
		t.Fatalf("RegisterInterceptor failed: %v", err)
	}
	resolver, err := container.New(beanRegistry)
	if err != nil {
		t.Fatalf("container.New failed: %v", err)
	}

	registry := web.NewRegistry()
	if err := web.ApplyConfigurers(t.Context(), resolver, registry); err != nil {
		t.Fatalf("ApplyConfigurers failed: %v", err)
	}
	if err := registry.GET("/trace", arkweb.HandlerFunc(func(_ *arkweb.Context) (arkweb.Result, error) {
		return arkweb.Text(http.StatusOK, "ok"), nil
	})); err != nil {
		t.Fatalf("GET failed: %v", err)
	}

	recorder := serveRegistry(t, registry, http.MethodGet, "/trace")
	if recorder.Header().Get("X-Interceptor") != "before" || recorder.Header().Get("X-Interceptor-After") != "after" {
		t.Fatalf("interceptor headers = %#v", recorder.Header())
	}
}

func TestRegisterFilterContributesConfigurer(t *testing.T) {
	t.Parallel()

	beanRegistry := container.NewRegistry()
	if err := web.RegisterFilter(beanRegistry, "traceFilter", servlet.FilterFunc(func(ctx context.Context, req *servlet.Request, res servlet.Response, chain servlet.Chain) error {
		res.Header().Set("X-Filter", "before")
		if err := chain.Next(ctx, req, res); err != nil {
			return err
		}
		res.Header().Set("X-Filter-After", "after")
		return nil
	})); err != nil {
		t.Fatalf("RegisterFilter failed: %v", err)
	}
	resolver, err := container.New(beanRegistry)
	if err != nil {
		t.Fatalf("container.New failed: %v", err)
	}

	registry := web.NewRegistry()
	if err := web.ApplyConfigurers(t.Context(), resolver, registry); err != nil {
		t.Fatalf("ApplyConfigurers failed: %v", err)
	}
	if err := registry.GET("/filtered", arkweb.HandlerFunc(func(_ *arkweb.Context) (arkweb.Result, error) {
		return arkweb.Text(http.StatusOK, "ok"), nil
	})); err != nil {
		t.Fatalf("GET failed: %v", err)
	}
	deployment, err := web.BuildDeployment(registry, web.DeploymentSpec{})
	if err != nil {
		t.Fatalf("BuildDeployment failed: %v", err)
	}
	handler, err := deployment.Handler()
	if err != nil {
		t.Fatalf("Deployment Handler failed: %v", err)
	}

	recorder := httptest.NewRecorder()
	servletnethttp.Handler(handler).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/filtered", nil))
	if recorder.Header().Get("X-Filter") != "before" || recorder.Header().Get("X-Filter-After") != "after" {
		t.Fatalf("filter headers = %#v", recorder.Header())
	}
}

func TestRegisterInterceptorAndFilterRejectNil(t *testing.T) {
	t.Parallel()

	if err := web.RegisterInterceptor(container.NewRegistry(), "nilInterceptor", nil); !errors.Is(err, web.ErrNilInterceptor) {
		t.Fatalf("interceptor err = %v, want ErrNilInterceptor", err)
	}
	if err := web.RegisterFilter(container.NewRegistry(), "nilFilter", nil); !errors.Is(err, web.ErrNilFilter) {
		t.Fatalf("filter err = %v, want ErrNilFilter", err)
	}
}
