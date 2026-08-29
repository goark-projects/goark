package mvc_test

import (
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

type tenantID struct {
	value string
}

type scopedTenantID struct {
	value string
}

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
	if err := mvc.RegisterConversionService(beanRegistry, "testConversionService", service); err != nil {
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
	if err := registry.GET("/items", mvc.JSON(http.StatusOK, func(ctx *arkweb.Context) (map[string]any, error) {
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

	service, err := convert.NewService(convert.ConverterFunc[string, int](func(value string) (int, error) {
		return len(value) + 100, nil
	}))
	if err != nil {
		t.Fatalf("NewService failed: %v", err)
	}
	beanRegistry := container.NewRegistry()
	if err := mvc.RegisterConversionService(beanRegistry, "testConversionService", service); err != nil {
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
			return payload{Page: criteria.Page, Tenant: criteria.Tenant.value, ParamTenant: tenant.value}, nil
		})),
	).WithInitBinders(mvc.BinderInitializerFunc(func(_ *arkweb.Context, binder *mvc.DataBinder) error {
		return binder.AddConverter(mvc.ConverterFunc[string, scopedTenantID](func(value string) (scopedTenantID, error) {
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
	servletnethttp.Handler(router).ServeHTTP(local, httptest.NewRequest(http.MethodGet, "/local?page=goark&tenant=blue", nil))
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
	servletnethttp.Handler(router).ServeHTTP(plain, httptest.NewRequest(http.MethodGet, "/plain?page=goark&tenant=blue", nil))
	if plain.Code != http.StatusBadRequest {
		t.Fatalf("plain status = %d, want 400, body=%s", plain.Code, plain.Body.String())
	}
}

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
	if err := mvc.RegisterConversionService(beanRegistry, "testConversionService", service); err != nil {
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
	if err := registry.GET("/search", mvc.JSON(http.StatusOK, func(ctx *arkweb.Context) (payload, error) {
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
	request := httptest.NewRequest(http.MethodGet, "/search?page=goark&limit=zz&tenant=blue&tag=red,green&tag=gold&pointerTag=red,green&pointerTag=gold", nil)
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
