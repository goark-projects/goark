package mvc_test

import (
	"errors"
	"net/http"
	"reflect"
	"testing"

	arkweb "goark.dev/arkarta/web"
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
