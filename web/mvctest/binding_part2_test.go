package mvc_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	arkjson "goark.dev/arkarta/json"
	servletnethttp "goark.dev/arkarta/servlet/nethttp"
	arkweb "goark.dev/arkarta/web"

	"goark.dev/goark/container"
	"goark.dev/goark/core/convert"
	"goark.dev/goark/web"
	"goark.dev/goark/web/mvc"
)

func TestControllerInitBinderAppliesScopedConversionService(t *testing.T) {
	t.Parallel()

	type searchCriteria struct {
		Page   int            `form:"page"`
		Tenant scopedTenantID `form:"tenant"`
	}
	type payload struct {
		Page        int    `json:"page"`
		Tenant      string `json:"tenant"`
		ParamTenant string `json:"paramTenant"`
	}

	service, err := convert.NewService(convert.ConverterFunc[string, int](func(value string) (int,
		error) {
		return len(value) + 100, nil
	}))
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
	localController := mvc.NewRestController("local-search",
		mvc.GET("/local", mvc.JSON(http.StatusOK, func(ctx *arkweb.Context) (payload, error) {
			criteria, err := mvc.ModelAttribute[searchCriteria](ctx)
			if err != nil {
				return payload{}, err
			}
			tenant, err := mvc.RequestParamAs[scopedTenantID](ctx, "tenant")
			if err != nil {
				return payload{}, err
			}
			return payload{Page: criteria.Page, Tenant: criteria.Tenant.value,
				ParamTenant: tenant.value}, nil
		})),
	).WithInitBinders(mvc.BinderInitializerFunc(func(_ *arkweb.Context, binder *mvc.DataBinder) error {
		return binder.AddConverter(mvc.ConverterFunc[string, scopedTenantID](func(value string) (
			scopedTenantID, error) {
			return scopedTenantID{value: "local:" + value}, nil
		}))
	}))
	if err := localController.Register(registry); err != nil {
		t.Fatalf("local Register failed: %v", err)
	}
	plainController := mvc.NewRestController("plain-search",
		mvc.GET("/plain", mvc.JSON(http.StatusOK, func(ctx *arkweb.Context) (payload, error) {
			criteria, err := mvc.ModelAttribute[searchCriteria](ctx)
			if err != nil {
				return payload{}, err
			}
			return payload{Page: criteria.Page, Tenant: criteria.Tenant.value}, nil
		})),
	)
	if err := plainController.Register(registry); err != nil {
		t.Fatalf("plain Register failed: %v", err)
	}
	router, err := registry.Router()
	if err != nil {
		t.Fatalf("Router failed: %v", err)
	}

	local := httptest.NewRecorder()
	servletnethttp.Handler(router).ServeHTTP(local, httptest.NewRequest(http.MethodGet,
		"/local?page=goark&tenant=blue", nil))
	if local.Code != http.StatusOK {
		t.Fatalf("local status = %d, want 200, body=%s", local.Code, local.Body.String())
	}
	var got payload
	if err := arkjson.Unmarshal(nil, local.Body.Bytes(), &got); err != nil {
		t.Fatalf("local response json invalid: %v", err)
	}
	if got.Page != 105 || got.Tenant != "local:blue" || got.ParamTenant != "local:blue" {
		t.Fatalf("local payload = %#v, want scoped conversion", got)
	}

	plain := httptest.NewRecorder()
	servletnethttp.Handler(router).ServeHTTP(plain, httptest.NewRequest(http.MethodGet,
		"/plain?page=goark&tenant=blue", nil))
	if plain.Code != http.StatusBadRequest {
		t.Fatalf("plain status = %d, want 400, body=%s", plain.Code, plain.Body.String())
	}
}

func TestControllerInitBinderCustomizesFieldPrefixes(t *testing.T) {
	t.Parallel()

	registry := web.NewRegistry()
	controller := mvc.NewRestController("preferences",
		mvc.GET("/preferences", mvc.JSON(http.StatusOK, func(ctx *arkweb.Context) (
			binderPreferencePayload, error) {
			input, err := mvc.ModelAttribute[binderPreferenceInput](ctx)
			if err != nil {
				return binderPreferencePayload{}, err
			}
			return binderPreferencePayloadFromInput(input), nil
		})),
	).WithInitBinders(mvc.BinderInitializerFunc(func(_ *arkweb.Context, binder *mvc.DataBinder) error {
		if err := binder.SetFieldDefaultPrefix("~"); err != nil {
			return err
		}
		return binder.SetFieldMarkerPrefix("__")
	}))
	if err := controller.Register(registry); err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	router, err := registry.Router()
	if err != nil {
		t.Fatalf("Router failed: %v", err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet,
		"/preferences?~theme=dark&__notify=on&__profile.subscribed=on", nil)
	servletnethttp.Handler(router).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", recorder.Code, recorder.Body.String())
	}
	var got binderPreferencePayload
	if err := arkjson.Unmarshal(nil, recorder.Body.Bytes(), &got); err != nil {
		t.Fatalf("response json invalid: %v", err)
	}
	if got.Theme != "dark" ||
		!got.NotifySet ||
		got.Notify ||
		!got.ProfileSet ||
		got.Subscribed {
		t.Fatalf("payload = %#v, want custom field prefix binding", got)
	}
}

