package client_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	webclient "goark.dev/goark/web/client"
)

func TestClientRejectsInvalidRequestConfiguration(t *testing.T) {
	t.Parallel()

	client, err := webclient.New()
	if err != nil {
		t.Fatalf("client new failed: %v", err)
	}
	if _, err := client.Get(t.Context(), "/relative"); !errors.Is(
		err,
		webclient.ErrInvalidBaseURL,
	) {
		t.Fatalf("relative err = %v, want ErrInvalidBaseURL", err)
	}
	if _, err := webclient.New(webclient.WithBaseURL("://bad")); !errors.Is(
		err,
		webclient.ErrInvalidBaseURL,
	) {
		t.Fatalf("base url err = %v, want ErrInvalidBaseURL", err)
	}
	if _, err := client.Get(t.Context(), "http://example.com", webclient.WithHeader("X-Bad",
		"a\r\nb")); !errors.Is(
		err,
		webclient.ErrInvalidHeader,
	) {
		t.Fatalf("header err = %v, want ErrInvalidHeader", err)
	}
	if _, err := webclient.New(webclient.WithDefaultCookieValue("", "x")); !errors.Is(
		err,
		webclient.ErrInvalidCookie,
	) {
		t.Fatalf("default cookie err = %v, want ErrInvalidCookie", err)
	}
	if _, err := client.Get(t.Context(), "http://example.com", webclient.WithCookieValue(
		"bad name", "x")); !errors.Is(
		err,
		webclient.ErrInvalidCookie,
	) {
		t.Fatalf("request cookie err = %v, want ErrInvalidCookie", err)
	}
	if _, err := client.Get(t.Context(), "http://example.com", webclient.WithIfNoneMatch(
		`bad"etag`)); !errors.Is(
		err,
		webclient.ErrInvalidHeader,
	) {
		t.Fatalf("If-None-Match err = %v, want ErrInvalidHeader", err)
	}
	if _, err := client.Get(t.Context(), "http://example.com", webclient.WithIfNoneMatch("*",
		"job-1")); !errors.Is(
		err,
		webclient.ErrInvalidHeader,
	) {
		t.Fatalf("If-None-Match wildcard err = %v, want ErrInvalidHeader", err)
	}
	if _, err := client.Get(t.Context(), "http://example.com", webclient.WithIfModifiedSince(
		time.Time{})); !errors.Is(
		err,
		webclient.ErrInvalidHeader,
	) {
		t.Fatalf("If-Modified-Since err = %v, want ErrInvalidHeader", err)
	}
}

func TestClientStatusHandlersRunInDefaultThenRequestOrder(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(
		http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
			writer.WriteHeader(http.StatusCreated)
			_, _ = io.WriteString(writer, "created")
		}),
	)
	defer server.Close()

	order := make([]string, 0, 2)
	client, err := webclient.New(
		webclient.WithStatusHandlerFunc(
			webclient.StatusRange(200, 300),
			func(context.Context, *webclient.Response) error {
				order = append(order, "default")
				return nil
			},
		),
	)
	if err != nil {
		t.Fatalf("client new failed: %v", err)
	}
	response, err := client.Get(
		t.Context(),
		server.URL,
		webclient.OnStatusFunc(
			webclient.StatusCode(http.StatusCreated),
			func(context.Context, *webclient.Response) error {
				order = append(order, "request")
				return nil
			},
		),
	)
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

func TestClientSendsDefaultAndRequestCookies(t *testing.T) {
	t.Parallel()

	serverErrors := make(chan error, 1)
	server := httptest.NewServer(
		http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
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
		}),
	)
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

func TestRetrieveJSONReturnsResponseWhenStatusHandlerFails(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(
		http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
			http.Error(writer, "missing", http.StatusNotFound)
		}),
	)
	defer server.Close()

	client, err := webclient.New(
		webclient.WithStatusHandlerFunc(webclient.IsErrorStatus, webclient.RaiseStatusError),
	)
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

func TestClientLimitsRetrievedResponseBody(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(
		http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
			_, _ = io.WriteString(writer, "too-large")
		}),
	)
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

func assertNoServerError(t *testing.T, errors <-chan error) {
	t.Helper()
	select {
	case err := <-errors:
		t.Fatal(err)
	default:
	}
}

type createJobRequest struct {
	Name string `json:"name"`
}
