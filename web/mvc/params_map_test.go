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
	"goark.dev/goark/web/mvc"
)

func TestParameterHelpersBindRequestParamMaps(t *testing.T) {
	t.Parallel()

	type payload struct {
		Params map[string]string   `json:"params"`
		Values map[string][]string `json:"values"`
	}
	router := arkweb.NewRouter()
	err := router.Handle(http.MethodPost, "/search", mvc.JSON(http.StatusOK, func(ctx *arkweb.Context) (payload, error) {
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
	request := httptest.NewRequest(http.MethodPost, "/search?tag=query&empty=", strings.NewReader("tag=form&q=goark"))
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
	if !reflect.DeepEqual(got.Values, map[string][]string{"tag": {"query", "form"}, "empty": {""}, "q": {"goark"}}) {
		t.Fatalf("values = %#v", got.Values)
	}
}

func TestParameterHelpersBindRequestHeaderMaps(t *testing.T) {
	t.Parallel()

	type payload struct {
		Headers map[string]string   `json:"headers"`
		Values  map[string][]string `json:"values"`
	}
	router := arkweb.NewRouter()
	err := router.Handle(http.MethodGet, "/headers", mvc.JSON(http.StatusOK, func(ctx *arkweb.Context) (payload, error) {
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
