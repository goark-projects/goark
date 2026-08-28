package view_test

import (
	"errors"
	"html/template"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	servletnethttp "goark.dev/arkarta/servlet/nethttp"
	arkweb "goark.dev/arkarta/web"
	"goark.dev/goark/web/mvc/view"
)

func TestTemplateResolverRendersViewWithExplicitResolver(t *testing.T) {
	t.Parallel()

	resolver := newTemplateResolver(t, fstest.MapFS{
		"home.html": &fstest.MapFile{Data: []byte("<h1>{{.Title}}</h1>")},
	})
	router := arkweb.NewRouter()
	if err := router.GET("/home", arkweb.HandlerFunc(func(_ *arkweb.Context) (arkweb.Result, error) {
		return view.Using(resolver, "home", map[string]string{"Title": "Goark"}, view.WithStatus(http.StatusAccepted)), nil
	})); err != nil {
		t.Fatalf("register route failed: %v", err)
	}

	recorder := serveViewRouter(router, http.MethodGet, "/home")
	if recorder.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusAccepted)
	}
	if got := recorder.Header().Get("Content-Type"); got != "text/html; charset=utf-8" {
		t.Fatalf("Content-Type = %q, want text/html", got)
	}
	if recorder.Body.String() != "<h1>Goark</h1>" {
		t.Fatalf("body = %q, want rendered html", recorder.Body.String())
	}
}

func TestTemplateResolverRendersViewFromRequestContext(t *testing.T) {
	t.Parallel()

	resolver := newTemplateResolver(t, fstest.MapFS{
		"pages/home.tmpl": &fstest.MapFile{Data: []byte("{{upper .Title}}")},
	}, view.WithPrefix("pages"), view.WithSuffix(".tmpl"), view.WithFuncs(template.FuncMap{
		"upper": strings.ToUpper,
	}))
	router := arkweb.NewRouter()
	router.Use(view.Interceptor(resolver))
	if err := router.GET("/home", arkweb.HandlerFunc(func(_ *arkweb.Context) (arkweb.Result, error) {
		return view.Render("home", map[string]string{"Title": "goark"}), nil
	})); err != nil {
		t.Fatalf("register route failed: %v", err)
	}

	recorder := serveViewRouter(router, http.MethodGet, "/home")
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if recorder.Body.String() != "GOARK" {
		t.Fatalf("body = %q, want GOARK", recorder.Body.String())
	}
}

func TestTemplateResolverRejectsUnsafeViewName(t *testing.T) {
	t.Parallel()

	resolver := newTemplateResolver(t, fstest.MapFS{
		"home.html": &fstest.MapFile{Data: []byte("home")},
	})
	_, _, err := resolver.ResolveView(nil, "../secret")
	if !errors.Is(err, view.ErrInvalidViewName) {
		t.Fatalf("err = %v, want ErrInvalidViewName", err)
	}
}

func TestNewTemplateResolverRequiresTemplates(t *testing.T) {
	t.Parallel()

	_, err := view.NewTemplateResolver(fstest.MapFS{})
	if !errors.Is(err, view.ErrNoTemplates) {
		t.Fatalf("err = %v, want ErrNoTemplates", err)
	}
}

func newTemplateResolver(t *testing.T, root fstest.MapFS, options ...view.TemplateOption) *view.TemplateResolver {
	t.Helper()

	resolver, err := view.NewTemplateResolver(root, options...)
	if err != nil {
		t.Fatalf("NewTemplateResolver failed: %v", err)
	}
	return resolver
}

func serveViewRouter(router *arkweb.Router, method string, target string) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	servletnethttp.Handler(router).ServeHTTP(recorder, httptest.NewRequest(method, target, nil))
	return recorder
}
