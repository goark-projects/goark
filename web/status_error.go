package web

import "goark.dev/arkarta/servlet"

// StatusError 表示可由 Goark Web 映射为 HTTP 状态响应的错误。
type StatusError = servlet.StatusError

// ResponseStatusException 是 Spring ResponseStatusException 的 Go 等价类型。
type ResponseStatusException = servlet.HTTPError

// NewStatusError 创建带 HTTP 状态码和安全公开消息的错误。
func NewStatusError(statusCode int, publicMessage string, cause error) error {
	return servlet.NewHTTPError(statusCode, publicMessage, cause)
}

// NewResponseStatusException 创建可直接从处理器返回的 HTTP 状态异常。
func NewResponseStatusException(statusCode int, reason string, cause error) *ResponseStatusException {
	return servlet.NewHTTPError(statusCode, reason, cause)
}
