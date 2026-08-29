package client

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"time"

	arkjson "goark.dev/arkarta/json"
)

const (
	defaultTimeout          = 30 * time.Second
	defaultMaxResponseBytes = 16 << 20
)

// Option 定制 HTTP 客户端。
type Option func(*Client) error

// Client 是 Goark Web 的同步 HTTP 客户端。
type Client struct {
	httpClient       *http.Client
	baseURL          *url.URL
	defaultHeaders   http.Header
	defaultCookies   []*http.Cookie
	codec            arkjson.Codec
	interceptors     []Interceptor
	statusHandlers   []statusHandler
	maxResponseBytes int64
}

// New 创建 HTTP 客户端。
func New(options ...Option) (*Client, error) {
	client := &Client{
		httpClient:       &http.Client{Timeout: defaultTimeout},
		defaultHeaders:   make(http.Header),
		codec:            arkjson.DefaultCodec(),
		maxResponseBytes: defaultMaxResponseBytes,
	}
	for _, option := range options {
		if option == nil {
			continue
		}
		if err := option(client); err != nil {
			return nil, err
		}
	}
	if client.httpClient == nil {
		return nil, ErrNilHTTPClient
	}
	if client.codec == nil {
		client.codec = arkjson.DefaultCodec()
	}
	return client, nil
}

// WithHTTPClient 设置底层标准库 HTTP 客户端。
func WithHTTPClient(httpClient *http.Client) Option {
	return func(client *Client) error {
		if httpClient == nil {
			return ErrNilHTTPClient
		}
		client.httpClient = httpClient
		return nil
	}
}

// WithTimeout 设置默认 HTTP 请求超时。
func WithTimeout(timeout time.Duration) Option {
	return func(client *Client) error {
		if timeout <= 0 {
			return nil
		}
		copied := *client.httpClient
		copied.Timeout = timeout
		client.httpClient = &copied
		return nil
	}
}

// WithBaseURL 设置相对请求使用的基础 URL。
func WithBaseURL(rawURL string) Option {
	return func(client *Client) error {
		parsed, err := parseBaseURL(rawURL)
		if err != nil {
			return err
		}
		client.baseURL = parsed
		return nil
	}
}

// WithDefaultHeader 设置每次请求都携带的请求头。
func WithDefaultHeader(name string, values ...string) Option {
	name, values, err := cleanHeader(name, values)
	return func(client *Client) error {
		if err != nil {
			return err
		}
		for _, value := range values {
			client.defaultHeaders.Add(name, value)
		}
		return nil
	}
}

// WithJSONCodec 设置请求和响应 JSON 编解码器。
func WithJSONCodec(codec arkjson.Codec) Option {
	return func(client *Client) error {
		if codec != nil {
			client.codec = codec
		}
		return nil
	}
}

// WithInterceptor 追加请求拦截器。
func WithInterceptor(interceptor Interceptor) Option {
	return func(client *Client) error {
		if interceptor != nil {
			client.interceptors = append(client.interceptors, interceptor)
		}
		return nil
	}
}

// WithStatusHandler 追加默认响应状态处理器。
func WithStatusHandler(predicate StatusPredicate, handler StatusHandler) Option {
	statusHandler, err := newStatusHandler(predicate, handler)
	return func(client *Client) error {
		if err != nil {
			return err
		}
		client.statusHandlers = append(client.statusHandlers, statusHandler)
		return nil
	}
}

// WithStatusHandlerFunc 追加函数型默认响应状态处理器。
func WithStatusHandlerFunc(predicate StatusPredicate, handler func(context.Context, *Response) error) Option {
	return WithStatusHandler(predicate, StatusHandlerFunc(handler))
}

// WithMaxResponseBytes 设置响应体快照最大读取字节数；负数表示不限制。
func WithMaxResponseBytes(size int64) Option {
	return func(client *Client) error {
		client.maxResponseBytes = size
		return nil
	}
}

func parseBaseURL(rawURL string) (*url.URL, error) {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return nil, ErrInvalidBaseURL
	}
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return nil, ErrInvalidBaseURL
	}
	return parsed, nil
}
