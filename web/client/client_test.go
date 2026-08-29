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

func TestClientSendsDefaultAndRequestCookies(t *testing.T) {
	t.Parallel()

	serverErrors := make(chan error, 1)
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		tenant, err := request.Cookie("tenant")
		if err != nil || tenant.Value != "core" {
			failServer(serverErrors, writer, "tenant cookie = %#v err %v", tenant, err)
			return
		}
		session, err := request.Cookie("sid")
		if err != nil || session.Value != "abc" {
			failServer(serverErrors, writer, "sid cookie = %#v err %v", session, err)
			return
		}
		if request.Header.Get("Cookie") != "tenant=core; sid=abc" {
			failServer(serverErrors, writer, "cookie header = %q", request.Header.Get("Cookie"))
			return
		}
		_, _ = io.WriteString(writer, "cookies")
	}))
	defer server.Close()

	client, err := webclient.NewBuilder().
		BaseURL(server.URL).
		DefaultCookieValue("tenant", "core").
		Build()
	if err != nil {
		t.Fatalf("client build failed: %v", err)
	}
	response, err := client.Get(t.Context(), "/profile", webclient.WithCookieValue("sid", "abc"))
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	assertNoServerError(t, serverErrors)
	if response.BodyString() != "cookies" {
		t.Fatalf("body = %q, want cookies", response.BodyString())
	}
}

func TestClientSendsConditionalHeaders(t *testing.T) {
	t.Parallel()

	modified := time.Date(2026, time.August, 29, 8, 30, 0, 900, time.FixedZone("CST", 8*60*60))
	serverErrors := make(chan error, 1)
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Header.Get("If-None-Match") != `"job-1", W/"list"` {
			failServer(serverErrors, writer, "If-None-Match = %q", request.Header.Get("If-None-Match"))
			return
		}
		if request.Header.Get("If-Modified-Since") != "Sat, 29 Aug 2026 00:30:00 GMT" {
			failServer(serverErrors, writer, "If-Modified-Since = %q", request.Header.Get("If-Modified-Since"))
			return
		}
		writer.WriteHeader(http.StatusNotModified)
	}))
	defer server.Close()

	client, err := webclient.New(webclient.WithBaseURL(server.URL))
	if err != nil {
		t.Fatalf("client new failed: %v", err)
	}
	response, err := client.Get(
		t.Context(),
		"/jobs/1",
		webclient.WithIfNoneMatch("job-1", `W/"list"`),
		webclient.WithIfModifiedSince(modified),
	)
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	assertNoServerError(t, serverErrors)
	if response.StatusCode() != http.StatusNotModified {
		t.Fatalf("status = %d, want 304", response.StatusCode())
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

func TestClientStatusHandlerCanRaiseStatusError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		http.Error(writer, "missing", http.StatusNotFound)
	}))
	defer server.Close()

	client, err := webclient.New(webclient.WithStatusHandlerFunc(webclient.IsErrorStatus, webclient.RaiseStatusError))
	if err != nil {
		t.Fatalf("client new failed: %v", err)
	}
	response, err := client.Get(t.Context(), server.URL)
	var statusErr *webclient.StatusError
	if !errors.As(err, &statusErr) {
		t.Fatalf("err = %v, want StatusError", err)
	}
	if response == nil || response.StatusCode() != http.StatusNotFound {
		t.Fatalf("response = %#v, want 404 response", response)
	}
	if !strings.Contains(string(statusErr.Body), "missing") {
		t.Fatalf("status error body = %q, want missing", string(statusErr.Body))
	}
}

func TestClientStatusHandlersRunInDefaultThenRequestOrder(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusCreated)
		_, _ = io.WriteString(writer, "created")
	}))
	defer server.Close()

	order := make([]string, 0, 2)
	client, err := webclient.New(webclient.WithStatusHandlerFunc(webclient.StatusRange(200, 300), func(context.Context, *webclient.Response) error {
		order = append(order, "default")
		return nil
	}))
	if err != nil {
		t.Fatalf("client new failed: %v", err)
	}
	response, err := client.Get(t.Context(), server.URL, webclient.OnStatusFunc(webclient.StatusCode(http.StatusCreated), func(context.Context, *webclient.Response) error {
		order = append(order, "request")
		return nil
	}))
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if response.BodyString() != "created" {
		t.Fatalf("body = %q, want created", response.BodyString())
	}
	want := []string{"default", "request"}
	if !reflect.DeepEqual(order, want) {
		t.Fatalf("order = %#v, want %#v", order, want)
	}
}

