package client

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/url"
	"strings"

	arkjson "goark.dev/arkarta/json"
)

// RequestOption 定制单次 HTTP 请求。
type RequestOption func(*requestConfig) error

type requestConfig struct {
	codec          arkjson.Codec
	headers        http.Header
	cookies        []*http.Cookie
	query          url.Values
	pathVariables  map[string]string
	body           io.Reader
	statusHandlers []statusHandler
}

func newRequestConfig(codec arkjson.Codec) requestConfig {
	if codec == nil {
		codec = arkjson.DefaultCodec()
	}
	return requestConfig{
		codec:         codec,
		headers:       make(http.Header),
		query:         make(url.Values),
		pathVariables: make(map[string]string),
	}
}

// Exchange 执行请求并返回原始 HTTP 响应，调用方负责关闭响应体。
func (c *Client) Exchange(ctx context.Context, method string, target string, options ...RequestOption) (*http.Response, error) {
	request, _, err := c.newRequest(ctx, method, target, options...)
	if err != nil {
		return nil, err
	}
	return c.exchangeFunc()(ctx, request)
}

// Retrieve 执行请求并读取响应体快照。
func (c *Client) Retrieve(ctx context.Context, method string, target string, options ...RequestOption) (*Response, error) {
	request, config, err := c.newRequest(ctx, method, target, options...)
	if err != nil {
		return nil, err
	}
	raw, err := c.exchangeFunc()(ctx, request)
	if err != nil {
		return nil, err
	}
	if raw == nil {
		return nil, ErrNilHTTPResponse
	}
	if raw.Body != nil {
		defer raw.Body.Close()
	}
	body, err := readResponseBody(raw.Body, c.maxResponseBytes)
	if err != nil {
		return nil, err
	}
	response := newResponse(raw, body, c.codec)
	if err := applyStatusHandlers(ctx, response, c.statusHandlers, config.statusHandlers); err != nil {
		return response, err
	}
	return response, nil
}

// NewRequest 构造 HTTP 请求但不发送。
func (c *Client) NewRequest(ctx context.Context, method string, target string, options ...RequestOption) (*http.Request, error) {
	request, _, err := c.newRequest(ctx, method, target, options...)
	return request, err
}

func (c *Client) newRequest(ctx context.Context, method string, target string, options ...RequestOption) (*http.Request, requestConfig, error) {
	if c == nil || c.httpClient == nil {
		return nil, requestConfig{}, ErrNilHTTPClient
	}
	if ctx == nil {
		ctx = context.Background()
	}
	config := newRequestConfig(c.codec)
	for _, option := range options {
		if option == nil {
			continue
		}
		if err := option(&config); err != nil {
			return nil, requestConfig{}, err
		}
	}
	resolved, err := resolveURL(c.baseURL, target, config.pathVariables, config.query)
	if err != nil {
		return nil, requestConfig{}, err
	}
	request, err := http.NewRequestWithContext(ctx, strings.ToUpper(strings.TrimSpace(method)), resolved, config.body)
	if err != nil {
		return nil, requestConfig{}, err
	}
	request.Header = cloneHeader(c.defaultHeaders)
	appendHeaders(request.Header, config.headers)
	addCookies(request, c.defaultCookies)
	addCookies(request, config.cookies)
	return request, config, nil
}

// Get 执行 GET 请求并读取响应体快照。
func (c *Client) Get(ctx context.Context, target string, options ...RequestOption) (*Response, error) {
	return c.Retrieve(ctx, http.MethodGet, target, options...)
}

// Post 执行 POST 请求并读取响应体快照。
func (c *Client) Post(ctx context.Context, target string, options ...RequestOption) (*Response, error) {
	return c.Retrieve(ctx, http.MethodPost, target, options...)
}

// Put 执行 PUT 请求并读取响应体快照。
func (c *Client) Put(ctx context.Context, target string, options ...RequestOption) (*Response, error) {
	return c.Retrieve(ctx, http.MethodPut, target, options...)
}

// Patch 执行 PATCH 请求并读取响应体快照。
func (c *Client) Patch(ctx context.Context, target string, options ...RequestOption) (*Response, error) {
	return c.Retrieve(ctx, http.MethodPatch, target, options...)
}

