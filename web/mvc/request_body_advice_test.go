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

func httptestResponse(router *arkweb.Router, method string, target string, body string) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(method, target, strings.NewReader(body))
	request.Header.Set("Content-Type", arkjson.ContentType)
	request.Header.Set("Accept", arkjson.ContentType)
	servletnethttp.Handler(router).ServeHTTP(recorder, request)
	return recorder
}
