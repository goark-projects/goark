package client

import (
	"context"
	"net/http"
	"time"

	arkjson "goark.dev/arkarta/json"
)

// Builder 以不可变方式累积 HTTP 客户端选项。
type Builder struct {
	options []Option
}

// NewBuilder 创建 HTTP 客户端构建器。
func NewBuilder(options ...Option) *Builder {
	return (&Builder{}).Apply(options...)
}

// Apply 返回附加选项后的新构建器。
func (b *Builder) Apply(options ...Option) *Builder {
	copied := b.optionSnapshot()
	copied = append(copied, options...)
	return &Builder{options: copied}
}

// HTTPClient 设置底层标准库 HTTP 客户端。
func (b *Builder) HTTPClient(httpClient *http.Client) *Builder {
	return b.Apply(WithHTTPClient(httpClient))
}

// Timeout 设置默认 HTTP 请求超时。
func (b *Builder) Timeout(timeout time.Duration) *Builder {
	return b.Apply(WithTimeout(timeout))
}

// BaseURL 设置相对请求使用的基础 URL。
func (b *Builder) BaseURL(rawURL string) *Builder {
	return b.Apply(WithBaseURL(rawURL))
}

// DefaultHeader 设置每次请求都携带的请求头。
func (b *Builder) DefaultHeader(name string, values ...string) *Builder {
	return b.Apply(WithDefaultHeader(name, values...))
}

// DefaultCookie 设置每次请求都携带的 Cookie。
func (b *Builder) DefaultCookie(cookie *http.Cookie) *Builder {
	return b.Apply(WithDefaultCookie(cookie))
}

// DefaultCookieValue 设置每次请求都携带的简单 Cookie。
func (b *Builder) DefaultCookieValue(name, value string) *Builder {
	return b.Apply(WithDefaultCookieValue(name, value))
}

// JSONCodec 设置请求和响应 JSON 编解码器。
func (b *Builder) JSONCodec(codec arkjson.Codec) *Builder {
	return b.Apply(WithJSONCodec(codec))
}

// Interceptor 追加请求拦截器。
func (b *Builder) Interceptor(interceptor Interceptor) *Builder {
	return b.Apply(WithInterceptor(interceptor))
}

// StatusHandler 追加默认响应状态处理器。
func (b *Builder) StatusHandler(predicate StatusPredicate, handler StatusHandler) *Builder {
	return b.Apply(WithStatusHandler(predicate, handler))
}

// StatusHandlerFunc 追加函数型默认响应状态处理器。
func (b *Builder) StatusHandlerFunc(predicate StatusPredicate, handler func(context.Context, *Response) error) *Builder {
	return b.Apply(WithStatusHandlerFunc(predicate, handler))
}

// MaxResponseBytes 设置响应体快照最大读取字节数。
func (b *Builder) MaxResponseBytes(size int64) *Builder {
	return b.Apply(WithMaxResponseBytes(size))
}

// Build 创建 HTTP 客户端。
func (b *Builder) Build() (*Client, error) {
	return New(b.optionSnapshot()...)
}

func (b *Builder) optionSnapshot() []Option {
	if b == nil || len(b.options) == 0 {
		return nil
	}
	return append([]Option(nil), b.options...)
}