// Delete 执行 DELETE 请求并读取响应体快照。
func (c *Client) Delete(ctx context.Context, target string, options ...RequestOption) (*Response, error) {
	return c.Retrieve(ctx, http.MethodDelete, target, options...)
}

// WithHeader 追加请求头。
func WithHeader(name string, values ...string) RequestOption {
	name, values, err := cleanHeader(name, values)
	return func(config *requestConfig) error {
		if err != nil {
			return err
		}
		for _, value := range values {
			config.headers.Add(name, value)
		}
		return nil
	}
}

// WithAccept 设置 Accept 请求头。
func WithAccept(value string) RequestOption {
	return WithHeader("Accept", value)
}

// WithContentType 设置 Content-Type 请求头。
func WithContentType(value string) RequestOption {
	return WithHeader("Content-Type", value)
}

// WithQueryParam 追加查询参数。
func WithQueryParam(name string, values ...string) RequestOption {
	return func(config *requestConfig) error {
		name = strings.TrimSpace(name)
		if name == "" {
			return ErrInvalidRequest
		}
		for _, value := range values {
			config.query.Add(name, value)
		}
		return nil
	}
}

// WithPathParam 设置 URI 模板变量。
func WithPathParam(name string, value string) RequestOption {
	return func(config *requestConfig) error {
		name = strings.TrimSpace(name)
		if name == "" {
			return ErrInvalidRequest
		}
		config.pathVariables[name] = value
		return nil
	}
}

// WithBody 设置原始请求体。
func WithBody(body io.Reader) RequestOption {
	return func(config *requestConfig) error {
		config.body = body
		return nil
	}
}

// WithBodyString 设置字符串请求体。
func WithBodyString(value string) RequestOption {
	return WithBody(strings.NewReader(value))
}

// WithBodyBytes 设置字节请求体。
func WithBodyBytes(value []byte) RequestOption {
	return WithBody(bytes.NewReader(value))
}

// WithJSONBody 使用当前 JSON 编解码器写入请求体。
func WithJSONBody(value any) RequestOption {
	return func(config *requestConfig) error {
		body, err := arkjson.Marshal(config.codec, value)
		if err != nil {
			return err
		}
		config.body = bytes.NewReader(body)
		if config.headers.Get("Content-Type") == "" {
			config.headers.Set("Content-Type", arkjson.ContentType)
		}
		return nil
	}
}

// WithFormBody 写入 application/x-www-form-urlencoded 请求体。
func WithFormBody(values url.Values) RequestOption {
	return func(config *requestConfig) error {
		config.body = strings.NewReader(values.Encode())
		if config.headers.Get("Content-Type") == "" {
			config.headers.Set("Content-Type", "application/x-www-form-urlencoded")
		}
		return nil
	}
}

// OnStatus 追加单次请求响应状态处理器。
func OnStatus(predicate StatusPredicate, handler StatusHandler) RequestOption {
	statusHandler, err := newStatusHandler(predicate, handler)
	return func(config *requestConfig) error {
		if err != nil {
			return err
		}
		config.statusHandlers = append(config.statusHandlers, statusHandler)
		return nil
	}
}

// OnStatusFunc 追加函数型单次请求响应状态处理器。
func OnStatusFunc(predicate StatusPredicate, handler func(context.Context, *Response) error) RequestOption {
	return OnStatus(predicate, StatusHandlerFunc(handler))
}

func cleanHeader(name string, values []string) (string, []string, error) {
	name = http.CanonicalHeaderKey(strings.TrimSpace(name))
	if name == "" || len(values) == 0 {
		return "", nil, ErrInvalidHeader
	}
	cleaned := make([]string, 0, len(values))
	for _, value := range values {
		if strings.ContainsAny(value, "\r\n") {
			return "", nil, ErrInvalidHeader
		}
		cleaned = append(cleaned, value)
	}
	return name, cleaned, nil
}

func cloneHeader(src http.Header) http.Header {
	if len(src) == 0 {
		return make(http.Header)
	}
	out := make(http.Header, len(src))
	for name, values := range src {
		out[name] = append([]string(nil), values...)
	}
	return out
}

func appendHeaders(target http.Header, source http.Header) {
	for name, values := range source {
		for _, value := range values {
			target.Add(name, value)
		}
	}
}
