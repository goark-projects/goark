package mvc_test

import (
	"errors"
	"net/http"
	"reflect"
	"testing"

	arkweb "goark.dev/arkarta/web"
	"goark.dev/goark/container"
	"goark.dev/goark/web"
	"goark.dev/goark/web/mvc"
)

func TestHandlerInterceptorAdapterRunsLifecycle(t *testing.T) {
	t.Parallel()

	calls := make([]string, 0, 3)
	registry := web.NewRegistry()
	registry.Use(mvc.HandlerInterceptorAdapter(mvc.HandlerInterceptorFuncs{
		PreHandleFunc: func(ctx *arkweb.Context) (bool, error) {
			calls = append(calls, "pre")
			ctx.Response().Header().Set("X-Pre-Handle", "hit")
			return true, nil
		},
		PostHandleFunc: func(ctx *arkweb.Context, _ arkweb.Result) (arkweb.Result, error) {
			calls = append(calls, "post")
			ctx.Response().Header().Set("X-Post-Handle", "hit")
			return arkweb.Text(http.StatusAccepted, "post"), nil
		},
		AfterCompletionFunc: func(_ *arkweb.Context, err error) {
			if err != nil {
				t.Fatalf("afterCompletion err = %v, want nil", err)
			}
			calls = append(calls, "after")
		},
	}))
	if err := registry.GET("/intercepted", mvc.Text(http.StatusOK, func(_ *arkweb.Context) (string, error) {
		return "origin", nil
	})); err != nil {
		t.Fatalf("GET failed: %v", err)
	}

	recorder := serveMVCRegistry(t, registry, http.MethodGet, "/intercepted")
	if recorder.Code != http.StatusAccepted || recorder.Body.String() != "post" {
		t.Fatalf("response = %d %q, want 202 post", recorder.Code, recorder.Body.String())
	}
	if recorder.Header().Get("X-Pre-Handle") != "hit" || recorder.Header().Get("X-Post-Handle") != "hit" {
		t.Fatalf("headers = %#v, want pre and post markers", recorder.Header())
	}
	if want := []string{"pre", "post", "after"}; !reflect.DeepEqual(calls, want) {
		t.Fatalf("calls = %#v, want %#v", calls, want)
	}
}

func TestHandlerInterceptorAdapterCanShortCircuit(t *testing.T) {
	t.Parallel()

	handlerCalled := false
	registry := web.NewRegistry()
	registry.Use(mvc.HandlerInterceptorAdapter(mvc.HandlerInterceptorFuncs{
		PreHandleFunc: func(ctx *arkweb.Context) (bool, error) {
			ctx.Response().SetStatus(http.StatusNoContent)
			ctx.Response().Header().Set("X-Pre-Handle", "stopped")
			return false, nil
		},
		PostHandleFunc: func(*arkweb.Context, arkweb.Result) (arkweb.Result, error) {
			t.Fatal("postHandle must not run after preHandle short-circuit")
			return nil, nil
		},
		AfterCompletionFunc: func(*arkweb.Context, error) {
			t.Fatal("afterCompletion must not run after preHandle short-circuit")
		},
	}))
	if err := registry.GET("/blocked", mvc.Text(http.StatusOK, func(_ *arkweb.Context) (string, error) {
		handlerCalled = true
		return "origin", nil
	})); err != nil {
		t.Fatalf("GET failed: %v", err)
	}

	recorder := serveMVCRegistry(t, registry, http.MethodGet, "/blocked")
	if recorder.Code != http.StatusNoContent || recorder.Body.String() != "" {
		t.Fatalf("response = %d %q, want 204 empty", recorder.Code, recorder.Body.String())
	}
	if got := recorder.Header().Get("X-Pre-Handle"); got != "stopped" {
		t.Fatalf("X-Pre-Handle = %q, want stopped", got)
	}
	if handlerCalled {
		t.Fatal("handler must not run after preHandle short-circuit")
	}
}

func TestHandlerInterceptorAdapterRunsAfterCompletionOnHandlerError(t *testing.T) {
	t.Parallel()

	boom := errors.New("handler failed")
	var completedErr error
	registry := web.NewRegistry()
	registry.Use(mvc.HandlerInterceptorAdapter(mvc.HandlerInterceptorFuncs{
		AfterCompletionFunc: func(_ *arkweb.Context, err error) {
			completedErr = err
		},
	}))
	if err := registry.GET("/failed", mvc.Text(http.StatusOK, func(_ *arkweb.Context) (string, error) {
		return "", boom
	})); err != nil {
		t.Fatalf("GET failed: %v", err)
	}

	recorder := serveMVCRegistry(t, registry, http.MethodGet, "/failed")
	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", recorder.Code)
	}
	if !errors.Is(completedErr, boom) {
		t.Fatalf("afterCompletion err = %v, want handler error", completedErr)
	}
}

func TestRegisterHandlerInterceptorContributesConfigurer(t *testing.T) {
	t.Parallel()

	beanRegistry := container.NewRegistry()
	if err := mvc.RegisterHandlerInterceptor(beanRegistry, "traceHandlerInterceptor", mvc.HandlerInterceptorFuncs{
		PreHandleFunc: func(ctx *arkweb.Context) (bool, error) {
			ctx.Response().Header().Set("X-Handler-Interceptor", "hit")
			return true, nil
		},
	}); err != nil {
		t.Fatalf("RegisterHandlerInterceptor failed: %v", err)
	}
	resolver, err := container.New(beanRegistry)
	if err != nil {
		t.Fatalf("container.New failed: %v", err)
	}

	registry := web.NewRegistry()
	if err := web.ApplyConfigurers(t.Context(), resolver, registry); err != nil {
		t.Fatalf("ApplyConfigurers failed: %v", err)
	}
	if err := registry.GET("/registered", mvc.Text(http.StatusOK, func(_ *arkweb.Context) (string, error) {
		return "ok", nil
	})); err != nil {
		t.Fatalf("GET failed: %v", err)
	}

	recorder := serveMVCRegistry(t, registry, http.MethodGet, "/registered")
	if got := recorder.Header().Get("X-Handler-Interceptor"); got != "hit" {
		t.Fatalf("X-Handler-Interceptor = %q, want hit", got)
	}
}

