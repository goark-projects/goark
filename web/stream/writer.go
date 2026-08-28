package stream

import (
	"context"

	servletasync "goark.dev/arkarta/servlet/async"
)

// Writer 提供带上下文检查和串行化保护的流式写入。
type Writer struct {
	ctx    context.Context
	stream *servletasync.Stream
}

func newWriter(ctx context.Context, target *servletasync.Stream) *Writer {
	return &Writer{
		ctx:    ctx,
		stream: target,
	}
}

// Write 写出二进制片段。
func (w *Writer) Write(data []byte) (int, error) {
	if w == nil || w.stream == nil {
		return 0, ErrNilWriter
	}
	return w.stream.Write(w.ctx, data)
}

// WriteString 写出字符串片段。
func (w *Writer) WriteString(value string) (int, error) {
	return w.Write([]byte(value))
}

// Flush 刷新当前响应片段。
func (w *Writer) Flush() error {
	if w == nil || w.stream == nil {
		return ErrNilWriter
	}
	return w.stream.Flush(w.ctx)
}

// Close 完成流式写入并执行最终 Flush。
func (w *Writer) Close() error {
	if w == nil || w.stream == nil {
		return ErrNilWriter
	}
	return w.stream.Close(w.ctx)
}
