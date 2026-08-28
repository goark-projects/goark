package web

import (
	arkweb "goark.dev/arkarta/web"
	"goark.dev/goark/web/stream"
)

// StreamWriter 是 Goark Web 流式写入器。
type StreamWriter = stream.Writer

// StreamFunc 是 Goark Web 流式写入函数。
type StreamFunc = stream.WriterFunc

// SSEWriter 是 Server-Sent Events 写入器。
type SSEWriter = stream.SSEWriter

// SSEFunc 是 Server-Sent Events 事件生产函数。
type SSEFunc = stream.SSEFunc

// SSEEvent 描述一个 Server-Sent Events 事件。
type SSEEvent = stream.Event

// Stream 创建指定媒体类型的流式响应。
func Stream(contentType string, write StreamFunc, options ...stream.Option) arkweb.Result {
	return stream.New(contentType, write, options...)
}

// TextStream 创建 text/plain 流式响应。
func TextStream(write StreamFunc, options ...stream.Option) arkweb.Result {
	return stream.Text(write, options...)
}

// BinaryStream 创建 application/octet-stream 流式响应。
func BinaryStream(write StreamFunc, options ...stream.Option) arkweb.Result {
	return stream.Binary(write, options...)
}

// SSE 创建 Server-Sent Events 响应。
func SSE(write SSEFunc, options ...stream.Option) arkweb.Result {
	return stream.Events(write, options...)
}