func TestRegisterMappedHandlerInterceptorContributesScopedConfigurer(t *testing.T) {
	t.Parallel()

	mapping, err := web.NewInterceptorMapping(
		web.WithInterceptorPathPatterns("/api/**"),
		web.WithInterceptorExcludePathPatterns("/api/public/**"),
	)
	if err != nil {
		t.Fatalf("NewInterceptorMapping failed: %v", err)
	}
	beanRegistry := container.NewRegistry()
	if err := mvc.RegisterMappedHandlerInterceptor(beanRegistry, "apiHandlerInterceptor", mvc.HandlerInterceptorFuncs{
		PreHandleFunc: func(ctx *arkweb.Context) (bool, error) {
			ctx.Response().Header().Set("X-Mapped-Handler-Interceptor", "hit")
			return true, nil
		},
	}, mapping); err != nil {
		t.Fatalf("RegisterMappedHandlerInterceptor failed: %v", err)
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
		if err := registry.GET(target, mvc.Text(http.StatusOK, func(_ *arkweb.Context) (string, error) {
			return "ok", nil
		})); err != nil {
			t.Fatalf("GET %s failed: %v", target, err)
		}
	}

	matched := serveMVCRegistry(t, registry, http.MethodGet, "/api/users")
	if got := matched.Header().Get("X-Mapped-Handler-Interceptor"); got != "hit" {
		t.Fatalf("matched header = %q, want hit", got)
	}
	excluded := serveMVCRegistry(t, registry, http.MethodGet, "/api/public/ping")
	if got := excluded.Header().Get("X-Mapped-Handler-Interceptor"); got != "" {
		t.Fatalf("excluded header = %q, want empty", got)
	}
	unmatched := serveMVCRegistry(t, registry, http.MethodGet, "/admin")
	if got := unmatched.Header().Get("X-Mapped-Handler-Interceptor"); got != "" {
		t.Fatalf("unmatched header = %q, want empty", got)
	}
}

func TestConfigurerAppliesHandlerInterceptors(t *testing.T) {
	t.Parallel()

	registry := web.NewRegistry()
	configurer := mvc.NewConfigurer(mvc.NewRestController("accounts",
		mvc.GET("/accounts", mvc.Text(http.StatusOK, func(_ *arkweb.Context) (string, error) {
			return "ok", nil
		})),
	)).WithHandlerInterceptors(mvc.HandlerInterceptorFuncs{
		PreHandleFunc: func(ctx *arkweb.Context) (bool, error) {
			ctx.Response().Header().Set("X-MVC-Configurer-Interceptor", "hit")
			return true, nil
		},
	})
	if err := configurer.ConfigureWeb(t.Context(), registry); err != nil {
		t.Fatalf("ConfigureWeb failed: %v", err)
	}

	recorder := serveMVCRegistry(t, registry, http.MethodGet, "/accounts")
	if got := recorder.Header().Get("X-MVC-Configurer-Interceptor"); got != "hit" {
		t.Fatalf("X-MVC-Configurer-Interceptor = %q, want hit", got)
	}
}

func TestConfigurationAppliesMappedHandlerInterceptors(t *testing.T) {
	t.Parallel()

	mapping, err := web.NewInterceptorMapping(
		web.WithInterceptorPathPatterns("/api/**"),
		web.WithInterceptorExcludePathPatterns("/api/public/**"),
	)
	if err != nil {
		t.Fatalf("NewInterceptorMapping failed: %v", err)
	}
	beanRegistry := container.NewRegistry()
	configuration := mvc.NewConfiguration("api", mvc.NewRestController("accounts",
		mvc.GET("/api/accounts", mvc.Text(http.StatusOK, func(_ *arkweb.Context) (string, error) {
			return "ok", nil
		})),
		mvc.GET("/api/public/ping", mvc.Text(http.StatusOK, func(_ *arkweb.Context) (string, error) {
			return "ok", nil
		})),
	)).WithMappedHandlerInterceptor(mvc.HandlerInterceptorFuncs{
		PreHandleFunc: func(ctx *arkweb.Context) (bool, error) {
			ctx.Response().Header().Set("X-MVC-Configuration-Interceptor", "hit")
			return true, nil
		},
	}, mapping)
	if err := configuration.Register(t.Context(), beanRegistry); err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	resolver, err := container.New(beanRegistry)
	if err != nil {
		t.Fatalf("container.New failed: %v", err)
	}

	registry := web.NewRegistry()
	if err := web.ApplyConfigurers(t.Context(), resolver, registry); err != nil {
		t.Fatalf("ApplyConfigurers failed: %v", err)
	}
	matched := serveMVCRegistry(t, registry, http.MethodGet, "/api/accounts")
	if got := matched.Header().Get("X-MVC-Configuration-Interceptor"); got != "hit" {
		t.Fatalf("matched header = %q, want hit", got)
	}
	excluded := serveMVCRegistry(t, registry, http.MethodGet, "/api/public/ping")
	if got := excluded.Header().Get("X-MVC-Configuration-Interceptor"); got != "" {
		t.Fatalf("excluded header = %q, want empty", got)
	}
}
