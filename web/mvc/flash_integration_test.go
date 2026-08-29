package mvc_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	arkjson "goark.dev/arkarta/json"
	"goark.dev/arkarta/servlet"
	servletnethttp "goark.dev/arkarta/servlet/nethttp"
	"goark.dev/arkarta/servlet/session"
	arkweb "goark.dev/arkarta/web"
	"goark.dev/goark/web"
	"goark.dev/goark/web/mvc"
	"goark.dev/goark/web/mvc/flash"
)

func TestRedirectAttributesCarryFlashAttributesOnce(t *testing.T) {
	t.Parallel()

	handler := newFlashTestHandler(t)

	first := httptest.NewRecorder()
	servletnethttp.Handler(handler).ServeHTTP(first, httptest.NewRequest(http.MethodPost, "/accounts", nil))
	if first.Code != http.StatusSeeOther {
		t.Fatalf("first status = %d, want 303", first.Code)
	}
	if got := first.Header().Get("Location"); got != "/accounts/created?id=42" {
		t.Fatalf("Location = %q, want redirect target", got)
	}
	cookies := first.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("Set-Cookie count = %d, want 1", len(cookies))
	}
	sessionCookie := cookies[0]

	unmatched := httptest.NewRecorder()
	unmatchedReq := httptest.NewRequest(http.MethodGet, "/other", nil)
	unmatchedReq.AddCookie(sessionCookie)
	servletnethttp.Handler(handler).ServeHTTP(unmatched, unmatchedReq)
	if unmatched.Code != http.StatusNoContent {
		t.Fatalf("unmatched status = %d, want 204, body=%s", unmatched.Code, unmatched.Body.String())
	}

	matched := httptest.NewRecorder()
	matchedReq := httptest.NewRequest(http.MethodGet, "/accounts/created?id=42", nil)
	matchedReq.AddCookie(sessionCookie)
	servletnethttp.Handler(handler).ServeHTTP(matched, matchedReq)
	if matched.Code != http.StatusOK {
		t.Fatalf("matched status = %d, want 200, body=%s", matched.Code, matched.Body.String())
	}
	var payload struct {
		Notice      string `json:"notice"`
		ModelNotice string `json:"modelNotice"`
	}
	if err := arkjson.Unmarshal(nil, matched.Body.Bytes(), &payload); err != nil {
		t.Fatalf("response json invalid: %v", err)
	}
	if payload.Notice != "created" || payload.ModelNotice != "created" {
		t.Fatalf("payload = %#v, want flash attributes", payload)
	}

	repeated := httptest.NewRecorder()
	repeatedReq := httptest.NewRequest(http.MethodGet, "/accounts/created?id=42", nil)
	repeatedReq.AddCookie(sessionCookie)
	servletnethttp.Handler(handler).ServeHTTP(repeated, repeatedReq)
	if repeated.Code != http.StatusNoContent {
		t.Fatalf("repeated status = %d, want 204, body=%s", repeated.Code, repeated.Body.String())
	}
}

func newFlashTestHandler(t *testing.T) servlet.Handler {
	t.Helper()
	registry := web.NewRegistry()
	if err := mvc.NewController("accounts",
		mvc.POST("/accounts", mvc.Return(0, func(_ *arkweb.Context) (mvc.ModelAndView, error) {
			attributes := mvc.NewRedirectAttributes().
				AddAttribute("id", "42").
				AddFlashAttribute("notice", "created")
			return mvc.Redirect("/accounts/created", attributes, mvc.WithViewStatus(http.StatusSeeOther)), nil
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
