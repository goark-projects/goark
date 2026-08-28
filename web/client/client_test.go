package client_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	arkjson "goark.dev/arkarta/json"
	webclient "goark.dev/goark/web/client"
)

type createJobRequest struct {
	Name string `json:"name"`
}

type jobPayload struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Trace string `json:"trace"`
}

func TestClientPostJSONWithBaseURLAndInterceptor(t *testing.T) {
	t.Parallel()

	serverErrors := make(chan error, 1)
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost {
			failServer(serverErrors, writer, "method = %q, want POST", request.Method)
			return
		}
		if request.URL.Path != "/api/jobs/42" {
			failServer(serverErrors, writer, "path = %q, want /api/jobs/42", request.URL.Path)
			return
		}
		if request.URL.Query().Get("view") != "full" || request.URL.Query().Get("trace") != "true" {
			failServer(serverErrors, writer, "query = %q", request.URL.RawQuery)
			return
		}
		if request.Header.Get("X-App") != "goark" || request.Header.Get("X-Trace") != "trace-1" {
			failServer(serverErrors, writer, "headers = %#v", request.Header)
			return
		}
		if request.Header.Get("Content-Type") != arkjson.ContentType {
			failServer(serverErrors, writer, "content type = %q", request.Header.Get("Content-Type"))
			return
		}
		body, err := io.ReadAll(request.Body)
		if err != nil {
			failServer(serverErrors, writer, "read request body failed: %v", err)
			return
		}
		if string(body) != `{"name":"sync"}` {
			failServer(serverErrors, writer, "request body = %q", string(body))
			return
		}
		if err := writeJSON(writer, http.StatusCreated, jobPayload{
			ID:    "42",
			Name:  "sync",
			Trace: request.Header.Get("X-Trace"),
		}); err != nil {
			failServer(serverErrors, writer, "write json failed: %v", err)
		}
	}))
	defer server.Close()

	client, err := webclient.New(
		webclient.WithBaseURL(server.URL+"/api"),
		webclient.WithDefaultHeader("X-App", "goark"),
		webclient.WithInterceptor(webclient.InterceptorFunc(func(ctx context.Context, request *http.Request, next webclient.ExchangeFunc) (*http.Response, error) {
			request.Header.Set("X-Trace", "trace-1")
			return next(ctx, request)
		})),
	)
	if err != nil {
		t.Fatalf("client new failed: %v", err)
	}
	response, err := client.Post(t.Context(), "/jobs/{id}?view=full",
		webclient.WithPathParam("id", "42"),
		webclient.WithQueryParam("trace", "true"),
		webclient.WithAccept(arkjson.ContentType),
		webclient.WithJSONBody(createJobRequest{Name: "sync"}),
	)
	if err != nil {
		t.Fatalf("post failed: %v", err)
	}
	assertNoServerError(t, serverErrors)
	if err := response.EnsureSuccess(); err != nil {
		t.Fatalf("status failed: %v", err)
	}
	if response.StatusCode() != http.StatusCreated {
		t.Fatalf("status = %d, want 201", response.StatusCode())
	}

	var payload jobPayload
	if err := response.DecodeJSON(&payload); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if payload != (jobPayload{ID: "42", Name: "sync", Trace: "trace-1"}) {
		t.Fatalf("payload = %#v", payload)
	}
}

func TestBuilderBuildsClientWithoutMutatingBase(t *testing.T) {
	t.Parallel()

	serverErrors := make(chan error, 1)
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Header.Get("X-App") != "goark" || request.Header.Get("X-Derived") != "yes" {
			failServer(serverErrors, writer, "headers = %#v", request.Header)
			return
		}
		_, _ = io.WriteString(writer, "builder")
	}))
	defer server.Close()

	base := webclient.NewBuilder(webclient.WithDefaultHeader("X-App", "goark"))
	derived := base.BaseURL(server.URL).DefaultHeader("X-Derived", "yes")

	baseClient, err := base.Build()
	if err != nil {
		t.Fatalf("base build failed: %v", err)
	}
	baseRequest, err := baseClient.NewRequest(t.Context(), http.MethodGet, server.URL)
	if err != nil {
		t.Fatalf("base request failed: %v", err)
	}
	if baseRequest.Header.Get("X-App") != "goark" || baseRequest.Header.Get("X-Derived") != "" {
		t.Fatalf("base headers = %#v", baseRequest.Header)
	}

	derivedClient, err := derived.Build()
	if err != nil {
		t.Fatalf("derived build failed: %v", err)
	}
	response, err := derivedClient.Get(t.Context(), "/jobs")
	if err != nil {
		t.Fatalf("derived get failed: %v", err)
	}
	assertNoServerError(t, serverErrors)
	if response.BodyString() != "builder" {
		t.Fatalf("body = %q, want builder", response.BodyString())
	}
}

func TestNilBuilderBuildsDefaultClient(t *testing.T) {
	t.Parallel()

	var builder *webclient.Builder
	client, err := builder.Build()
	if err != nil {
		t.Fatalf("nil builder build failed: %v", err)
	}
	request, err := client.NewRequest(t.Context(), http.MethodGet, "http://example.com")
	if err != nil {
		t.Fatalf("new request failed: %v", err)
	}
	if request.URL.String() != "http://example.com" {
		t.Fatalf("url = %q", request.URL.String())
	}
}

