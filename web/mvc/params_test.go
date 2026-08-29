package mvc_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	arkjson "goark.dev/arkarta/json"
	"goark.dev/arkarta/servlet"
	servletnethttp "goark.dev/arkarta/servlet/nethttp"
	"goark.dev/arkarta/servlet/session"
	arkweb "goark.dev/arkarta/web"
	"goark.dev/goark/web/filter"
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

func TestParameterHelpersBindExtendedConversions(t *testing.T) {
	t.Parallel()

	type payload struct {
		Score     float64   `json:"score"`
		PathIDs   []int64   `json:"pathIds"`
		Tags      []string  `json:"tags"`
		IDs       []int64   `json:"ids"`
		Enabled   []bool    `json:"enabled"`
		Ratios    []float64 `json:"ratios"`
		HeaderAt  string    `json:"headerAt"`
		Roles     []string  `json:"roles"`
		Threshold float64   `json:"threshold"`
		Day       string    `json:"day"`
	}

	router := arkweb.NewRouter()
	err := router.Handle(http.MethodGet, "/reports/{date}/{pathIds}", mvc.JSON(http.StatusOK, func(ctx *arkweb.Context) (payload, error) {
		score, err := mvc.RequestParamFloat64(ctx, "score")
		if err != nil {
			return payload{}, err
		}
		pathIDs, err := mvc.PathInt64s(ctx, "pathIds")
		if err != nil {
			return payload{}, err
		}
		tags, err := mvc.RequestParamStrings(ctx, "tag")
		if err != nil {
			return payload{}, err
		}
		ids, err := mvc.RequestParamInt64s(ctx, "ids")
		if err != nil {
			return payload{}, err
		}
		enabled, err := mvc.RequestParamBools(ctx, "enabled")
		if err != nil {
			return payload{}, err
		}
		ratios, err := mvc.RequestParamFloat64s(ctx, "ratio")
		if err != nil {
			return payload{}, err
		}
		headerAt, err := mvc.RequestHeaderTime(ctx, "X-At")
		if err != nil {
			return payload{}, err
		}
		roles, err := mvc.RequestHeaderStrings(ctx, "X-Role")
		if err != nil {
			return payload{}, err
		}
		threshold, err := mvc.CookieValueFloat64(ctx, "threshold")
		if err != nil {
			return payload{}, err
		}
		day, err := mvc.PathTime(ctx, "date", mvc.WithTimeLayout("20060102"))
		if err != nil {
			return payload{}, err
		}
		return payload{
			Score:     score,
			PathIDs:   pathIDs,
			Tags:      tags,
			IDs:       ids,
			Enabled:   enabled,
			Ratios:    ratios,
			HeaderAt:  headerAt.UTC().Format(time.RFC3339),
			Roles:     roles,
			Threshold: threshold,
			Day:       day.Format("2006-01-02"),
		}, nil
	}))
	if err != nil {
		t.Fatalf("Handle failed: %v", err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/reports/20260829/7,8?score=98.5&tag=ops&tag=web,api&ids=1,2&enabled=true,false&ratio=1.5&ratio=2.5", nil)
	request.Header.Set("X-At", "2026-08-29T02:30:00Z")
	request.Header.Add("X-Role", "admin,ops")
	request.AddCookie(&http.Cookie{Name: "threshold", Value: "0.75"})
	servletnethttp.Handler(router).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", recorder.Code, recorder.Body.String())
	}
	var got payload
	if err := arkjson.Unmarshal(nil, recorder.Body.Bytes(), &got); err != nil {
		t.Fatalf("response json invalid: %v", err)
	}
	if got.Score != 98.5 ||
		!reflect.DeepEqual(got.PathIDs, []int64{7, 8}) ||
		!reflect.DeepEqual(got.Tags, []string{"ops", "web", "api"}) ||
		!reflect.DeepEqual(got.IDs, []int64{1, 2}) ||
		!reflect.DeepEqual(got.Enabled, []bool{true, false}) ||
		!reflect.DeepEqual(got.Ratios, []float64{1.5, 2.5}) ||
		got.HeaderAt != "2026-08-29T02:30:00Z" ||
		!reflect.DeepEqual(got.Roles, []string{"admin", "ops"}) ||
		got.Threshold != 0.75 ||
		got.Day != "2026-08-29" {
		t.Fatalf("payload = %#v", got)
	}
}

func TestParameterHelpersBindPathVariableAliases(t *testing.T) {
	t.Parallel()

	type payload struct {
		ID    string  `json:"id"`
		Count int     `json:"count"`
		Codes []int64 `json:"codes"`
		Ratio float64 `json:"ratio"`
	}

	router := arkweb.NewRouter()
	err := router.Handle(http.MethodGet, "/aliases/{id}/{count}/{codes}/{ratio}", mvc.JSON(http.StatusOK, func(ctx *arkweb.Context) (payload, error) {
		id, err := mvc.PathVariableString(ctx, "id")
		if err != nil {
			return payload{}, err
		}
		count, err := mvc.PathVariableAs[int](ctx, "count")
		if err != nil {
			return payload{}, err
		}
		codes, err := mvc.PathVariableInt64s(ctx, "codes")
		if err != nil {
			return payload{}, err
		}
		ratio, err := mvc.PathVariableFloat64(ctx, "ratio")
		if err != nil {
			return payload{}, err
		}
		return payload{ID: id, Count: count, Codes: codes, Ratio: ratio}, nil
	}))
	if err != nil {
		t.Fatalf("Handle failed: %v", err)
	}

	recorder := httptest.NewRecorder()
	servletnethttp.Handler(router).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/aliases/u-42/3/7,8/1.5", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", recorder.Code, recorder.Body.String())
	}
	var got payload
	if err := arkjson.Unmarshal(nil, recorder.Body.Bytes(), &got); err != nil {
		t.Fatalf("response json invalid: %v", err)
	}
	if got.ID != "u-42" || got.Count != 3 || !reflect.DeepEqual(got.Codes, []int64{7, 8}) || got.Ratio != 1.5 {
		t.Fatalf("payload = %#v", got)
	}
}

