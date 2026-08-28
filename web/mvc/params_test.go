package mvc_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	arkjson "goark.dev/arkarta/json"
	"goark.dev/arkarta/servlet"
	servletnethttp "goark.dev/arkarta/servlet/nethttp"
	"goark.dev/arkarta/servlet/session"
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

func TestParameterHelpersBindRequestAndSessionAttributes(t *testing.T) {
	t.Parallel()

	router := arkweb.NewRouter()
	err := router.Handle(http.MethodGet, "/attributes", mvc.JSON(http.StatusOK, func(ctx *arkweb.Context) (map[string]any, error) {
		traceID, err := mvc.RequestAttributeString(ctx, "traceID")
		if err != nil {
			return nil, err
		}
		attempt, err := mvc.RequestAttributeInt(ctx, "attempt")
		if err != nil {
			return nil, err
		}
		principal, err := mvc.SessionAttributeString(ctx, "principal")
		if err != nil {
			return nil, err
		}
		return map[string]any{
			"traceID":   traceID,
			"attempt":   attempt,
			"principal": principal,
		}, nil
	}))
	if err != nil {
		t.Fatalf("Handle failed: %v", err)
	}
	handler := servlet.ChainFilters(router, servlet.FilterFunc(func(ctx context.Context, req *servlet.Request, res servlet.Response, chain servlet.Chain) error {
		req.SetAttribute("traceID", "trace-1")
		req.SetAttribute("attempt", 2)
		current, err := session.NewMemoryManager().Create(ctx)
		if err != nil {
			return err
		}
		if err := current.SetAttribute("principal", "alice"); err != nil {
			return err
		}
		req.SetAttribute(session.AttributeCurrentSession, current)
		return chain.Next(ctx, req, res)
	}))

	recorder := httptest.NewRecorder()
	servletnethttp.Handler(handler).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/attributes", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"traceID":"trace-1"`) ||
		!strings.Contains(recorder.Body.String(), `"attempt":2`) ||
		!strings.Contains(recorder.Body.String(), `"principal":"alice"`) {
		t.Fatalf("body = %s, want attributes", recorder.Body.String())
	}
}

func TestParameterHelpersBindMatrixVariables(t *testing.T) {
	t.Parallel()

	router := arkweb.NewRouter()
	err := router.Handle(http.MethodGet, "/cars/{id}", mvc.JSON(http.StatusOK, func(ctx *arkweb.Context) (map[string]any, error) {
		id, err := mvc.PathInt64(ctx, "id")
		if err != nil {
			return nil, err
		}
		color, err := mvc.MatrixVariableString(ctx, "color")
		if err != nil {
			return nil, err
		}
		year, err := mvc.MatrixVariableInt(ctx, "year")
		if err != nil {
			return nil, err
		}
		return map[string]any{"id": id, "color": color, "year": year}, nil
	}))
	if err != nil {
		t.Fatalf("Handle failed: %v", err)
	}

	recorder := httptest.NewRecorder()
	servletnethttp.Handler(router).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/cars/42;color=red;year=2026", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"id":42`) ||
		!strings.Contains(recorder.Body.String(), `"color":"red"`) ||
		!strings.Contains(recorder.Body.String(), `"year":2026`) {
		t.Fatalf("body = %s, want matrix variables", recorder.Body.String())
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