func TestParameterHelpersBindRequestParamMaps(t *testing.T) {
	t.Parallel()

	type payload struct {
		Params map[string]string   `json:"params"`
		Values map[string][]string `json:"values"`
	}
	router := arkweb.NewRouter()
	err := router.Handle(http.MethodPost, "/search", mvc.JSON(http.StatusOK, func(
		ctx *arkweb.Context) (payload, error) {
		params, err := mvc.RequestParamMap(ctx)
		if err != nil {
			return payload{}, err
		}
		values, err := mvc.RequestParamValuesMap(ctx)
		if err != nil {
			return payload{}, err
		}
		return payload{Params: params, Values: values}, nil
	}))
	if err != nil {
		t.Fatalf("Handle failed: %v", err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/search?tag=query&empty=", strings.NewReader(
		"tag=form&q=goark"))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	servletnethttp.Handler(router).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", recorder.Code, recorder.Body.String())
	}
	var got payload
	if err := arkjson.Unmarshal(nil, recorder.Body.Bytes(), &got); err != nil {
		t.Fatalf("response json invalid: %v", err)
	}
	if !reflect.DeepEqual(got.Params, map[string]string{"tag": "query", "empty": "", "q": "goark"}) {
		t.Fatalf("params = %#v", got.Params)
	}
	if !reflect.DeepEqual(got.Values, map[string][]string{"tag": {"query", "form"}, "empty": {""},
		"q": {"goark"}}) {
		t.Fatalf("values = %#v", got.Values)
	}
}

func TestModelAttributeAppliesDefaultFieldPrefixes(t *testing.T) {
	t.Parallel()

	router := arkweb.NewRouter()
	if err := router.Handle(http.MethodGet, "/preferences", mvc.JSON(http.StatusOK, func(
		ctx *arkweb.Context) (binderPreferencePayload, error) {
		input, err := mvc.ModelAttribute[binderPreferenceInput](ctx)
		if err != nil {
			return binderPreferencePayload{}, err
		}
		return binderPreferencePayloadFromInput(input), nil
	})); err != nil {
		t.Fatalf("Handle failed: %v", err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet,
		"/preferences?!theme=dark&_notify=on&confirm=true&_confirm=on&_profile.subscribed=on&_tags="+
			"on", nil)
	servletnethttp.Handler(router).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", recorder.Code, recorder.Body.String())
	}
	var got binderPreferencePayload
	if err := arkjson.Unmarshal(nil, recorder.Body.Bytes(), &got); err != nil {
		t.Fatalf("response json invalid: %v", err)
	}
	if got.Theme != "dark" ||
		!got.NotifySet ||
		got.Notify ||
		!got.ConfirmSet ||
		!got.Confirm ||
		!got.ProfileSet ||
		got.Subscribed ||
		got.TagsNil ||
		got.TagsLength != 0 {
		t.Fatalf("payload = %#v, want default field prefix binding", got)
	}
}

func TestModelAttributeRejectsOversizedMapProperties(t *testing.T) {
	t.Parallel()

	type userSearchCriteria struct {
		Filters map[string]string `form:"filters" json:"filters"`
	}

	router := arkweb.NewRouter()
	err := router.Handle(http.MethodGet, "/users/search", mvc.JSON(http.StatusOK, func(
		ctx *arkweb.Context) (map[string]int, error) {
		criteria, err := mvc.ModelAttribute[userSearchCriteria](ctx)
		if err != nil {
			return nil, err
		}
		return map[string]int{"filters": len(criteria.Filters)}, nil
	}))
	if err != nil {
		t.Fatalf("Handle failed: %v", err)
	}

	var query strings.Builder
	query.WriteString("/users/search?")
	for i := 0; i < 257; i++ {
		if i > 0 {
			query.WriteByte('&')
		}
		fmt.Fprintf(&query, "filters[k%d]=v", i)
	}
	recorder := httptest.NewRecorder()
	servletnethttp.Handler(router).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet,
		query.String(), nil))

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400, body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestCookieValueStringAllowsMissingOptionalCookie(t *testing.T) {
	t.Parallel()

	router := arkweb.NewRouter()
	if err := router.Handle(http.MethodGet, "/optional-cookie", mvc.JSON(http.StatusOK, func(
		ctx *arkweb.Context) (map[string]string, error) {
		theme, err := mvc.CookieValueString(ctx, "theme", mvc.WithRequired(false))
		if err != nil {
			return nil, err
		}
		return map[string]string{"theme": theme}, nil
	})); err != nil {
		t.Fatalf("GET failed: %v", err)
	}

	recorder := httptest.NewRecorder()
	servletnethttp.Handler(router).ServeHTTP(
		recorder,
		httptest.NewRequest(http.MethodGet, "/optional-cookie", nil),
	)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", recorder.Code, recorder.Body.String())
	}
}

type binderUserInput struct {
	Name     string            `form:"name" json:"name"`
	Admin    bool              `form:"admin" json:"admin"`
	Profile  *binderProfile    `form:"profile" json:"profile"`
	Roles    []binderRole      `form:"roles" json:"roles"`
	Metadata map[string]string `form:"metadata" json:"metadata"`
}

type uploadBindingPayload struct {
	Valid    bool   `json:"valid"`
	Field    string `json:"field"`
	Filename string `json:"filename"`
}

type tenantID struct {
	value string
}
