package mvc_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	arkjson "goark.dev/arkarta/json"
	servletnethttp "goark.dev/arkarta/servlet/nethttp"
	arkweb "goark.dev/arkarta/web"
	"goark.dev/goark/web"
	"goark.dev/goark/web/mvc"
)

type binderProfile struct {
	Email string `form:"email" json:"email"`
	Admin bool   `form:"admin" json:"admin"`
}

type binderRole struct {
	Name  string `form:"name" json:"name"`
	Admin bool   `form:"admin" json:"admin"`
}

type binderUserInput struct {
	Name     string            `form:"name" json:"name"`
	Admin    bool              `form:"admin" json:"admin"`
	Profile  *binderProfile    `form:"profile" json:"profile"`
	Roles    []binderRole      `form:"roles" json:"roles"`
	Metadata map[string]string `form:"metadata" json:"metadata"`
}

type binderUserPayload struct {
	Name         string            `json:"name"`
	Admin        bool              `json:"admin"`
	ProfileEmail string            `json:"profileEmail"`
	ProfileAdmin bool              `json:"profileAdmin"`
	Roles        []binderRole      `json:"roles"`
	Metadata     map[string]string `json:"metadata"`
}

func TestControllerInitBinderRestrictsModelAttributeAllowedFields(t *testing.T) {
	t.Parallel()

	registry := web.NewRegistry()
	controller := mvc.NewRestController("users",
		mvc.GET("/users/search", mvc.JSON(http.StatusOK, func(ctx *arkweb.Context) (binderUserPayload, error) {
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
		"roles[0].name=reader&roles[0].admin=true&metadata[department]=engineering&metadata[secret]=root", nil)
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

func TestControllerAdviceInitBinderSkipsDisallowedModelAttributeFields(t *testing.T) {
	t.Parallel()

	registry := web.NewRegistry()
	controller := mvc.NewRestController("users",
		mvc.GET("/users/search", mvc.JSON(http.StatusOK, func(ctx *arkweb.Context) (binderUserPayload, error) {
			input, err := mvc.ModelAttribute[binderUserInput](ctx)
			if err != nil {
				return binderUserPayload{}, err
			}
			return binderPayload(input), nil
		})),
	)
	advice := mvc.NewRestControllerAdvice("global-binder").WithInitBinders(
		mvc.BinderInitializerFunc(func(_ *arkweb.Context, binder *mvc.DataBinder) error {
			return binder.SetDisallowedFields("ADMIN", "PROFILE.ADMIN", "ROLES[*].ADMIN", "METADATA[SECRET]")
		}),
	)
	if err := mvc.NewConfigurer(controller).WithControllerAdvices(advice).ConfigureWeb(t.Context(), registry); err != nil {
		t.Fatalf("ConfigureWeb failed: %v", err)
	}
	router, err := registry.Router()
	if err != nil {
		t.Fatalf("Router failed: %v", err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/users/search?"+
		"name=ada&admin=true&profile.email=ada@example.test&profile.admin=true&"+
		"roles[0].name=reader&roles[0].admin=true&metadata[department]=engineering&metadata[secret]=root", nil)
	servletnethttp.Handler(router).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", recorder.Code, recorder.Body.String())
	}
	var got binderUserPayload
	if err := arkjson.Unmarshal(nil, recorder.Body.Bytes(), &got); err != nil {
		t.Fatalf("response json invalid: %v", err)
	}
	if got.Name != "ada" || got.Admin || got.ProfileEmail != "ada@example.test" || got.ProfileAdmin {
		t.Fatalf("payload = %#v, want disallowed scalar and nested values skipped", got)
	}
	if len(got.Roles) != 1 || got.Roles[0].Name != "reader" || got.Roles[0].Admin {
		t.Fatalf("roles = %#v, want disallowed indexed role admin skipped", got.Roles)
	}
	if got.Metadata["department"] != "engineering" {
		t.Fatalf("metadata = %#v, want allowed department", got.Metadata)
	}
	if _, ok := got.Metadata["secret"]; ok {
		t.Fatalf("metadata = %#v, want disallowed secret skipped", got.Metadata)
	}
}

func binderPayload(input binderUserInput) binderUserPayload {
	out := binderUserPayload{
		Name:     input.Name,
		Admin:    input.Admin,
		Roles:    input.Roles,
		Metadata: input.Metadata,
	}
	if input.Profile != nil {
		out.ProfileEmail = input.Profile.Email
		out.ProfileAdmin = input.Profile.Admin
	}
	return out
}
