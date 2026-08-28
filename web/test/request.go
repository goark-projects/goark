package webtest

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/url"
	"strings"

	arkjson "goark.dev/arkarta/json"
)

// RequestOption 定制测试请求。
type RequestOption func(*requestConfig) error

type requestConfig struct {
	codec      arkjson.Codec
	body       io.Reader
	headers    http.Header
	cookies    []*http.Cookie
	host       string
	remoteAddr string
}

func newRequest(method string, target string, codec arkjson.Codec, options ...RequestOption) (*http.Request, error) {
	config := requestConfig{
		codec:      codec,
		headers:    make(http.Header),
		remoteAddr: "192.0.2.1:1234",
	}
	if config.codec == nil {
		config.codec = arkjson.DefaultCodec()
	}
	for _, option := range options {
		if option == nil {
			continue
		}
		if err := option(&config); err != nil {
			return nil, err
		}
	}
	request, err := http.NewRequestWithContext(context.Background(), method, target, config.body)
	if err != nil {
		return nil, err
	}
	request.Header = cloneHeader(config.headers)
	for _, cookie := range config.cookies {
		if cookie != nil {
			request.AddCookie(cookie)
		}
	}
	if config.host != "" {
		request.Host = config.host
	}
	request.RemoteAddr = config.remoteAddr
	return request, nil
}

// NewRequest 创建使用默认 sonic JSON 编解码器的 HTTP 请求。
func NewRequest(method string, target string, options ...RequestOption) (*http.Request, error) {
	return newRequest(method, target, arkjson.DefaultCodec(), options...)
}

// WithHeader 追加请求头。
func WithHeader(name string, values ...string) RequestOption {
	return func(config *requestConfig) error {
		name = http.CanonicalHeaderKey(strings.TrimSpace(name))
		if name == "" || len(values) == 0 {
			return ErrInvalidHeader
		}
		for _, value := range values {
			if strings.ContainsAny(value, "\r\n") {
				return ErrInvalidHeader
			}
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
	return func(config *requestConfig) error {
		if strings.ContainsAny(value, "\r\n") {
			return ErrInvalidHeader
		}
		config.headers.Set("Content-Type", value)
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

// WithJSONBody 使用当前 WebTest JSON 编解码器写入请求体。
func WithJSONBody(value any) RequestOption {
	return func(config *requestConfig) error {
		data, err := arkjson.Marshal(config.codec, value)
		if err != nil {
			return err
		}
		config.body = bytes.NewReader(data)
		if config.headers.Get("Content-Type") == "" {
			config.headers.Set("Content-Type", arkjson.ContentType)
		}
		return nil
	}
}

// WithFormBody 设置 application/x-www-form-urlencoded 请求体。
func WithFormBody(values url.Values) RequestOption {
	return func(config *requestConfig) error {
		config.body = strings.NewReader(values.Encode())
		if config.headers.Get("Content-Type") == "" {
			config.headers.Set("Content-Type", "application/x-www-form-urlencoded")
		}
		return nil
	}
}

// WithCookie 追加请求 Cookie。
func WithCookie(cookie *http.Cookie) RequestOption {
	return func(config *requestConfig) error {
		if cookie != nil {
			config.cookies = append(config.cookies, cookie)
		}
		return nil
	}
}

// WithHost 设置请求 Host。
func WithHost(host string) RequestOption {
	return func(config *requestConfig) error {
		config.host = strings.TrimSpace(host)
		return nil
	}
}

// WithRemoteAddr 设置请求远端地址。
func WithRemoteAddr(remoteAddr string) RequestOption {
	return func(config *requestConfig) error {
		if remoteAddr = strings.TrimSpace(remoteAddr); remoteAddr != "" {
			config.remoteAddr = remoteAddr
		}
		return nil
	}
}

func cloneHeader(src http.Header) http.Header {
	if len(src) == 0 {
		return make(http.Header)
	}
	dst := make(http.Header, len(src))
	for name, values := range src {
		dst[name] = append([]string(nil), values...)
	}
	return dst
}
