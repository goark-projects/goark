package mvc_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	arkjson "goark.dev/arkarta/json"
	servletnethttp "goark.dev/arkarta/servlet/nethttp"
	arkweb "goark.dev/arkarta/web"

	"goark.dev/arkarta/servlet"
	"goark.dev/arkarta/servlet/session"
	"goark.dev/goark/container"
	"goark.dev/goark/core/convert"
	"goark.dev/goark/web"
	"goark.dev/goark/web/mvc"
)

func TestConversionServiceAppliesToMVCParameters(t *testing.T) {
	t.Parallel()

	service, err := convert.NewService(
		convert.ConverterFunc[string, int](func(value string) (int, error) {
			return len(value) + 100, nil
		}),
		convert.ConverterFunc[string, tenantID](func(value string) (tenantID, error) {
			return tenantID{value: "tenant:" + value}, nil
		}),
	)
	if err != nil {
		t.Fatalf("NewService failed: %v", err)
	}
	beanRegistry := container.NewRegistry()
	if err := mvc.RegisterConversionService(beanRegistry, "testConversionService",
		service); err != nil {
		t.Fatalf("RegisterConversionService failed: %v", err)
	}
	resolver, err := container.New(beanRegistry)
	if err != nil {
		t.Fatalf("container.New failed: %v", err)
	}

	registry := web.NewRegistry()
	if err := web.ApplyConfigurers(t.Context(), resolver, registry); err != nil {
		t.Fatalf("ApplyConfigurers failed: %v", err)
	}
	if err := registry.GET("/items", mvc.JSON(http.StatusOK, func(ctx *arkweb.Context) (
		map[string]any, error) {
		page, err := mvc.RequestParamInt(ctx, "page")
		if err != nil {
			return nil, err
		}
		tenant, err := mvc.RequestParamAs[tenantID](ctx, "tenant")
		if err != nil {
			return nil, err
		}
		return map[string]any{
			"page":   page,
			"tenant": tenant.value,
		}, nil
	})); err != nil {
		t.Fatalf("GET failed: %v", err)
	}
	router, err := registry.Router()
	if err != nil {
		t.Fatalf("Router failed: %v", err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/items?page=abc&tenant=blue", nil)
	servletnethttp.Handler(router).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"page":103`) ||
		!strings.Contains(recorder.Body.String(), `"tenant":"tenant:blue"`) {
		t.Fatalf("body = %s, want converted parameters", recorder.Body.String())
	}
}

func TestParameterHelpersBindRequestAndSessionAttributes(t *testing.T) {
	t.Parallel()

	router := arkweb.NewRouter()
	err := router.Handle(http.MethodGet, "/attributes", mvc.JSON(http.StatusOK, func(
		ctx *arkweb.Context) (map[string]any, error) {
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
	handler := servlet.ChainFilters(router, servlet.FilterFunc(func(ctx context.Context,
		req *servlet.Request, res servlet.Response, chain servlet.Chain) error {
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
	servletnethttp.Handler(handler).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet,
		"/attributes", nil))
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
	err := router.Handle(http.MethodGet, "/cars/{id}", mvc.JSON(http.StatusOK, func(
		ctx *arkweb.Context) (map[string]any, error) {
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
	servletnethttp.Handler(router).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet,
		"/cars/42;color=red;year=2026;code=1,2;flag=true,false", nil))
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
	err := router.Handle(http.MethodGet, "/users/search", mvc.JSON(http.StatusOK, func(
		ctx *arkweb.Context) (map[string]any, error) {
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
	request := httptest.NewRequest(http.MethodGet,
		"/users/search?owner.name=ada&owner.age=37&page=2", nil)
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

func TestBindJSONResultPassesValidationErrorsToHandler(t *testing.T) {
	t.Parallel()

	router := arkweb.NewRouter()
	if err := router.Handle(http.MethodPost, "/users", mvc.BindJSONResult(http.StatusOK,
		func(_ *arkweb.Context, input bindingCreateRequest, result mvc.BindingResult) (
			bindingResultPayload, error) {
			field, ok := result.FieldError("code")
			if !result.HasErrors() || !ok {
				return bindingResultPayload{Valid: true}, nil
			}
			return bindingResultPayload{
				Valid:   result.Valid(),
				Field:   field.Path(),
				Message: field.Message(),
				Name:    input.Name,
			}, nil
		},
	)); err != nil {
		t.Fatalf("Handle failed: %v", err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(`{}`))
	request.Header.Set("Content-Type", arkjson.ContentType)
	servletnethttp.Handler(router).ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", recorder.Code, recorder.Body.String())
	}
	var payload bindingResultPayload
	if err := arkjson.Unmarshal(nil, recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("response json invalid: %v", err)
	}
	if payload.Valid || payload.Field != "code" || payload.Message == "" || payload.Name != "" {
		t.Fatalf("payload = %#v, want binding result violation", payload)
	}
}

func TestParameterHelpersBindPathVariableMap(t *testing.T) {
	t.Parallel()

	type payload struct {
		Path map[string]string `json:"path"`
	}
	router := arkweb.NewRouter()
	err := router.Handle(http.MethodGet, "/teams/{teamId}/users/{userId}", mvc.JSON(http.StatusOK,
		func(ctx *arkweb.Context) (payload, error) {
			path, err := mvc.PathVariableMap(ctx)
			if err != nil {
				return payload{}, err
			}
			return payload{Path: path}, nil
		}))
	if err != nil {
		t.Fatalf("Handle failed: %v", err)
	}

	recorder := httptest.NewRecorder()
	servletnethttp.Handler(router).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet,
		"/teams/core;ignored=true/users/42;role=admin", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", recorder.Code, recorder.Body.String())
	}
	var got payload
	if err := arkjson.Unmarshal(nil, recorder.Body.Bytes(), &got); err != nil {
		t.Fatalf("response json invalid: %v", err)
	}
	if !reflect.DeepEqual(got.Path, map[string]string{"teamId": "core", "userId": "42"}) {
		t.Fatalf("path = %#v", got.Path)
	}
}

type binderPreferencePayload struct {
	Theme      string `json:"theme"`
	NotifySet  bool   `json:"notifySet"`
	Notify     bool   `json:"notify"`
	ConfirmSet bool   `json:"confirmSet"`
	Confirm    bool   `json:"confirm"`
	ProfileSet bool   `json:"profileSet"`
	Subscribed bool   `json:"subscribed"`
	TagsNil    bool   `json:"tagsNil"`
	TagsLength int    `json:"tagsLength"`
}

type binderPreferenceInput struct {
	Theme   string                   `form:"theme" json:"theme"`
	Notify  *bool                    `form:"notify" json:"notify"`
	Confirm *bool                    `form:"confirm" json:"confirm"`
	Profile *binderPreferenceProfile `form:"profile" json:"profile"`
	Tags    []string                 `form:"tags" json:"tags"`
}

type binderRole struct {
	Name  string `form:"name" json:"name"`
	Admin bool   `form:"admin" json:"admin"`
}

type adviceTenantID struct {
	value string
}