func TestGetJSONDecodesTypedBody(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Add("X-Mode", "first")
		writer.Header().Add("X-Mode", "second")
		http.SetCookie(writer, &http.Cookie{Name: "sid", Value: "abc", HttpOnly: true})
		if err := writeJSON(writer, http.StatusOK, jobPayload{ID: "7", Name: "typed"}); err != nil {
			t.Errorf("write json failed: %v", err)
		}
	}))
	defer server.Close()

	client, err := webclient.New()
	if err != nil {
		t.Fatalf("client new failed: %v", err)
	}
	typed, err := webclient.GetJSON[jobPayload](client, t.Context(), server.URL)
	if err != nil {
		t.Fatalf("get json failed: %v", err)
	}
	if typed.Response.StatusCode() != http.StatusOK {
		t.Fatalf("status = %d, want 200", typed.Response.StatusCode())
	}
	if typed.Body != (jobPayload{ID: "7", Name: "typed"}) {
		t.Fatalf("body = %#v", typed.Body)
	}
	if typed.Response.HeaderValue("X-Mode") != "first" {
		t.Fatalf("X-Mode = %q, want first", typed.Response.HeaderValue("X-Mode"))
	}
	if values := typed.Response.HeaderValues("X-Mode"); !reflect.DeepEqual(values, []string{"first", "second"}) {
		t.Fatalf("X-Mode values = %#v, want two values", values)
	}
	cookie, ok := typed.Response.Cookie("sid")
	if !ok || cookie.Value != "abc" || !cookie.HttpOnly {
		t.Fatalf("sid cookie = %#v/%v, want abc HttpOnly", cookie, ok)
	}
}

func TestRetrieveJSONReturnsResponseWhenStatusHandlerFails(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		http.Error(writer, "missing", http.StatusNotFound)
	}))
	defer server.Close()

	client, err := webclient.New(webclient.WithStatusHandlerFunc(webclient.IsErrorStatus, webclient.RaiseStatusError))
	if err != nil {
		t.Fatalf("client new failed: %v", err)
	}
	typed, err := webclient.GetJSON[jobPayload](client, t.Context(), server.URL)
	var statusErr *webclient.StatusError
	if !errors.As(err, &statusErr) {
		t.Fatalf("err = %v, want StatusError", err)
	}
	if typed.Response == nil || typed.Response.StatusCode() != http.StatusNotFound {
		t.Fatalf("response = %#v, want 404 response", typed.Response)
	}
	if typed.Body != (jobPayload{}) {
		t.Fatalf("typed body = %#v, want zero", typed.Body)
	}
}

func TestClientRejectsInvalidStatusHandlers(t *testing.T) {
	t.Parallel()

	if _, err := webclient.New(webclient.WithStatusHandler(nil, webclient.StatusHandlerFunc(webclient.RaiseStatusError))); !errors.Is(err, webclient.ErrInvalidStatusHandler) {
		t.Fatalf("new err = %v, want ErrInvalidStatusHandler", err)
	}
	client, err := webclient.New()
	if err != nil {
		t.Fatalf("client new failed: %v", err)
	}
	if _, err := client.Get(t.Context(), "http://example.com", webclient.OnStatusFunc(nil, webclient.RaiseStatusError)); !errors.Is(err, webclient.ErrInvalidStatusHandler) {
		t.Fatalf("request err = %v, want ErrInvalidStatusHandler", err)
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
	if _, err := webclient.New(webclient.WithDefaultCookieValue("", "x")); !errors.Is(err, webclient.ErrInvalidCookie) {
		t.Fatalf("default cookie err = %v, want ErrInvalidCookie", err)
	}
	if _, err := client.Get(t.Context(), "http://example.com", webclient.WithCookieValue("bad name", "x")); !errors.Is(err, webclient.ErrInvalidCookie) {
		t.Fatalf("request cookie err = %v, want ErrInvalidCookie", err)
	}
	if _, err := client.Get(t.Context(), "http://example.com", webclient.WithIfNoneMatch(`bad"etag`)); !errors.Is(err, webclient.ErrInvalidHeader) {
		t.Fatalf("If-None-Match err = %v, want ErrInvalidHeader", err)
	}
	if _, err := client.Get(t.Context(), "http://example.com", webclient.WithIfNoneMatch("*", "job-1")); !errors.Is(err, webclient.ErrInvalidHeader) {
		t.Fatalf("If-None-Match wildcard err = %v, want ErrInvalidHeader", err)
	}
	if _, err := client.Get(t.Context(), "http://example.com", webclient.WithIfModifiedSince(time.Time{})); !errors.Is(err, webclient.ErrInvalidHeader) {
		t.Fatalf("If-Modified-Since err = %v, want ErrInvalidHeader", err)
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
