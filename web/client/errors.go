package client

import (
	"errors"
	"fmt"
)

var (
	// ErrNilHTTPClient 表示底层 HTTP 客户端为空。
	ErrNilHTTPClient = errors.New("goark/web/client: http client is nil")
	// ErrInvalidBaseURL 表示基础 URL 配置非法。
	ErrInvalidBaseURL = errors.New("goark/web/client: invalid base url")
	// ErrInvalidHeader 表示请求头名称或值非法。
	ErrInvalidHeader = errors.New("goark/web/client: invalid header")
	// ErrInvalidCookie 表示请求 Cookie 配置非法。
	ErrInvalidCookie = errors.New("goark/web/client: invalid cookie")
	// ErrInvalidRequest 表示请求构造失败。
	ErrInvalidRequest = errors.New("goark/web/client: invalid request")
	// ErrInvalidStatusHandler 表示状态处理器配置非法。
	ErrInvalidStatusHandler = errors.New("goark/web/client: invalid status handler")
	// ErrNilJSONTarget 表示 JSON 解码目标为空。
	ErrNilJSONTarget = errors.New("goark/web/client: json target is nil")
	// ErrNilHTTPResponse 表示底层传输返回了空响应。
	ErrNilHTTPResponse = errors.New("goark/web/client: http response is nil")
	// ErrResponseTooLarge 表示响应体超过客户端限制。
	ErrResponseTooLarge = errors.New("goark/web/client: response body too large")
)

// StatusError 表示非 2xx HTTP 响应。
type StatusError struct {
	StatusCode int
	Status     string
	Body       []byte
}

func (e *StatusError) Error() string {
	if e == nil {
		return "<nil>"
	}
	if e.Status != "" {
		return fmt.Sprintf("goark/web/client: unexpected http status %s", e.Status)
	}
	return fmt.Sprintf("goark/web/client: unexpected http status %d", e.StatusCode)
}
