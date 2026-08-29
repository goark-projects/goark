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

func TestRegisterMappedInterceptorContributesScopedConfigurer(t *testing.T) {
	t.Parallel()

	mapping, err := web.NewInterceptorMapping(
		web.WithInterceptorPathPatterns("/api/**"),
		web.WithInterceptorExcludePathPatterns("/api/public/**"),
	)
	if err != nil {
		t.Fatalf("NewInterceptorMapping failed: %v", err)
	}
	beanRegistry := container.NewRegistry()
	if err := web.RegisterMappedInterceptor(beanRegistry, "apiInterceptor", arkweb.InterceptorFunc(func(ctx *arkweb.Context, next arkweb.Handler) (arkweb.Result, error) {
		ctx.Response().Header().Set("X-Scoped-Interceptor", "hit")
		return next.Handle(ctx)
	}), mapping); err != nil {
		t.Fatalf("RegisterMappedInterceptor failed: %v", err)
	}
	resolver, err := container.New(beanRegistry)
	if err != nil {
		t.Fatalf("container.New failed: %v", err)
	}

	registry := web.NewRegistry()
	if err := web.ApplyConfigurers(t.Context(), resolver, registry); err != nil {
		t.Fatalf("ApplyConfigurers failed: %v", err)
	}
	for _, target := range []string{"/api/users", "/api/public/ping", "/admin"} {
		if err := registry.GET(target, arkweb.HandlerFunc(func(_ *arkweb.Context) (arkweb.Result, error) {
			return arkweb.Text(http.StatusOK, "ok"), nil
		})); err != nil {
			t.Fatalf("GET %s failed: %v", target, err)
		}
	}

	matched := serveRegistry(t, registry, http.MethodGet, "/api/users")
	if got := matched.Header().Get("X-Scoped-Interceptor"); got != "hit" {
		t.Fatalf("matched header = %q, want hit", got)
	}
	excluded := serveRegistry(t, registry, http.MethodGet, "/api/public/ping")
	if got := excluded.Header().Get("X-Scoped-Interceptor"); got != "" {
		t.Fatalf("excluded header = %q, want empty", got)
	}
	unmatched := serveRegistry(t, registry, http.MethodGet, "/admin")
	if got := unmatched.Header().Get("X-Scoped-Interceptor"); got != "" {
		t.Fatalf("unmatched header = %q, want empty", got)
	}
}

func TestRegisterMappedFilterContributesScopedConfigurer(t *testing.T) {
	t.Parallel()

	mapping, err := web.NewFilterMapping(
		web.WithFilterPathPatterns("/secure/**"),
		web.WithFilterExcludePathPatterns("/secure/public/**"),
	)
	if err != nil {
		t.Fatalf("NewFilterMapping failed: %v", err)
	}
	beanRegistry := container.NewRegistry()
	if err := web.RegisterMappedFilter(beanRegistry, "secureFilter", servlet.FilterFunc(func(ctx context.Context, req *servlet.Request, res servlet.Response, chain servlet.Chain) error {
		res.Header().Set("X-Scoped-Filter", "hit")
		return chain.Next(ctx, req, res)
	}), mapping); err != nil {
		t.Fatalf("RegisterMappedFilter failed: %v", err)
	}
	resolver, err := container.New(beanRegistry)
	if err != nil {
		t.Fatalf("container.New failed: %v", err)
	}

	registry := web.NewRegistry()
	if err := web.ApplyConfigurers(t.Context(), resolver, registry); err != nil {
		t.Fatalf("ApplyConfigurers failed: %v", err)
	}
	for _, target := range []string{"/secure/data", "/secure/public/info", "/health"} {
		if err := registry.GET(target, arkweb.HandlerFunc(func(_ *arkweb.Context) (arkweb.Result, error) {
			return arkweb.Text(http.StatusOK, "ok"), nil
		})); err != nil {
			t.Fatalf("GET %s failed: %v", target, err)
		}
	}
	deployment, err := web.BuildDeployment(registry, web.DeploymentSpec{})
	if err != nil {
		t.Fatalf("BuildDeployment failed: %v", err)
	}
	handler, err := deployment.Handler()
	if err != nil {
		t.Fatalf("Deployment Handler failed: %v", err)
	}

	matched := httptest.NewRecorder()
	servletnethttp.Handler(handler).ServeHTTP(matched, httptest.NewRequest(http.MethodGet, "/secure/data", nil))
	if got := matched.Header().Get("X-Scoped-Filter"); got != "hit" {
		t.Fatalf("matched header = %q, want hit", got)
	}
	excluded := httptest.NewRecorder()
	servletnethttp.Handler(handler).ServeHTTP(excluded, httptest.NewRequest(http.MethodGet, "/secure/public/info", nil))
	if got := excluded.Header().Get("X-Scoped-Filter"); got != "" {
		t.Fatalf("excluded header = %q, want empty", got)
	}
	unmatched := httptest.NewRecorder()
	servletnethttp.Handler(handler).ServeHTTP(unmatched, httptest.NewRequest(http.MethodGet, "/health", nil))
	if got := unmatched.Header().Get("X-Scoped-Filter"); got != "" {
		t.Fatalf("unmatched header = %q, want empty", got)
	}
}

func TestInterceptorMappingSupportsAntStyleDoubleWildcard(t *testing.T) {
	t.Parallel()

	mapping, err := web.NewInterceptorMapping(
		web.WithInterceptorPathPatterns("/api/**/admin", "/files/*/meta"),
		web.WithInterceptorExcludePathPatterns("/api/**/internal"),
	)
	if err != nil {
		t.Fatalf("NewInterceptorMapping failed: %v", err)
	}

	cases := []struct {
		path  string
		match bool
	}{
		{path: "/api/admin", match: true},
		{path: "/api/v1/users/admin", match: true},
		{path: "/api/v1/internal", match: false},
		{path: "/files/42/meta", match: true},
		{path: "/files/42/detail/meta", match: false},
		{path: "/health", match: false},
	}
	for _, tc := range cases {
		if got := mapping.Matches(tc.path); got != tc.match {
			t.Fatalf("Matches(%q) = %t, want %t", tc.path, got, tc.match)
		}
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
	if _, err := web.NewInterceptorMapping(web.WithInterceptorPathPatterns("/api/**suffix/bad")); !errors.Is(err, web.ErrInvalidInterceptorMapping) {
		t.Fatalf("mapping err = %v, want ErrInvalidInterceptorMapping", err)
	}
}
