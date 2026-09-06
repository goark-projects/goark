package mvc_test

import (
	"context"
	"net/http"
	"testing"
	"testing/fstest"

	arkweb "goark.dev/arkarta/web"

	"goark.dev/arkarta/servlet"
	"goark.dev/arkarta/servlet/session"
	"goark.dev/goark/web"
	"goark.dev/goark/web/message"
	"goark.dev/goark/web/mvc"
	"goark.dev/goark/web/mvc/flash"
	"goark.dev/goark/web/mvc/sessionattrs"
	"goark.dev/goark/web/mvc/view"
	webtest "goark.dev/goark/web/test"
)

func newFlashTestHandler(t *testing.T) servlet.Handler {
	t.Helper()
	registry := web.NewRegistry()
	if err := mvc.NewController("accounts",
		mvc.POST("/accounts", mvc.Return(0, func(_ *arkweb.Context) (mvc.ModelAndView, error) {
			attributes := mvc.NewRedirectAttributes().
				AddAttribute("id", "42").
				AddFlashAttribute("notice", "created")
			return mvc.Redirect("/accounts/created", attributes, mvc.WithViewStatus(
				http.StatusSeeOther)), nil
		})),
		mvc.GET("/accounts/created", mvc.Handler(func(ctx *arkweb.Context) (arkweb.Result, error) {
			notice, err := mvc.FlashAttribute[string](ctx, "notice", mvc.WithRequired(false))
			if err != nil {
				return nil, err
			}
			if notice == "" {
				return arkweb.NoContent(), nil
			}
			modelNotice, _ := mvc.CurrentModel(ctx).Attribute("notice")
			return arkweb.JSON(http.StatusOK, map[string]any{
				"notice":      notice,
				"modelNotice": modelNotice,
			}), nil
		})),
		mvc.GET("/other", mvc.NoContent(func(*arkweb.Context) error {
			return nil
		})),
	).Register(registry); err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	router, err := registry.Router()
	if err != nil {
		t.Fatalf("Router failed: %v", err)
	}
	accessor, err := session.NewAccessor(session.NewMemoryManager())
	if err != nil {
		t.Fatalf("NewAccessor failed: %v", err)
	}
	filter, err := flash.NewFilter(accessor)
	if err != nil {
		t.Fatalf("NewFilter failed: %v", err)
	}
	return servlet.ChainFilters(router, filter)
}

func newSessionAttributesTestHandler(t *testing.T) servlet.Handler {
	t.Helper()

	registry := web.NewRegistry()
	controller := mvc.NewRestController("wizard",
		mvc.POST("/wizard/start", mvc.NoContent(func(ctx *arkweb.Context) error {
			mvc.CurrentModel(ctx).AddAttribute("draft", "step1")
			return nil
		})),
		mvc.GET("/wizard/current", mvc.JSON(http.StatusOK, func(ctx *arkweb.Context) (
			sessionAttributesPayload, error) {
			draft, err := mvc.SessionAttribute[string](ctx, "draft", mvc.WithRequired(false))
			if err != nil {
				return sessionAttributesPayload{}, err
			}
			modelDraft, _ := mvc.CurrentModel(ctx).Attribute("draft")
			return sessionAttributesPayload{
				Draft:      draft,
				ModelDraft: stringAttribute(modelDraft),
			}, nil
		})),
		mvc.POST("/wizard/complete", mvc.NoContent(func(ctx *arkweb.Context) error {
			mvc.CurrentSessionStatus(ctx).SetComplete()
			return nil
		})),
	).WithSessionAttributes("draft")
	if err := controller.Register(registry); err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	router, err := registry.Router()
	if err != nil {
		t.Fatalf("Router failed: %v", err)
	}
	accessor, err := session.NewAccessor(session.NewMemoryManager())
	if err != nil {
		t.Fatalf("NewAccessor failed: %v", err)
	}
	filter, err := sessionattrs.NewFilter(accessor)
	if err != nil {
		t.Fatalf("NewFilter failed: %v", err)
	}
	return servlet.ChainFilters(router, filter)
}

func TestControllerReturnForwardViewNameDispatchesTarget(t *testing.T) {
	t.Parallel()

	registry := web.NewRegistry()
	if err := mvc.NewController("pages",
		mvc.GET("/source", mvc.Return(0, func(_ *arkweb.Context) (string, error) {
			return "forward:/target?from=source", nil
		})),
		mvc.GET("/target", mvc.ResponseBody(http.StatusAccepted, func(ctx *arkweb.Context) (string,
			error) {
			forwardURI, _ := ctx.Request().Attribute(servlet.AttributeForwardRequestURI)
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

func TestForwardViewControllerDispatchesTarget(t *testing.T) {
	t.Parallel()

	registry := web.NewRegistry()
	if err := mvc.NewController("pages",
		mvc.ForwardViewController("/source", "/target?from=view-controller"),
		mvc.GET("/target", mvc.ResponseBody(http.StatusAccepted, func(ctx *arkweb.Context) (string,
			error) {
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

func TestControllerReturnRedirectViewNameWritesRedirect(t *testing.T) {
	t.Parallel()

	registry := web.NewRegistry()
	if err := mvc.NewController("pages",
		mvc.GET("/accounts", mvc.Return(http.StatusOK, func(_ *arkweb.Context) (string, error) {
			return "redirect:/signin", nil
		})),
	).Register(registry); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	recorder := serveMVC(t, registry, http.MethodGet, "/accounts", "")
	if recorder.Code != http.StatusFound {
		t.Fatalf("status = %d, want 302", recorder.Code)
	}
	if got := recorder.Header().Get("Location"); got != "/signin" {
		t.Fatalf("Location = %q, want /signin", got)
	}
	if recorder.Body.Len() != 0 {
		t.Fatalf("body = %q, want empty", recorder.Body.String())
	}
}

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

func TestModelAndViewRedirectViewNameWritesRedirect(t *testing.T) {
	t.Parallel()

	registry := web.NewRegistry()
	if err := mvc.NewController("pages",
		mvc.GET("/accounts", mvc.Return(0, func(_ *arkweb.Context) (mvc.ModelAndView, error) {
			return mvc.NewModelAndView("redirect:/signin", nil, mvc.WithViewStatus(http.StatusSeeOther)), nil
		})),
	).Register(registry); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	recorder := serveMVC(t, registry, http.MethodGet, "/accounts", "")
	if recorder.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want 303", recorder.Code)
	}
	if got := recorder.Header().Get("Location"); got != "/signin" {
		t.Fatalf("Location = %q, want /signin", got)
	}
}

func TestRedirectViewControllerWritesRedirect(t *testing.T) {
	t.Parallel()

	registry := web.NewRegistry()
	if err := mvc.NewController("pages",
		mvc.RedirectViewController("/legacy", "/signin", mvc.WithViewControllerStatus(
			http.StatusPermanentRedirect)),
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

func stringAttribute(value any) string {
	if text, ok := value.(string); ok {
		return text
	}
	return ""
}

type sessionAttributesPayload struct {
	Draft      string `json:"draft"`
	ModelDraft string `json:"modelDraft"`
}
