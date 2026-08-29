package mvc_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	arkjson "goark.dev/arkarta/json"
	servletnethttp "goark.dev/arkarta/servlet/nethttp"
	arkweb "goark.dev/arkarta/web"
	"goark.dev/goark/web"
	"goark.dev/goark/web/message"
	"goark.dev/goark/web/mvc"
)

type advisedCreateRequest struct {
	Name string `json:"name"`
}

type panicRequestBodyAdvice struct{}

func (*panicRequestBodyAdvice) BeforeRead(*arkweb.Context, message.ReadAdviceContext) error {
	panic("nil request body advice must not run")
}

func (*panicRequestBodyAdvice) AfterRead(*arkweb.Context, message.ReadAdviceContext) error {
	panic("nil request body advice must not run")
}

func TestBindJSONUsesRequestBodyAdvice(t *testing.T) {
	t.Parallel()

	registry := web.NewRegistry()
	registry.UseRequestBodyAdvice(message.ReadAdviceFunc{
		After: func(_ *arkweb.Context, input message.ReadAdviceContext) error {
			target := input.Target.(*advisedCreateRequest)
			target.Name += "-advised"
			return nil
		},
	})
	if err := mvc.NewRestController("users",
		mvc.POST("/users", mvc.BindJSON(http.StatusCreated, func(_ *arkweb.Context, input advisedCreateRequest) (map[string]string, error) {
			return map[string]string{"name": input.Name}, nil
		})),
	).Register(registry); err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	router, err := registry.Router()
	if err != nil {
		t.Fatalf("Router failed: %v", err)
	}

	recorder := httptestResponse(router, http.MethodPost, "/users", `{"name":"goark"}`)
	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201, body=%s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"name":"goark-advised"`) {
		t.Fatalf("body = %s, want advised request body", recorder.Body.String())
	}
}

func TestControllerAdviceRequestBodyAdviceAdvisesBindJSON(t *testing.T) {
	t.Parallel()

	var typedNil *panicRequestBodyAdvice
	registry := web.NewRegistry()
	advice := mvc.NewRestControllerAdvice("api-bodies").WithRequestBodyAdvice(nil, typedNil, web.RequestBodyAdviceFunc{
		After: func(_ *arkweb.Context, input web.RequestBodyAdviceContext) error {
			target := input.Target.(*advisedCreateRequest)
			target.Name += "-controller-advised"
			return nil
		},
	})
	configurer := mvc.NewConfigurer(mvc.NewRestController("users",
		mvc.POST("/users", mvc.BindJSON(http.StatusCreated, func(_ *arkweb.Context, input advisedCreateRequest) (map[string]string, error) {
			return map[string]string{"name": input.Name}, nil
		})),
	)).WithControllerAdvices(advice)
	if err := configurer.ConfigureWeb(t.Context(), registry); err != nil {
		t.Fatalf("ConfigureWeb failed: %v", err)
	}
	router, err := registry.Router()
	if err != nil {
		t.Fatalf("Router failed: %v", err)
	}

	recorder := httptestResponse(router, http.MethodPost, "/users", `{"name":"goark"}`)
	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201, body=%s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"name":"goark-controller-advised"`) {
		t.Fatalf("body = %s, want controller advice request body", recorder.Body.String())
	}
}

func httptestResponse(router *arkweb.Router, method string, target string, body string) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(method, target, strings.NewReader(body))
	request.Header.Set("Content-Type", arkjson.ContentType)
	request.Header.Set("Accept", arkjson.ContentType)
	servletnethttp.Handler(router).ServeHTTP(recorder, request)
	return recorder
}
