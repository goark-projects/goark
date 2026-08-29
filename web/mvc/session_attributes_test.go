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
	"goark.dev/goark/web/mvc/sessionattrs"
)

type sessionAttributesPayload struct {
	Draft      string `json:"draft"`
	ModelDraft string `json:"modelDraft"`
}

func TestControllerSessionAttributesPersistUntilSessionStatusComplete(t *testing.T) {
	t.Parallel()

	handler := newSessionAttributesTestHandler(t)

	first := httptest.NewRecorder()
	servletnethttp.Handler(handler).ServeHTTP(first, httptest.NewRequest(http.MethodPost, "/wizard/start", nil))
	if first.Code != http.StatusNoContent {
		t.Fatalf("first status = %d, want 204", first.Code)
	}
	cookies := first.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("Set-Cookie count = %d, want 1", len(cookies))
	}
	sessionCookie := cookies[0]

	current := requestSessionAttributesPayload(t, handler, sessionCookie)
	if current.Draft != "step1" || current.ModelDraft != "step1" {
		t.Fatalf("current payload = %#v, want persisted draft", current)
	}

	complete := httptest.NewRecorder()
	completeReq := httptest.NewRequest(http.MethodPost, "/wizard/complete", nil)
	completeReq.AddCookie(sessionCookie)
	servletnethttp.Handler(handler).ServeHTTP(complete, completeReq)
	if complete.Code != http.StatusNoContent {
		t.Fatalf("complete status = %d, want 204", complete.Code)
	}

	consumed := requestSessionAttributesPayload(t, handler, sessionCookie)
	if consumed.Draft != "" || consumed.ModelDraft != "" {
		t.Fatalf("consumed payload = %#v, want cleared draft", consumed)
	}
}

func TestControllerSessionAttributesNormalizeNames(t *testing.T) {
	t.Parallel()

	controller := mvc.NewRestController("wizard").WithSessionAttributes(" draft ", "", "draft", "step")
	names := controller.SessionAttributes()
	if len(names) != 2 || names[0] != "draft" || names[1] != "step" {
		t.Fatalf("session attributes = %#v, want normalized unique names", names)
	}
}

func newSessionAttributesTestHandler(t *testing.T) servlet.Handler {
	t.Helper()

	registry := web.NewRegistry()
	controller := mvc.NewRestController("wizard",
		mvc.POST("/wizard/start", mvc.NoContent(func(ctx *arkweb.Context) error {
			mvc.CurrentModel(ctx).AddAttribute("draft", "step1")
			return nil
		})),
		mvc.GET("/wizard/current", mvc.JSON(http.StatusOK, func(ctx *arkweb.Context) (sessionAttributesPayload, error) {
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

func requestSessionAttributesPayload(t *testing.T, handler servlet.Handler, cookie *http.Cookie) sessionAttributesPayload {
	t.Helper()

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/wizard/current", nil)
	request.AddCookie(cookie)
	servletnethttp.Handler(handler).ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("current status = %d, want 200, body=%s", recorder.Code, recorder.Body.String())
	}
	var payload sessionAttributesPayload
	if err := arkjson.Unmarshal(nil, recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("current response json invalid: %v", err)
	}
	return payload
}

func stringAttribute(value any) string {
	if text, ok := value.(string); ok {
		return text
	}
	return ""
}
