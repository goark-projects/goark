package web

import "goark.dev/arkarta/servlet"

// StatusError 表示可由 Goark Web 映射为 HTTP 状态响应的错误。
type StatusError = servlet.StatusError

// NewStatusError 创建带 HTTP 状态码和安全公开消息的错误。
func NewStatusError(statusCode int, publicMessage string, cause error) error {
	return servlet.NewHTTPError(statusCode, publicMessage, cause)
}
