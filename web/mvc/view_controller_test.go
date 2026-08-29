package mvc_test

import (
	"context"
	"net/http"
	"testing"
	"testing/fstest"

	arkweb "goark.dev/arkarta/web"
	"goark.dev/goark/web"
	"goark.dev/goark/web/mvc"
	"goark.dev/goark/web/mvc/view"
	webtest "goark.dev/goark/web/test"
)

func TestViewControllerRendersConfiguredView(t *testing.T) {
	t.Parallel()

	registry := web.NewRegistry()
	registry.Use(view.Interceptor(modelViewResolver(t, fstest.MapFS{
		"login.html": {Data: []byte("<h1>login</h1>")},
	})))
	if err := mvc.NewController("pages",
		mvc.ViewController("/login", "login"),
	).Register(registry); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	recorder := serveMVC(t, registry, http.MethodGet, "/login", "text/html")
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	if recorder.Body.String() != "<h1>login</h1>" {
		t.Fatalf("body = %q, want rendered view", recorder.Body.String())
	}
}

func TestViewControllerAppliesConfiguredStatus(t *testing.T) {
	t.Parallel()

	registry := web.NewRegistry()
	registry.Use(view.Interceptor(modelViewResolver(t, fstest.MapFS{
		"created.html": {Data: []byte("<h1>created</h1>")},
	})))
	if err := mvc.NewController("pages",
		mvc.ViewController("/created", "created", mvc.WithViewControllerStatus(http.StatusCreated)),
	).Register(registry); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	recorder := serveMVC(t, registry, http.MethodGet, "/created", "text/html")
	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201", recorder.Code)
	}
	if recorder.Body.String() != "<h1>created</h1>" {
		t.Fatalf("body = %q, want rendered view", recorder.Body.String())
	}
}

func TestRedirectViewControllerWritesRedirect(t *testing.T) {
	t.Parallel()

	registry := web.NewRegistry()
	if err := mvc.NewController("pages",
		mvc.RedirectViewController("/legacy", "/signin", mvc.WithViewControllerStatus(http.StatusPermanentRedirect)),
	).Register(registry); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	recorder := serveMVC(t, registry, http.MethodGet, "/legacy", "")
	if recorder.Code != http.StatusPermanentRedirect {
		t.Fatalf("status = %d, want 308", recorder.Code)
	}
	if got := recorder.Header().Get("Location"); got != "/signin" {
		t.Fatalf("Location = %q, want /signin", got)
	}
}

func TestForwardViewControllerDispatchesTarget(t *testing.T) {
	t.Parallel()

	registry := web.NewRegistry()
	if err := mvc.NewController("pages",
		mvc.ForwardViewController("/source", "/target?from=view-controller"),
		mvc.GET("/target", mvc.ResponseBody(http.StatusAccepted, func(ctx *arkweb.Context) (string, error) {
			return ctx.QueryValue("from"), nil
		})),
	).Register(registry); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	client, err := webtest.NewRegistry(t.Context(), registry, web.DeploymentSpec{})
	client = webtest.Must(t, client, err)
	t.Cleanup(func() {
		if err := client.Close(context.Background()); err != nil {
			t.Fatalf("Close failed: %v", err)
		}
	})

	client.Perform(t, http.MethodGet, "/source", webtest.WithAccept("text/plain")).
		ExpectStatus(t, http.StatusAccepted).
		ExpectBody(t, "view-controller")
}
