package problem_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"goark.dev/arkarta/servlet"
	servletnethttp "goark.dev/arkarta/servlet/nethttp"
	arkweb "goark.dev/arkarta/web"
	"goark.dev/goark/web"
	"goark.dev/goark/web/problem"
)

func TestResultWritesProblemJSON(t *testing.T) {
	t.Parallel()

	registry := web.NewRegistry()
	if err := registry.GET("/problems/1", arkweb.HandlerFunc(func(_ *arkweb.Context) (arkweb.Result, error) {
		return problem.New(http.StatusNotFound,
			problem.WithDetail("job 1 not found"),
			problem.WithInstance("/problems/1"),
			problem.WithExtension("code", "JOB_NOT_FOUND"),
		), nil
	})); err != nil {
		t.Fatalf("GET failed: %v", err)
	}

	recorder := serveProblemRegistry(t, registry, http.MethodGet, "/problems/1")
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", recorder.Code)
	}
	if got := recorder.Header().Get("Content-Type"); got != problem.MediaType {
		t.Fatalf("content type = %q, want problem json", got)
	}
	body := recorder.Body.String()
	for _, want := range []string{
		`"type":"about:blank"`,
		`"title":"Not Found"`,
		`"status":404`,
		`"detail":"job 1 not found"`,
		`"instance":"/problems/1"`,
		`"code":"JOB_NOT_FOUND"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("body = %s, want %s", body, want)
		}
	}
}

func TestMapperMapsStatusErrorToProblemDetail(t *testing.T) {
	t.Parallel()

	registry := web.NewRegistry()
	registry.UseErrorMapper(problem.NewMapper())
	if err := registry.GET("/jobs/1", arkweb.HandlerFunc(func(_ *arkweb.Context) (arkweb.Result, error) {
		return nil, servlet.NewHTTPError(http.StatusNotFound, "job not found", errors.New("db miss"))
	})); err != nil {
		t.Fatalf("GET failed: %v", err)
	}

	recorder := serveProblemRegistry(t, registry, http.MethodGet, "/jobs/1")
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", recorder.Code)
	}
	if got := recorder.Header().Get("Content-Type"); got != problem.MediaType {
		t.Fatalf("content type = %q, want problem json", got)
	}
	body := recorder.Body.String()
	if !strings.Contains(body, `"detail":"job not found"`) ||
		!strings.Contains(body, `"instance":"/jobs/1"`) ||
		!strings.Contains(body, `"error":"HTTP_404"`) {
		t.Fatalf("body = %s, want status problem detail", body)
	}
}

func TestMapperDoesNotExposeInternalErrorDetail(t *testing.T) {
	t.Parallel()

	registry := web.NewRegistry()
	registry.UseErrorMapper(problem.NewMapper())
	if err := registry.GET("/boom", arkweb.HandlerFunc(func(_ *arkweb.Context) (arkweb.Result, error) {
		return nil, errors.New("database password leaked")
	})); err != nil {
		t.Fatalf("GET failed: %v", err)
	}

	recorder := serveProblemRegistry(t, registry, http.MethodGet, "/boom")
	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", recorder.Code)
	}
	if strings.Contains(recorder.Body.String(), "database password leaked") {
		t.Fatalf("body exposes internal error: %s", recorder.Body.String())
	}
}

func serveProblemRegistry(t *testing.T, registry *web.Registry, method, target string) *httptest.ResponseRecorder {
	t.Helper()

	router, err := registry.Router()
	if err != nil {
		t.Fatalf("Router failed: %v", err)
	}
	recorder := httptest.NewRecorder()
	servletnethttp.Handler(router).ServeHTTP(recorder, httptest.NewRequest(method, target, nil))
	return recorder
}
