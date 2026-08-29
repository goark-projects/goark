package mvc_test

import (
	"context"
	"net/http"
	"testing"

	"goark.dev/arkarta/servlet"
	arkweb "goark.dev/arkarta/web"
	"goark.dev/goark/web"
	"goark.dev/goark/web/message"
	"goark.dev/goark/web/mvc"
	webtest "goark.dev/goark/web/test"
)

func TestControllerReturnForwardViewNameDispatchesTarget(t *testing.T) {
	t.Parallel()

	registry := web.NewRegistry()
	if err := mvc.NewController("pages",
		mvc.GET("/source", mvc.Return(0, func(_ *arkweb.Context) (string, error) {
			return "forward:/target?from=source", nil
		})),
		mvc.GET("/target", mvc.ResponseBody(http.StatusAccepted, func(ctx *arkweb.Context) (string, error) {
			forwardURI, ok := ctx.Request().Attribute(servlet.AttributeForwardRequestURI)
			uri, ok := forwardURI.(string)
			if !ok {
				return "missing-forward-attribute", nil
			}
			return ctx.QueryValue("from") + ":" + uri, nil
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
		ExpectHeader(t, "Content-Type", message.MediaTypeTextPlain).
		ExpectBody(t, "source:/source")
}

func TestRestControllerReturnForwardViewNameWritesBody(t *testing.T) {
	t.Parallel()

	registry := web.NewRegistry()
	if err := mvc.NewRestController("api",
		mvc.GET("/source", mvc.Return(http.StatusOK, func(_ *arkweb.Context) (string, error) {
			return "forward:/target", nil
		})),
	).Register(registry); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	recorder := serveMVC(t, registry, http.MethodGet, "/source", "text/plain")
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	if recorder.Header().Get("Content-Type") != message.MediaTypeTextPlain {
		t.Fatalf("Content-Type = %q, want text/plain", recorder.Header().Get("Content-Type"))
	}
	if recorder.Body.String() != "forward:/target" {
		t.Fatalf("body = %q, want forward view name as body", recorder.Body.String())
	}
}

func TestModelAndViewForwardViewNameDispatchesTarget(t *testing.T) {
	t.Parallel()

	registry := web.NewRegistry()
	if err := mvc.NewController("pages",
		mvc.GET("/source", mvc.Return(0, func(_ *arkweb.Context) (mvc.ModelAndView, error) {
			model := mvc.NewModel().AddAttribute("notice", "saved")
			return mvc.NewModelAndView("forward:/target", model), nil
		})),
		mvc.GET("/target", mvc.ResponseBody(http.StatusOK, func(ctx *arkweb.Context) (string, error) {
			value, ok := mvc.CurrentModel(ctx).Attribute("notice")
			notice, ok := value.(string)
			if !ok {
				return "missing-notice", nil
			}
			return notice, nil
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
		ExpectStatus(t, http.StatusOK).
		ExpectBody(t, "saved")
}
