package web_test

import (
	"errors"
	"net/http"
	"strings"
	"testing"

	arkweb "goark.dev/arkarta/web"
	"goark.dev/goark/web"
	"goark.dev/goark/web/problem"
)

func TestNewResponseStatusExceptionCreatesStatusError(t *testing.T) {
	t.Parallel()

	cause := errors.New("database duplicate key")
	err := web.NewResponseStatusException(http.StatusConflict, "job already exists", cause)

	var statusErr web.StatusError
	if !errors.As(err, &statusErr) {
		t.Fatalf("err = %T, want StatusError", err)
	}
	if statusErr.StatusCode() != http.StatusConflict {
		t.Fatalf("status = %d, want 409", statusErr.StatusCode())
	}
	if statusErr.PublicMessage() != "job already exists" {
		t.Fatalf("public message = %q, want job already exists", statusErr.PublicMessage())
	}
	if !errors.Is(err, cause) {
		t.Fatalf("err does not unwrap cause")
	}

	var exception *web.ResponseStatusException
	if !errors.As(err, &exception) {
		t.Fatalf("err = %T, want ResponseStatusException", err)
	}
}

func TestResponseStatusExceptionMapsToProblemDetail(t *testing.T) {
	t.Parallel()

	cause := errors.New("internal storage detail")
	registry := web.NewRegistry()
	registry.UseErrorMapper(problem.NewMapper())
	if err := registry.POST("/jobs", arkweb.HandlerFunc(func(_ *arkweb.Context) (arkweb.Result, error) {
		return nil, web.NewResponseStatusException(http.StatusConflict, "job already exists", cause)
	})); err != nil {
		t.Fatalf("POST failed: %v", err)
	}

	recorder := serveRegistry(t, registry, http.MethodPost, "/jobs")
	if recorder.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409, body=%s", recorder.Code, recorder.Body.String())
	}
	body := recorder.Body.String()
	if !strings.Contains(body, `"detail":"job already exists"`) ||
		!strings.Contains(body, `"error":"HTTP_409"`) {
		t.Fatalf("body = %s, want conflict problem detail", body)
	}
	if strings.Contains(body, cause.Error()) {
		t.Fatalf("body exposes cause: %s", body)
	}
}
