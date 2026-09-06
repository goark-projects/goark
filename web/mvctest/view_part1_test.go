package mvc_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"

	arkjson "goark.dev/arkarta/json"
	servletnethttp "goark.dev/arkarta/servlet/nethttp"
	arkweb "goark.dev/arkarta/web"

	"goark.dev/arkarta/servlet"
	"goark.dev/goark/web"
	"goark.dev/goark/web/message"
	"goark.dev/goark/web/mvc"
	"goark.dev/goark/web/mvc/view"
	webtest "goark.dev/goark/web/test"
)

func TestRedirectAttributesCarryFlashAttributesOnce(t *testing.T) {
	t.Parallel()

	handler := newFlashTestHandler(t)

	first := httptest.NewRecorder()
	servletnethttp.Handler(handler).ServeHTTP(first, httptest.NewRequest(http.MethodPost,
		"/accounts", nil))
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

func TestControllerSessionAttributesPersistUntilSessionStatusComplete(t *testing.T) {
	t.Parallel()

	handler := newSessionAttributesTestHandler(t)

	first := httptest.NewRecorder()
	servletnethttp.Handler(handler).ServeHTTP(first, httptest.NewRequest(http.MethodPost,
		"/wizard/start", nil))
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

func TestModelAndViewForwardViewNameDispatchesTarget(t *testing.T) {
	t.Parallel()

	registry := web.NewRegistry()
	if err := mvc.NewController("pages",
		mvc.GET("/source", mvc.Return(0, func(_ *arkweb.Context) (mvc.ModelAndView, error) {
			model := mvc.NewModel().AddAttribute("notice", "saved")
			return mvc.NewModelAndView("forward:/target", model), nil
		})),
		mvc.GET("/target", mvc.ResponseBody(http.StatusOK, func(ctx *arkweb.Context) (string, error) {
			value, _ := mvc.CurrentModel(ctx).Attribute("notice")
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

func TestModelAndViewRedirectExpandsModelAttributes(t *testing.T) {
	t.Parallel()

	registry := web.NewRegistry()
	if err := mvc.NewController("pages",
		mvc.GET("/accounts", mvc.Return(0, func(_ *arkweb.Context) (mvc.ModelAndView, error) {
			model := mvc.NewModel().
				AddAttribute("id", "a/b").
				AddAttribute("page", 2).
				AddAttribute("tab", "security")
			return mvc.NewModelAndView("redirect:/users/{id}", model, mvc.WithViewStatus(
				http.StatusSeeOther)), nil
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
			return mvc.Redirect("/users/{id}", attributes, mvc.WithViewStatus(
				http.StatusTemporaryRedirect)), nil
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

func requestSessionAttributesPayload(t *testing.T, handler servlet.Handler,
	cookie *http.Cookie) sessionAttributesPayload {
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

func TestControllerSessionAttributesNormalizeNames(t *testing.T) {
	t.Parallel()

	controller := mvc.NewRestController("wizard").WithSessionAttributes(" draft ", "", "draft", "step")
	names := controller.SessionAttributes()
	if len(names) != 2 || names[0] != "draft" || names[1] != "step" {
		t.Fatalf("session attributes = %#v, want normalized unique names", names)
	}
}
