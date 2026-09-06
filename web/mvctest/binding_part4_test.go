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

	"goark.dev/arkarta/servlet"
	"goark.dev/goark/container"
	"goark.dev/goark/core/convert"
	"goark.dev/goark/web"
	"goark.dev/goark/web/filter"
	"goark.dev/goark/web/mvc"
)

func TestControllerAdviceInitBinderAppliesScopedConversionService(t *testing.T) {
	t.Parallel()

	type searchCriteria struct {
		Page   int            `form:"page"`
		Tenant adviceTenantID `form:"tenant"`
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
	advice := mvc.NewRestControllerAdvice("global-binders").WithInitBinders(
		mvc.BinderInitializerFunc(func(_ *arkweb.Context, binder *mvc.DataBinder) error {
			return binder.AddConverter(mvc.ConverterFunc[string, adviceTenantID](func(value string) (
				adviceTenantID, error) {
				return adviceTenantID{value: "advice:" + value}, nil
			}))
		}),
	)
	configurer := mvc.NewConfigurer(mvc.NewRestController("search",
		mvc.GET("/advice", mvc.JSON(http.StatusOK, func(ctx *arkweb.Context) (payload, error) {
			criteria, err := mvc.ModelAttribute[searchCriteria](ctx)
			if err != nil {
				return payload{}, err
			}
			tenant, err := mvc.RequestParamAs[adviceTenantID](ctx, "tenant")
			if err != nil {
				return payload{}, err
			}
			return payload{Page: criteria.Page, Tenant: criteria.Tenant.value,
				ParamTenant: tenant.value}, nil
		})),
	)).WithControllerAdvices(advice)
	if err := configurer.ConfigureWeb(t.Context(), registry); err != nil {
		t.Fatalf("ConfigureWeb failed: %v", err)
	}
	router, err := registry.Router()
	if err != nil {
		t.Fatalf("Router failed: %v", err)
	}

	recorder := httptest.NewRecorder()
	servletnethttp.Handler(router).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet,
		"/advice?page=goark&tenant=blue", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", recorder.Code, recorder.Body.String())
	}
	var got payload
	if err := arkjson.Unmarshal(nil, recorder.Body.Bytes(), &got); err != nil {
		t.Fatalf("response json invalid: %v", err)
	}
	if got.Page != 105 || got.Tenant != "advice:blue" || got.ParamTenant != "advice:blue" {
		t.Fatalf("payload = %#v, want advice-scoped conversion", got)
	}
	if _, err := convert.Convert[adviceTenantID](service, "blue"); err == nil {
		t.Fatal("global conversion service should not see advice converter")
	}
}

func TestControllerInitBinderRestrictsModelAttributeAllowedFields(t *testing.T) {
	t.Parallel()

	registry := web.NewRegistry()
	controller := mvc.NewRestController("users",
		mvc.GET("/users/search", mvc.JSON(http.StatusOK, func(ctx *arkweb.Context) (
			binderUserPayload, error) {
			input, err := mvc.ModelAttribute[binderUserInput](ctx)
			if err != nil {
				return binderUserPayload{}, err
			}
			return binderPayload(input), nil
		})),
	).WithInitBinders(mvc.BinderInitializerFunc(func(_ *arkweb.Context, binder *mvc.DataBinder) error {
		return binder.SetAllowedFields("name", "profile.email", "roles[*].name", "metadata[department]")
	}))
	if err := controller.Register(registry); err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	router, err := registry.Router()
	if err != nil {
		t.Fatalf("Router failed: %v", err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/users/search?"+
		"name=ada&admin=true&profile.email=ada@example.test&profile.admin=true&"+
		"roles[0].name=reader&roles[0].admin=true&metadata[department]=engineering&metadata[secret]="+
		"root", nil)
	servletnethttp.Handler(router).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", recorder.Code, recorder.Body.String())
	}
	var got binderUserPayload
	if err := arkjson.Unmarshal(nil, recorder.Body.Bytes(), &got); err != nil {
		t.Fatalf("response json invalid: %v", err)
	}
	if got.Name != "ada" || got.Admin || got.ProfileEmail != "ada@example.test" || got.ProfileAdmin {
		t.Fatalf("payload = %#v, want only allowed scalar and nested values", got)
	}
	if len(got.Roles) != 1 || got.Roles[0].Name != "reader" || got.Roles[0].Admin {
		t.Fatalf("roles = %#v, want only allowed indexed role name", got.Roles)
	}
	if got.Metadata["department"] != "engineering" {
		t.Fatalf("metadata = %#v, want allowed department", got.Metadata)
	}
	if _, ok := got.Metadata["secret"]; ok {
		t.Fatalf("metadata = %#v, want secret skipped", got.Metadata)
	}
}

