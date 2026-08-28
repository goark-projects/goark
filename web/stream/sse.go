package stream

import (
	"context"
	"strconv"
	"strings"
	"time"

	arkjson "goark.dev/arkarta/json"
	arkweb "goark.dev/arkarta/web"
)

const (
	// EventStreamContentType 是 Server-Sent Events 标准媒体类型。
	EventStreamContentType = "text/event-stream; charset=utf-8"
)

// SSEFunc 向 Server-Sent Events 流发送事件。
type SSEFunc func(ctx context.Context, writer *SSEWriter) error

// Event 描述一个 Server-Sent Events 事件。
type Event struct {
	ID      string
	Name    string
	Data    any
	Retry   time.Duration
	Comment string
}

// SSEWriter 写出 Server-Sent Events 帧。
type SSEWriter struct {
	writer *Writer
	codec  arkjson.Codec
}

// Events 创建 Server-Sent Events 响应。
func Events(write SSEFunc, options ...Option) arkweb.Result {
	defaults := []Option{
		WithHeader("Cache-Control", "no-cache"),
		WithHeader("X-Accel-Buffering", "no"),
	}
	defaults = append(defaults, options...)
	if write == nil {
		return New(EventStreamContentType, nil, defaults...)
	}
	return New(EventStreamContentType, func(ctx context.Context, writer *Writer) error {
		return write(ctx, &SSEWriter{
			writer: writer,
			codec:  jsonCodecFromContext(ctx),
		})
	}, defaults...)
}

// Send 写出一个 SSE 事件并 Flush。
func (w *SSEWriter) Send(event Event) error {
	if w == nil || w.writer == nil {
		return ErrNilWriter
	}
	payload, err := w.encode(event)
	if err != nil {
		return err
	}
	if _, err := w.writer.WriteString(payload); err != nil {
		return err
	}
	return w.writer.Flush()
}

func (w *SSEWriter) encode(event Event) (string, error) {
	var builder strings.Builder
	writeSSEComment(&builder, event.Comment)
	writeSSEField(&builder, "id", event.ID)
	writeSSEField(&builder, "event", event.Name)
	if event.Retry > 0 {
		writeSSEField(&builder, "retry", strconv.FormatInt(int64(event.Retry/time.Millisecond), 10))
	}
	if event.Data != nil {
		data, err := encodeSSEData(w.codec, event.Data)
		if err != nil {
			return "", err
		}
		writeSSEField(&builder, "data", data)
	}
	builder.WriteByte('\n')
	return builder.String(), nil
}

func encodeSSEData(codec arkjson.Codec, data any) (string, error) {
	switch typed := data.(type) {
	case string:
		return typed, nil
	case []byte:
		return string(typed), nil
	default:
		encoded, err := arkjson.Marshal(codec, typed)
		if err != nil {
			return "", err
		}
		return string(encoded), nil
	}
}

func writeSSEComment(builder *strings.Builder, value string) {
	value = normalizeSSEValue(value)
	if value == "" {
		return
	}
	for _, line := range strings.Split(value, "\n") {
		builder.WriteString(": ")
		builder.WriteString(line)
		builder.WriteByte('\n')
	}
}

func writeSSEField(builder *strings.Builder, name string, value string) {
	value = normalizeSSEValue(value)
	if value == "" {
		return
	}
	for _, line := range strings.Split(value, "\n") {
		builder.WriteString(name)
		builder.WriteString(": ")
		builder.WriteString(line)
		builder.WriteByte('\n')
	}
}

func normalizeSSEValue(value string) string {
	value = strings.ReplaceAll(value, "\r\n", "\n")
	value = strings.ReplaceAll(value, "\r", "\n")
	return strings.TrimSuffix(value, "\n")
}

type jsonCodecContextKey struct{}

func withJSONCodec(ctx context.Context, codec arkjson.Codec) context.Context {
	if codec == nil {
		return ctx
	}
	return context.WithValue(ctx, jsonCodecContextKey{}, codec)
}

func jsonCodecFromContext(ctx context.Context) arkjson.Codec {
	if ctx == nil {
		return nil
	}
	codec, _ := ctx.Value(jsonCodecContextKey{}).(arkjson.Codec)
	return codec
}
