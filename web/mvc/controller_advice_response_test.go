package mvc_test

import (
	"net/http"
	"strings"
	"testing"

	arkweb "goark.dev/arkarta/web"
	"goark.dev/goark/web"
	"goark.dev/goark/web/mvc"
)

type panicResponseAdvice struct{}

func (*panicResponseAdvice) BeforeWrite(_ *arkweb.Context, _ arkweb.Result) (arkweb.Result, error) {
	panic("nil response advice must not run")
}

func TestControllerAdviceResponseAdviceWrapsResponseBody(t *testing.T) {
	t.Parallel()

	registry := web.NewRegistry()
	advice := mvc.NewRestControllerAdvice("api-response").WithResponseAdvice(
		web.ResponseAdviceFunc(func(ctx *arkweb.Context, _ arkweb.Result) (arkweb.Result, error) {
			ctx.Response().Header().Set("X-Response-Advice", "configured")
			return arkweb.JSON(http.StatusAccepted, map[string]string{"status": "advised"}), nil
		}),
	)
	configurer := mvc.NewConfigurer(mvc.NewRestController("users",
		mvc.GET("/users", mvc.JSON(http.StatusOK, func(_ *arkweb.Context) (map[string]string, error) {
			return map[string]string{"status": "origin"}, nil
		})),
	)).WithControllerAdvices(advice)
	if err := configurer.ConfigureWeb(t.Context(), registry); err != nil {
		t.Fatalf("ConfigureWeb failed: %v", err)
	}

	recorder := serveMVCRegistry(t, registry, http.MethodGet, "/users")
	if recorder.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want 202, body=%s", recorder.Code, recorder.Body.String())
	}
	if got := recorder.Header().Get("X-Response-Advice"); got != "configured" {
		t.Fatalf("X-Response-Advice = %q, want configured", got)
	}
	if body := recorder.Body.String(); !strings.Contains(body, `"status":"advised"`) {
		t.Fatalf("body = %s, want advised payload", body)
	}
}

func TestControllerAdviceResponseAdviceSkipsNilValues(t *testing.T) {
	t.Parallel()

	var typedNil *panicResponseAdvice
	registry := web.NewRegistry()
	advice := mvc.NewRestControllerAdvice("api-response").WithResponseAdvice(nil, typedNil)
	configurer := mvc.NewConfigurer(mvc.NewRestController("users",
		mvc.GET("/users", mvc.Text(http.StatusOK, func(_ *arkweb.Context) (string, error) {
			return "origin", nil
		})),
	)).WithControllerAdvices(advice)
	if err := configurer.ConfigureWeb(t.Context(), registry); err != nil {
		t.Fatalf("ConfigureWeb failed: %v", err)
	}

	recorder := serveMVCRegistry(t, registry, http.MethodGet, "/users")
	if recorder.Code != http.StatusOK || recorder.Body.String() != "origin" {
		t.Fatalf("response = %d %q, want 200 origin", recorder.Code, recorder.Body.String())
	}
}
