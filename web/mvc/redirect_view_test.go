package mvc_test

import (
	"net/http"
	"testing"

	arkweb "goark.dev/arkarta/web"
	"goark.dev/goark/web"
	"goark.dev/goark/web/message"
	"goark.dev/goark/web/mvc"
)

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

func TestRestControllerReturnRedirectViewNameWritesBody(t *testing.T) {
	t.Parallel()

	registry := web.NewRegistry()
	if err := mvc.NewRestController("api",
		mvc.GET("/accounts", mvc.Return(http.StatusOK, func(_ *arkweb.Context) (string, error) {
			return "redirect:/signin", nil
		})),
	).Register(registry); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	recorder := serveMVC(t, registry, http.MethodGet, "/accounts", "text/plain")
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	if recorder.Header().Get("Content-Type") != message.MediaTypeTextPlain {
		t.Fatalf("Content-Type = %q, want text/plain", recorder.Header().Get("Content-Type"))
	}
	if recorder.Body.String() != "redirect:/signin" {
		t.Fatalf("body = %q, want redirect view name as body", recorder.Body.String())
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

func TestModelAndViewRedirectExpandsModelAttributes(t *testing.T) {
	t.Parallel()

	registry := web.NewRegistry()
	if err := mvc.NewController("pages",
		mvc.GET("/accounts", mvc.Return(0, func(_ *arkweb.Context) (mvc.ModelAndView, error) {
			model := mvc.NewModel().
				AddAttribute("id", "a/b").
				AddAttribute("page", 2).
				AddAttribute("tab", "security")
			return mvc.NewModelAndView("redirect:/users/{id}", model, mvc.WithViewStatus(http.StatusSeeOther)), nil
		})),
	).Register(registry); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	recorder := serveMVC(t, registry, http.MethodGet, "/accounts", "")
	if recorder.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want 303", recorder.Code)
	}
	if got := recorder.Header().Get("Location"); got != "/users/a%2Fb?page=2&tab=security" {
		t.Fatalf("Location = %q, want expanded redirect", got)
	}
}

func TestRedirectAttributesBuildRedirectModelAndView(t *testing.T) {
	t.Parallel()

	registry := web.NewRegistry()
	if err := mvc.NewController("pages",
		mvc.GET("/accounts", mvc.Return(0, func(_ *arkweb.Context) (mvc.ModelAndView, error) {
			attributes := mvc.NewRedirectAttributes().
				AddAttribute("id", "42").
				AddAttribute("tab", "profile")
			return mvc.Redirect("/users/{id}", attributes, mvc.WithViewStatus(http.StatusTemporaryRedirect)), nil
		})),
	).Register(registry); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	recorder := serveMVC(t, registry, http.MethodGet, "/accounts", "")
	if recorder.Code != http.StatusTemporaryRedirect {
		t.Fatalf("status = %d, want 307", recorder.Code)
	}
	if got := recorder.Header().Get("Location"); got != "/users/42?tab=profile" {
		t.Fatalf("Location = %q, want redirect attributes", got)
	}
}

func TestModelAndViewRedirectReportsMissingPathVariable(t *testing.T) {
	t.Parallel()

	registry := web.NewRegistry()
	if err := mvc.NewController("pages",
		mvc.GET("/accounts", mvc.Return(0, func(_ *arkweb.Context) (mvc.ModelAndView, error) {
			return mvc.NewModelAndView("redirect:/users/{id}", mvc.NewModel()), nil
		})),
	).Register(registry); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	recorder := serveMVC(t, registry, http.MethodGet, "/accounts", "")
	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", recorder.Code)
	}
	if got := recorder.Header().Get("Location"); got != "" {
		t.Fatalf("Location = %q, want empty", got)
	}
}
