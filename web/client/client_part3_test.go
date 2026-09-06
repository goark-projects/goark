package client_test

import (
	"context"
	"errors"
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

func TestClientInterceptorChainOrder(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(
		http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
			_, _ = io.WriteString(writer, "ok")
		}),
	)
	defer server.Close()

	var order []string
	client, err := webclient.New(
		webclient.WithInterceptor(
			webclient.InterceptorFunc(
				func(ctx context.Context, request *http.Request, next webclient.ExchangeFunc) (
					*http.Response, error) {
					order = append(order, "first-before")
					response, err := next(ctx, request)
					order = append(order, "first-after")
					return response, err
				},
			),
		),
		webclient.WithInterceptor(
			webclient.InterceptorFunc(
				func(ctx context.Context, request *http.Request, next webclient.ExchangeFunc) (
					*http.Response, error) {
					order = append(order, "second-before")
					response, err := next(ctx, request)
					order = append(order, "second-after")
					return response, err
				},
			),
		),
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

func TestClientSendsConditionalHeaders(t *testing.T) {
	t.Parallel()

	modified := time.Date(2026, time.August, 29, 8, 30, 0, 900, time.FixedZone("CST", 8*60*60))
	serverErrors := make(chan error, 1)
	server := httptest.NewServer(
		http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			if request.Header.Get("If-None-Match") != `"job-1", W/"list"` {
				failServer(
					serverErrors,
					writer,
					"If-None-Match = %q",
					request.Header.Get("If-None-Match"),
				)
				return
			}
			if request.Header.Get("If-Modified-Since") != "Sat, 29 Aug 2026 00:30:00 GMT" {
				failServer(
					serverErrors,
					writer,
					"If-Modified-Since = %q",
					request.Header.Get("If-Modified-Since"),
				)
				return
			}
			writer.WriteHeader(http.StatusNotModified)
		}),
	)
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

func TestGetJSONDecodesTypedBody(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(
		http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
			writer.Header().Add("X-Mode", "first")
			writer.Header().Add("X-Mode", "second")
			http.SetCookie(writer, &http.Cookie{Name: "sid", Value: "abc", HttpOnly: true})
			if err := writeJSON(writer, http.StatusOK, jobPayload{ID: "7", Name: "typed"}); err != nil {
				t.Errorf("write json failed: %v", err)
			}
		}),
	)
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
	if values := typed.Response.HeaderValues("X-Mode"); !reflect.DeepEqual(
		values,
		[]string{"first", "second"},
	) {
		t.Fatalf("X-Mode values = %#v, want two values", values)
	}
	cookie, ok := typed.Response.Cookie("sid")
	if !ok || cookie.Value != "abc" || !cookie.HttpOnly {
		t.Fatalf("sid cookie = %#v/%v, want abc HttpOnly", cookie, ok)
	}
}

func TestClientStatusHandlerCanRaiseStatusError(t *testing.T) {
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

func TestClientRejectsInvalidStatusHandlers(t *testing.T) {
	t.Parallel()

	if _, err := webclient.New(webclient.WithStatusHandler(nil, webclient.StatusHandlerFunc(
		webclient.RaiseStatusError))); !errors.Is(
		err,
		webclient.ErrInvalidStatusHandler,
	) {
		t.Fatalf("new err = %v, want ErrInvalidStatusHandler", err)
	}
	client, err := webclient.New()
	if err != nil {
		t.Fatalf("client new failed: %v", err)
	}
	if _, err := client.Get(t.Context(), "http://example.com", webclient.OnStatusFunc(nil,
		webclient.RaiseStatusError)); !errors.Is(
		err,
		webclient.ErrInvalidStatusHandler,
	) {
		t.Fatalf("request err = %v, want ErrInvalidStatusHandler", err)
	}
}

func TestClientRetrieveRejectsNilResponse(t *testing.T) {
	t.Parallel()

	client, err := webclient.New(
		webclient.WithInterceptor(
			webclient.InterceptorFunc(
				func(context.Context, *http.Request, webclient.ExchangeFunc) (*http.Response, error) {
					return nil, nil
				},
			),
		),
	)
	if err != nil {
		t.Fatalf("client new failed: %v", err)
	}
	_, err = client.Get(t.Context(), "http://example.com")
	if !errors.Is(err, webclient.ErrNilHTTPResponse) {
		t.Fatalf("err = %v, want ErrNilHTTPResponse", err)
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
