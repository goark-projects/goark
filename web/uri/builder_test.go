package uri_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"goark.dev/arkarta/servlet/nethttp"
	arkweb "goark.dev/arkarta/web"
	goweb "goark.dev/goark/web"
	"goark.dev/goark/web/uri"
)

func TestBuilderBuildsLocationWithTemplateExpansion(t *testing.T) {
	t.Parallel()

	got, err := uri.New().
		Scheme("https").
		Host("api.example.com").
		Path("/jobs").
		Path("/{id}").
		QueryParam("view", "full").
		BuildAndExpand(map[string]string{"id": "a/b"})
	if err != nil {
		t.Fatalf("BuildAndExpand failed: %v", err)
	}
	want := "https://api.example.com/jobs/a%2Fb?view=full"
	if got != want {
		t.Fatalf("uri = %q, want %q", got, want)
	}
}

func TestBuilderReturnsMissingPathVariable(t *testing.T) {
	t.Parallel()

	_, err := uri.New().Path("/jobs/{id}").BuildAndExpand(map[string]string{})
	if !errors.Is(err, uri.ErrMissingPathVariable) {
		t.Fatalf("error = %v, want ErrMissingPathVariable", err)
	}
}

func TestFromCurrentRequestURIBuildsCreatedLocation(t *testing.T) {
	t.Parallel()

	router := arkweb.NewRouter()
	if err := router.Handle(http.MethodPost, "/jobs", arkweb.HandlerFunc(func(ctx *arkweb.Context) (arkweb.Result, error) {
		location, err := uri.FromCurrentRequestURI(ctx).
			Path("/{id}").
			BuildAndExpand(map[string]string{"id": "42"})
		if err != nil {
			return nil, err
		}
		return goweb.NoBody(http.StatusCreated).WithLocation(location), nil
	})); err != nil {
		t.Fatalf("Handle failed: %v", err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "https://api.example.com/jobs?draft=true", nil)
	nethttp.Handler(router).ServeHTTP(recorder, request)
	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201", recorder.Code)
	}
	if got := recorder.Header().Get("Location"); got != "https://api.example.com/jobs/42" {
		t.Fatalf("Location = %q, want created resource URI", got)
	}
}

func TestFromCurrentRequestURIKeepsContextPath(t *testing.T) {
	t.Parallel()

	router := arkweb.NewRouter()
	if err := router.Handle(http.MethodPost, "/jobs", arkweb.HandlerFunc(func(ctx *arkweb.Context) (arkweb.Result, error) {
		location, err := uri.FromCurrentRequestURI(ctx).
			Path("/{id}").
			BuildAndExpand(map[string]string{"id": "42"})
		if err != nil {
			return nil, err
		}
		return goweb.NoBody(http.StatusCreated).WithLocation(location), nil
	})); err != nil {
		t.Fatalf("Handle failed: %v", err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "https://api.example.com/admin/jobs", nil)
	nethttp.HandlerWithOptions(router, nethttp.WithRequestContextPath("/admin")).ServeHTTP(recorder, request)
	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201", recorder.Code)
	}
	if got := recorder.Header().Get("Location"); got != "https://api.example.com/admin/jobs/42" {
		t.Fatalf("Location = %q, want context-aware URI", got)
	}
}

func TestFromParsesAndReplacesQueryParameters(t *testing.T) {
	t.Parallel()

	builder, err := uri.From("https://api.example.com/jobs?view=compact&trace=1")
	if err != nil {
		t.Fatalf("From failed: %v", err)
	}
	got := builder.
		ReplaceQueryParam("view", "full").
		ReplaceQueryParam("trace").
		Build()
	want := "https://api.example.com/jobs?view=full"
	if got != want {
		t.Fatalf("uri = %q, want %q", got, want)
	}
}