func TestClientInterceptorChainOrder(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(writer, "ok")
	}))
	defer server.Close()

	var order []string
	client, err := webclient.New(
		webclient.WithInterceptor(webclient.InterceptorFunc(func(ctx context.Context, request *http.Request, next webclient.ExchangeFunc) (*http.Response, error) {
			order = append(order, "first-before")
			response, err := next(ctx, request)
			order = append(order, "first-after")
			return response, err
		})),
		webclient.WithInterceptor(webclient.InterceptorFunc(func(ctx context.Context, request *http.Request, next webclient.ExchangeFunc) (*http.Response, error) {
			order = append(order, "second-before")
			response, err := next(ctx, request)
			order = append(order, "second-after")
			return response, err
		})),
	)
	if err != nil {
		t.Fatalf("client new failed: %v", err)
	}
	response, err := client.Get(t.Context(), server.URL)
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if response.BodyString() != "ok" {
		t.Fatalf("body = %q, want ok", response.BodyString())
	}

	want := []string{"first-before", "second-before", "second-after", "first-after"}
	if !reflect.DeepEqual(order, want) {
		t.Fatalf("order = %#v, want %#v", order, want)
	}
}

func TestClientExchangeReturnsRawResponse(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(writer, "raw")
	}))
	defer server.Close()

	client, err := webclient.New()
	if err != nil {
		t.Fatalf("client new failed: %v", err)
	}
	response, err := client.Exchange(t.Context(), http.MethodGet, server.URL)
	if err != nil {
		t.Fatalf("exchange failed: %v", err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read raw response failed: %v", err)
	}
	if string(body) != "raw" {
		t.Fatalf("body = %q, want raw", string(body))
	}
}

func TestClientRetrieveRejectsNilResponse(t *testing.T) {
	t.Parallel()

	client, err := webclient.New(webclient.WithInterceptor(webclient.InterceptorFunc(func(context.Context, *http.Request, webclient.ExchangeFunc) (*http.Response, error) {
		return nil, nil
	})))
	if err != nil {
		t.Fatalf("client new failed: %v", err)
	}
	_, err = client.Get(t.Context(), "http://example.com")
	if !errors.Is(err, webclient.ErrNilHTTPResponse) {
		t.Fatalf("err = %v, want ErrNilHTTPResponse", err)
	}
}

func TestResponseEnsureSuccessReturnsStatusError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		http.Error(writer, "missing", http.StatusNotFound)
	}))
	defer server.Close()

	client, err := webclient.New(webclient.WithBaseURL(server.URL))
	if err != nil {
		t.Fatalf("client new failed: %v", err)
	}
	response, err := client.Get(t.Context(), "/missing")
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}

	var statusErr *webclient.StatusError
	if err := response.EnsureSuccess(); !errors.As(err, &statusErr) {
		t.Fatalf("status err = %v, want StatusError", err)
	}
	if statusErr.StatusCode != http.StatusNotFound || !strings.Contains(string(statusErr.Body), "missing") {
		t.Fatalf("status err = %+v", statusErr)
	}
}

func TestClientLimitsRetrievedResponseBody(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(writer, "too-large")
	}))
	defer server.Close()

	client, err := webclient.New(
		webclient.WithBaseURL(server.URL),
		webclient.WithMaxResponseBytes(3),
	)
	if err != nil {
		t.Fatalf("client new failed: %v", err)
	}
	_, err = client.Get(t.Context(), "/payload")
	if !errors.Is(err, webclient.ErrResponseTooLarge) {
		t.Fatalf("err = %v, want ErrResponseTooLarge", err)
	}
}

func TestClientRejectsInvalidRequestConfiguration(t *testing.T) {
	t.Parallel()

	client, err := webclient.New()
	if err != nil {
		t.Fatalf("client new failed: %v", err)
	}
	if _, err := client.Get(t.Context(), "/relative"); !errors.Is(err, webclient.ErrInvalidBaseURL) {
		t.Fatalf("relative err = %v, want ErrInvalidBaseURL", err)
	}
	if _, err := webclient.New(webclient.WithBaseURL("://bad")); !errors.Is(err, webclient.ErrInvalidBaseURL) {
		t.Fatalf("base url err = %v, want ErrInvalidBaseURL", err)
	}
	if _, err := client.Get(t.Context(), "http://example.com", webclient.WithHeader("X-Bad", "a\r\nb")); !errors.Is(err, webclient.ErrInvalidHeader) {
		t.Fatalf("header err = %v, want ErrInvalidHeader", err)
	}
}

func TestWithTimeoutCopiesHTTPClient(t *testing.T) {
	t.Parallel()

	base := &http.Client{Timeout: time.Second}
	client, err := webclient.New(webclient.WithHTTPClient(base), webclient.WithTimeout(2*time.Second))
	if err != nil {
		t.Fatalf("client new failed: %v", err)
	}
	request, err := client.NewRequest(t.Context(), http.MethodGet, "http://example.com")
	if err != nil {
		t.Fatalf("new request failed: %v", err)
	}
	if request.Method != http.MethodGet {
		t.Fatalf("method = %q", request.Method)
	}
	if base.Timeout != time.Second {
		t.Fatalf("base timeout changed to %s", base.Timeout)
	}
}

func writeJSON(writer http.ResponseWriter, statusCode int, value any) error {
	data, err := arkjson.Marshal(nil, value)
	if err != nil {
		return err
	}
	writer.Header().Set("Content-Type", arkjson.ContentType)
	writer.WriteHeader(statusCode)
	_, err = writer.Write(data)
	return err
}

func failServer(errors chan<- error, writer http.ResponseWriter, format string, args ...any) {
	select {
	case errors <- fmt.Errorf(format, args...):
	default:
	}
	http.Error(writer, "server assertion failed", http.StatusInternalServerError)
}

func assertNoServerError(t *testing.T, errors <-chan error) {
	t.Helper()
	select {
	case err := <-errors:
		t.Fatal(err)
	default:
	}
}
