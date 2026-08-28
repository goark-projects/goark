package mvc_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	arkjson "goark.dev/arkarta/json"
	servletnethttp "goark.dev/arkarta/servlet/nethttp"
	arkweb "goark.dev/arkarta/web"
	"goark.dev/goark/web/mvc"
)

func TestParameterHelpersBindRequestSources(t *testing.T) {
	t.Parallel()

	router := arkweb.NewRouter()
	err := router.Handle(http.MethodGet, "/users/{id}", mvc.JSON(http.StatusOK, func(ctx *arkweb.Context) (map[string]any, error) {
		id, err := mvc.PathInt64(ctx, "id")
		if err != nil {
			return nil, err
		}
		page, err := mvc.RequestParamInt(ctx, "page", mvc.WithDefaultValue("1"))
		if err != nil {
			return nil, err
		}
		requestID, err := mvc.RequestHeaderString(ctx, "X-Request-ID")
		if err != nil {
			return nil, err
		}
		theme, err := mvc.CookieValueString(ctx, "theme", mvc.WithRequired(false))
		if err != nil {
			return nil, err
		}
		return map[string]any{
			"id":        id,
			"page":      page,
			"requestID": requestID,
			"theme":     theme,
		}, nil
	}))
	if err != nil {
		t.Fatalf("Handle failed: %v", err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/users/42", nil)
	request.Header.Set("X-Request-ID", "req-1")
	request.AddCookie(&http.Cookie{Name: "theme", Value: "dark"})
	servletnethttp.Handler(router).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", recorder.Code, recorder.Body.String())
	}
	var payload struct {
		ID        int64  `json:"id"`
		Page      int    `json:"page"`
		RequestID string `json:"requestID"`
		Theme     string `json:"theme"`
	}
	if err := arkjson.Unmarshal(nil, recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("response json invalid: %v", err)
	}
	if payload.ID != 42 || payload.Page != 1 || payload.RequestID != "req-1" || payload.Theme != "dark" {
		t.Fatalf("payload = %#v", payload)
	}
}

func TestParameterHelpersRejectMissingRequiredParameter(t *testing.T) {
	t.Parallel()

	router := arkweb.NewRouter()
	err := router.Handle(http.MethodGet, "/users", mvc.JSON(http.StatusOK, func(ctx *arkweb.Context) (map[string]string, error) {
		query, err := mvc.RequestParamString(ctx, "q")
		if err != nil {
			return nil, err
		}
		return map[string]string{"q": query}, nil
	}))
	if err != nil {
		t.Fatalf("Handle failed: %v", err)
	}

	recorder := httptest.NewRecorder()
	servletnethttp.Handler(router).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/users", nil))
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400, body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestModelAttributeBindsQueryAndFormValues(t *testing.T) {
	t.Parallel()

	type userSearchCriteria struct {
		Username        string `form:"username" json:"username" arkarta:"required,min=2"`
		Page            int    `form:"page" json:"page"`
		IncludeDisabled bool   `form:"includeDisabled" json:"includeDisabled"`
	}
	router := arkweb.NewRouter()
	err := router.Handle(http.MethodPost, "/users/search", mvc.JSON(http.StatusOK, func(ctx *arkweb.Context) (map[string]any, error) {
		criteria, err := mvc.ModelAttribute[userSearchCriteria](ctx)
		if err != nil {
			return nil, err
		}
		return map[string]any{
			"username":        criteria.Username,
			"page":            criteria.Page,
			"includeDisabled": criteria.IncludeDisabled,
		}, nil
	}))
	if err != nil {
		t.Fatalf("Handle failed: %v", err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/users/search?username=ad", strings.NewReader("page=2&includeDisabled=true"))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	servletnethttp.Handler(router).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", recorder.Code, recorder.Body.String())
	}
	var payload struct {
		Username        string `json:"username"`
		Page            int    `json:"page"`
		IncludeDisabled bool   `json:"includeDisabled"`
	}
	if err := arkjson.Unmarshal(nil, recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("response json invalid: %v", err)
	}
	if payload.Username != "ad" || payload.Page != 2 || !payload.IncludeDisabled {
		t.Fatalf("payload = %#v", payload)
	}
}

func TestModelAttributeMapsValidationErrors(t *testing.T) {
	t.Parallel()

	type userSearchCriteria struct {
		Username string `form:"username" json:"username" arkarta:"required"`
	}
	router := arkweb.NewRouter()
	err := router.Handle(http.MethodGet, "/users/search", mvc.JSON(http.StatusOK, func(ctx *arkweb.Context) (map[string]string, error) {
		criteria, err := mvc.ModelAttribute[userSearchCriteria](ctx)
		if err != nil {
			return nil, err
		}
		return map[string]string{"username": criteria.Username}, nil
	}))
	if err != nil {
		t.Fatalf("Handle failed: %v", err)
	}

	recorder := httptest.NewRecorder()
	servletnethttp.Handler(router).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/users/search", nil))
	if recorder.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422, body=%s", recorder.Code, recorder.Body.String())
	}
}
