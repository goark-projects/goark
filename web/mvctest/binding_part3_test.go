package mvc_test

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	arkjson "goark.dev/arkarta/json"
	servletmultipart "goark.dev/arkarta/servlet/multipart"
	servletnethttp "goark.dev/arkarta/servlet/nethttp"
	arkweb "goark.dev/arkarta/web"

	"goark.dev/goark/container"
	"goark.dev/goark/core/convert"
	"goark.dev/goark/web"
	"goark.dev/goark/web/mvc"
)

func TestConversionServiceAppliesToModelAttribute(t *testing.T) {
	t.Parallel()

	type searchCriteria struct {
		Page        int         `form:"page"`
		Limit       *int        `form:"limit"`
		Tenant      tenantID    `form:"tenant"`
		Tags        []tenantID  `form:"tag"`
		PointerTags *[]tenantID `form:"pointerTag"`
	}
	type payload struct {
		Page        int      `json:"page"`
		Limit       int      `json:"limit"`
		Tenant      string   `json:"tenant"`
		Tags        []string `json:"tags"`
		PointerTags []string `json:"pointerTags"`
	}

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
	if err := registry.GET("/search", mvc.JSON(http.StatusOK, func(ctx *arkweb.Context) (payload,
		error) {
		criteria, err := mvc.ModelAttribute[searchCriteria](ctx)
		if err != nil {
			return payload{}, err
		}
		tags := make([]string, 0, len(criteria.Tags))
		for _, tag := range criteria.Tags {
			tags = append(tags, tag.value)
		}
		pointerTags := make([]string, 0, len(*criteria.PointerTags))
		for _, tag := range *criteria.PointerTags {
			pointerTags = append(pointerTags, tag.value)
		}
		return payload{
			Page:        criteria.Page,
			Limit:       *criteria.Limit,
			Tenant:      criteria.Tenant.value,
			Tags:        tags,
			PointerTags: pointerTags,
		}, nil
	})); err != nil {
		t.Fatalf("GET failed: %v", err)
	}
	router, err := registry.Router()
	if err != nil {
		t.Fatalf("Router failed: %v", err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet,
		"/search?page=goark&limit=zz&tenant=blue&tag=red,green&tag=gold&pointerTag=red,green&"+
			"pointerTag=gold", nil)
	servletnethttp.Handler(router).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", recorder.Code, recorder.Body.String())
	}
	var got payload
	if err := arkjson.Unmarshal(nil, recorder.Body.Bytes(), &got); err != nil {
		t.Fatalf("response json invalid: %v", err)
	}
	if got.Page != 105 ||
		got.Limit != 102 ||
		got.Tenant != "tenant:blue" ||
		!reflect.DeepEqual(got.Tags, []string{"tenant:red", "tenant:green", "tenant:gold"}) ||
		!reflect.DeepEqual(got.PointerTags, []string{"tenant:red", "tenant:green", "tenant:gold"}) {
		t.Fatalf("payload = %#v, want converted model attribute", got)
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
	err := router.Handle(http.MethodGet, "/aliases/{id}/{count}/{codes}/{ratio}", mvc.JSON(
		http.StatusOK, func(ctx *arkweb.Context) (payload, error) {
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
	servletnethttp.Handler(router).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet,
		"/aliases/u-42/3/7,8/1.5", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", recorder.Code, recorder.Body.String())
	}
	var got payload
	if err := arkjson.Unmarshal(nil, recorder.Body.Bytes(), &got); err != nil {
		t.Fatalf("response json invalid: %v", err)
	}
	if got.ID != "u-42" || got.Count != 3 || !reflect.DeepEqual(got.Codes, []int64{7, 8}) ||
		got.Ratio != 1.5 {
		t.Fatalf("payload = %#v", got)
	}
}

func TestParameterHelpersBindRequestHeaderMaps(t *testing.T) {
	t.Parallel()

	type payload struct {
		Headers map[string]string   `json:"headers"`
		Values  map[string][]string `json:"values"`
	}
	router := arkweb.NewRouter()
	err := router.Handle(http.MethodGet, "/headers", mvc.JSON(http.StatusOK, func(
		ctx *arkweb.Context) (payload, error) {
		headers, err := mvc.RequestHeaderMap(ctx)
		if err != nil {
			return payload{}, err
		}
		values, err := mvc.RequestHeaderValuesMap(ctx)
		if err != nil {
			return payload{}, err
		}
		return payload{Headers: headers, Values: values}, nil
	}))
	if err != nil {
		t.Fatalf("Handle failed: %v", err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/headers", nil)
	request.Header.Add("X-Role", "admin")
	request.Header.Add("X-Role", "ops")
	request.Header.Set("X-Request-ID", "req-1")
	servletnethttp.Handler(router).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", recorder.Code, recorder.Body.String())
	}
	var got payload
	if err := arkjson.Unmarshal(nil, recorder.Body.Bytes(), &got); err != nil {
		t.Fatalf("response json invalid: %v", err)
	}
	if got.Headers["X-Role"] != "admin" || got.Headers["X-Request-Id"] != "req-1" {
		t.Fatalf("headers = %#v", got.Headers)
	}
	if !reflect.DeepEqual(got.Values["X-Role"], []string{"admin", "ops"}) {
		t.Fatalf("header values = %#v", got.Values)
	}
}

func TestModelAttributeAdaptsEmptyArrayIndexFields(t *testing.T) {
	t.Parallel()

	router := arkweb.NewRouter()
	if err := router.Handle(http.MethodGet, "/tags", mvc.JSON(http.StatusOK, func(
		ctx *arkweb.Context) (binderTagPayload, error) {
		input, err := mvc.ModelAttribute[binderTagInput](ctx)
		if err != nil {
			return binderTagPayload{}, err
		}
		out := binderTagPayload{Tags: input.Tags}
		if input.Profile != nil {
			out.Aliases = input.Profile.Aliases
		}
		return out, nil
	})); err != nil {
		t.Fatalf("Handle failed: %v", err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet,
		"/tags?tags[]=red&tags[]=blue&profile.aliases[]=core&profile.aliases[]=web", nil)
	servletnethttp.Handler(router).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", recorder.Code, recorder.Body.String())
	}
	var got binderTagPayload
	if err := arkjson.Unmarshal(nil, recorder.Body.Bytes(), &got); err != nil {
		t.Fatalf("response json invalid: %v", err)
	}
	if len(got.Tags) != 2 ||
		got.Tags[0] != "red" ||
		got.Tags[1] != "blue" ||
		len(got.Aliases) != 2 ||
		got.Aliases[0] != "core" ||
		got.Aliases[1] != "web" {
		t.Fatalf("payload = %#v, want empty array index fields bound", got)
	}
}

func TestBindJSONResultGroupsUsesExplicitValidationGroup(t *testing.T) {
	t.Parallel()

	router := arkweb.NewRouter()
	if err := router.Handle(http.MethodPost, "/users", mvc.BindJSONResultGroups(http.StatusOK,
		func(_ *arkweb.Context, _ bindingCreateRequest, result mvc.BindingResult) (
			bindingResultPayload, error) {
			field, ok := result.FieldError("name")
			if !result.HasErrors() || !ok {
				return bindingResultPayload{Valid: true}, nil
			}
			return bindingResultPayload{Valid: result.Valid(), Field: field.Path(),
				Message: field.Message()}, nil
		},
		"create",
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
	if payload.Valid || payload.Field != "name" || payload.Message == "" {
		t.Fatalf("payload = %#v, want grouped binding result violation", payload)
	}
}

func TestParameterHelpersRejectMissingRequiredParameter(t *testing.T) {
	t.Parallel()

	router := arkweb.NewRouter()
	err := router.Handle(http.MethodGet, "/users", mvc.JSON(http.StatusOK, func(
		ctx *arkweb.Context) (map[string]string, error) {
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
	servletnethttp.Handler(router).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet,
		"/users", nil))
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400, body=%s", recorder.Code, recorder.Body.String())
	}
}

type binderUserPayload struct {
	Name         string            `json:"name"`
	Admin        bool              `json:"admin"`
	ProfileEmail string            `json:"profileEmail"`
	ProfileAdmin bool              `json:"profileAdmin"`
	Roles        []binderRole      `json:"roles"`
	Metadata     map[string]string `json:"metadata"`
}

type bindingCreateRequest struct {
	Name string `json:"name" arkarta:"required" arkarta-groups:"create"`
	Code string `json:"code" arkarta:"required"`
}

type uploadValidatedRequest struct {
	Title string                `form:"title" arkarta:"required"`
	File  servletmultipart.Part `multipart:"file"`
}