func TestRequestParamReadsFormContentFilterValues(t *testing.T) {
	t.Parallel()

	type payload struct {
		Active bool     `json:"active"`
		Tags   []string `json:"tags"`
	}
	router := arkweb.NewRouter()
	err := router.Handle(http.MethodDelete, "/users/42", mvc.JSON(http.StatusOK, func(ctx *arkweb.Context) (payload, error) {
		tags, err := mvc.RequestParamStrings(ctx, "tag")
		if err != nil {
			return payload{}, err
		}
		active, err := mvc.RequestParamBool(ctx, "active")
		if err != nil {
			return payload{}, err
		}
		return payload{Active: active, Tags: tags}, nil
	}))
	if err != nil {
		t.Fatalf("Handle failed: %v", err)
	}

	request := httptest.NewRequest(http.MethodDelete, "/users/42?tag=query", strings.NewReader("tag=form&active=true"))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	recorder := httptest.NewRecorder()
	servletnethttp.Handler(servlet.ChainFilters(router, filter.FormContent())).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", recorder.Code, recorder.Body.String())
	}
	var got payload
	if err := arkjson.Unmarshal(nil, recorder.Body.Bytes(), &got); err != nil {
		t.Fatalf("response json invalid: %v", err)
	}
	if !reflect.DeepEqual(got.Tags, []string{"query", "form"}) || !got.Active {
		t.Fatalf("payload = %#v", got)
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
		codes, err := mvc.MatrixVariableInt64s(ctx, "code")
		if err != nil {
			return nil, err
		}
		flags, err := mvc.MatrixVariableBools(ctx, "flag")
		if err != nil {
			return nil, err
		}
		return map[string]any{"id": id, "color": color, "year": year, "codes": codes, "flags": flags}, nil
	}))
	if err != nil {
		t.Fatalf("Handle failed: %v", err)
	}

	recorder := httptest.NewRecorder()
	servletnethttp.Handler(router).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/cars/42;color=red;year=2026;code=1,2;flag=true,false", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"id":42`) ||
		!strings.Contains(recorder.Body.String(), `"color":"red"`) ||
		!strings.Contains(recorder.Body.String(), `"year":2026`) ||
		!strings.Contains(recorder.Body.String(), `"codes":[1,2]`) ||
		!strings.Contains(recorder.Body.String(), `"flags":[true,false]`) {
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

func TestModelAttributeBindsNestedProperties(t *testing.T) {
	t.Parallel()

	type owner struct {
		Name string `form:"name" json:"name"`
		Age  int    `form:"age" json:"age"`
	}
	type userSearchCriteria struct {
		Owner *owner `form:"owner" json:"owner"`
		Page  int    `form:"page" json:"page"`
	}

	router := arkweb.NewRouter()
	err := router.Handle(http.MethodGet, "/users/search", mvc.JSON(http.StatusOK, func(ctx *arkweb.Context) (map[string]any, error) {
		criteria, err := mvc.ModelAttribute[userSearchCriteria](ctx)
		if err != nil {
			return nil, err
		}
		return map[string]any{
			"ownerName": criteria.Owner.Name,
			"ownerAge":  criteria.Owner.Age,
			"page":      criteria.Page,
		}, nil
	}))
	if err != nil {
		t.Fatalf("Handle failed: %v", err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/users/search?owner.name=ada&owner.age=37&page=2", nil)
	servletnethttp.Handler(router).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"ownerName":"ada"`) ||
		!strings.Contains(recorder.Body.String(), `"ownerAge":37`) ||
		!strings.Contains(recorder.Body.String(), `"page":2`) {
		t.Fatalf("body = %s, want nested model attribute values", recorder.Body.String())
	}
}

func TestModelAttributeReadsFormContentFilterValues(t *testing.T) {
	t.Parallel()

	type userSearchCriteria struct {
		Username string `form:"username" json:"username" arkarta:"required,min=2"`
		Page     int    `form:"page" json:"page"`
	}
	router := arkweb.NewRouter()
	err := router.Handle(http.MethodDelete, "/users/search", mvc.JSON(http.StatusOK, func(ctx *arkweb.Context) (map[string]any, error) {
		criteria, err := mvc.ModelAttribute[userSearchCriteria](ctx)
		if err != nil {
			return nil, err
		}
		return map[string]any{
			"username": criteria.Username,
			"page":     criteria.Page,
		}, nil
	}))
	if err != nil {
		t.Fatalf("Handle failed: %v", err)
	}

	request := httptest.NewRequest(http.MethodDelete, "/users/search?username=ad", strings.NewReader("page=2"))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	recorder := httptest.NewRecorder()
	servletnethttp.Handler(servlet.ChainFilters(router, filter.FormContent())).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"username":"ad"`) ||
		!strings.Contains(recorder.Body.String(), `"page":2`) {
		t.Fatalf("body = %s, want model attribute values", recorder.Body.String())
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
