package web

import (
	"net/http"
	"strings"
)

// RequestMetadata 表示一次 HTTP 请求的不可变元数据输入。
type RequestMetadata struct {
	Method        string
	URL           string
	RequestURI    string
	Path          string
	Headers       http.Header
	ContentLength int64
}

// RequestEntity 表示携带请求方法、URL、请求头和可选请求体的 Web 请求实体。
type RequestEntity[T any] struct {
	method        string
	url           string
	requestURI    string
	path          string
	headers       http.Header
	contentLength int64
	body          T
	hasBody       bool
}

// NewRequestEntity 创建请求实体快照。
func NewRequestEntity[T any](metadata RequestMetadata, body T, hasBody bool) RequestEntity[T] {
	return RequestEntity[T]{
		method:        cleanRequestMethod(metadata.Method),
		url:           cleanRequestMetadataValue(metadata.URL),
		requestURI:    cleanRequestMetadataValue(metadata.RequestURI),
		path:          cleanRequestMetadataValue(metadata.Path),
		headers:       cloneRequestHeaders(metadata.Headers),
		contentLength: metadata.ContentLength,
		body:          body,
		hasBody:       hasBody,
	}
}

// Method 返回 HTTP 请求方法。
func (e RequestEntity[T]) Method() string {
	return e.method
}

// URL 返回包含查询串的完整请求 URL。
func (e RequestEntity[T]) URL() string {
	return e.url
}

// RequestURI 返回不含查询串的请求 URI 路径。
func (e RequestEntity[T]) RequestURI() string {
	return e.requestURI
}

// Path 返回容器分发后的应用内请求路径。
func (e RequestEntity[T]) Path() string {
	return e.path
}

// ContentLength 返回请求体长度，-1 表示容器无法提前确定。
func (e RequestEntity[T]) ContentLength() int64 {
	return e.contentLength
}

// Headers 返回请求头副本。
func (e RequestEntity[T]) Headers() http.Header {
	return cloneRequestHeaders(e.headers)
}

// HeaderValue 返回指定请求头的第一个值。
func (e RequestEntity[T]) HeaderValue(name string) (string, bool) {
	values := e.headers.Values(name)
	if len(values) == 0 {
		return "", false
	}
	return values[0], true
}

// HeaderValues 返回指定请求头的全部值副本。
func (e RequestEntity[T]) HeaderValues(name string) []string {
	return append([]string(nil), e.headers.Values(name)...)
}

// Body 返回请求体和是否显式存在请求体。
func (e RequestEntity[T]) Body() (T, bool) {
	return e.body, e.hasBody
}

// HasBody 表示请求实体是否携带请求体。
func (e RequestEntity[T]) HasBody() bool {
	return e.hasBody
}

func cleanRequestMethod(method string) string {
	method = strings.ToUpper(strings.TrimSpace(method))
	if !isHTTPToken(method) {
		return ""
	}
	return method
}

func cleanRequestMetadataValue(value string) string {
	value = strings.TrimSpace(value)
	if strings.ContainsAny(value, "\r\n") {
		return ""
	}
	return value
}

func cloneRequestHeaders(src http.Header) http.Header {
	if len(src) == 0 {
		return nil
	}
	dst := make(http.Header, len(src))
	for name, values := range src {
		name = cleanHeaderName(name)
		if name == "" {
			continue
		}
		dst[name] = append([]string(nil), values...)
	}
	if len(dst) == 0 {
		return nil
	}
	return dst
}
