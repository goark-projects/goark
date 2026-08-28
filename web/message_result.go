package web

import (
	arkweb "goark.dev/arkarta/web"
	"goark.dev/goark/web/message"
)

type messageResult struct {
	statusCode int
	value      any
	mediaTypes []string
}

// Message 创建基于消息转换器和 Accept 协商的响应结果。
func Message(statusCode int, value any, mediaTypes ...string) arkweb.Result {
	return messageResult{
		statusCode: statusCode,
		value:      value,
		mediaTypes: append([]string(nil), mediaTypes...),
	}
}

// Write 将消息响应写入 Arkarta Web 上下文。
func (r messageResult) Write(ctx *arkweb.Context) error {
	return message.NewWriter().Write(ctx, r.statusCode, r.value, r.mediaTypes...)
}