func TestModelAttributeBindsMapProperties(t *testing.T) {
	t.Parallel()

	type userSearchCriteria struct {
		Flags   map[string]bool     `form:"flags" json:"flags"`
		Filters map[string]int      `form:"filters" json:"filters"`
		Tags    map[string][]string `form:"tags" json:"tags"`
		Page    int                 `form:"page" json:"page"`
	}

	router := arkweb.NewRouter()
	err := router.Handle(http.MethodGet, "/users/search", mvc.JSON(http.StatusOK, func(
		ctx *arkweb.Context) (map[string]any, error) {
		criteria, err := mvc.ModelAttribute[userSearchCriteria](ctx)
		if err != nil {
			return nil, err
		}
		return map[string]any{
			"enabled": criteria.Flags["enabled"],
			"level":   criteria.Filters["level"],
			"roles":   criteria.Tags["roles"],
			"groups":  criteria.Tags["groups"],
			"page":    criteria.Page,
		}, nil
	}))
	if err != nil {
		t.Fatalf("Handle failed: %v", err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/users/search?"+
		"flags[enabled]=true&filters[level]=7&tags[roles]=admin,ops&"+
		"tags[groups]=core&tags[groups]=web&page=2", nil)
	servletnethttp.Handler(router).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"enabled":true`) ||
		!strings.Contains(recorder.Body.String(), `"level":7`) ||
		!strings.Contains(recorder.Body.String(), `"roles":["admin","ops"]`) ||
		!strings.Contains(recorder.Body.String(), `"groups":["core","web"]`) ||
		!strings.Contains(recorder.Body.String(), `"page":2`) {
		t.Fatalf("body = %s, want map model attribute values", recorder.Body.String())
	}
}

func TestRequestParamReadsFormContentFilterValues(t *testing.T) {
	t.Parallel()

	type payload struct {
		Active bool     `json:"active"`
		Tags   []string `json:"tags"`
	}
	router := arkweb.NewRouter()
	err := router.Handle(http.MethodDelete, "/users/42", mvc.JSON(http.StatusOK, func(
		ctx *arkweb.Context) (payload, error) {
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

	request := httptest.NewRequest(http.MethodDelete, "/users/42?tag=query", strings.NewReader(
		"tag=form&active=true"))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	recorder := httptest.NewRecorder()
	servletnethttp.Handler(servlet.ChainFilters(router, filter.FormContent())).ServeHTTP(recorder,
		request)

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

func TestMatrixVariableCanBindFromNamedPathVariable(t *testing.T) {
	t.Parallel()

	router := arkweb.NewRouter()
	err := router.Handle(http.MethodGet, "/owners/{ownerId}/pets/{petId}", mvc.JSON(http.StatusOK,
		func(ctx *arkweb.Context) (map[string]any, error) {
			ownerQ, err := mvc.MatrixVariableString(ctx, "q", mvc.WithMatrixPathVariable("ownerId"))
			if err != nil {
				return nil, err
			}
			petQ, err := mvc.MatrixVariableString(ctx, "q", mvc.WithMatrixPathVariable("petId"))
			if err != nil {
				return nil, err
			}
			color, err := mvc.MatrixVariableString(ctx, "color", mvc.WithMatrixPathVariable("petId"))
			if err != nil {
				return nil, err
			}
			return map[string]any{"owner": ownerQ, "pet": petQ, "color": color}, nil
		}))
	if err != nil {
		t.Fatalf("Handle failed: %v", err)
	}

	recorder := httptest.NewRecorder()
	servletnethttp.Handler(router).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet,
		"/owners/42;q=owner/pets/21;q=pet;color=black", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"owner":"owner"`) ||
		!strings.Contains(recorder.Body.String(), `"pet":"pet"`) ||
		!strings.Contains(recorder.Body.String(), `"color":"black"`) {
		t.Fatalf("body = %s, want path-scoped matrix variables", recorder.Body.String())
	}
}

func TestModelAttributeMapsValidationErrors(t *testing.T) {
	t.Parallel()

	type userSearchCriteria struct {
		Username string `form:"username" json:"username" arkarta:"required"`
	}
	router := arkweb.NewRouter()
	err := router.Handle(http.MethodGet, "/users/search", mvc.JSON(http.StatusOK, func(
		ctx *arkweb.Context) (map[string]string, error) {
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
	servletnethttp.Handler(router).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet,
		"/users/search", nil))
	if recorder.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422, body=%s", recorder.Code, recorder.Body.String())
	}
}

type bindingResultPayload struct {
	Valid        bool   `json:"valid"`
	BindingError bool   `json:"bindingError"`
	Field        string `json:"field"`
	Message      string `json:"message"`
	Name         string `json:"name"`
	Page         int    `json:"page"`
}

type binderSuppressedPayload struct {
	Name             string   `json:"name"`
	Admin            bool     `json:"admin"`
	SuppressedFields []string `json:"suppressedFields"`
}

type binderTagPayload struct {
	Tags    []string `json:"tags"`
	Aliases []string `json:"aliases"`
}

type binderTagProfile struct {
	Aliases []string `form:"aliases" json:"aliases"`
}
