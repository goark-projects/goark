package stream

import (
	"context"
	"net/http"

	servletasync "goark.dev/arkarta/servlet/async"
	arkweb "goark.dev/arkarta/web"
)

const defaultStatus = http.StatusOK

// WriterFunc 向响应流写出业务数据。
type WriterFunc func(ctx context.Context, writer *Writer) error

// Result 表示一个流式 Web 响应。
type Result struct {
	status      int
	contentType string
	headers     http.Header
	write       WriterFunc
}

// New 创建指定媒体类型的流式响应。
func New(contentType string, write WriterFunc, options ...Option) arkweb.Result {
	result := &Result{
		status:      defaultStatus,
		contentType: stringsTrimHeader(contentType),
		headers:     make(http.Header),
		write:       write,
	}
	for _, option := range options {
		if option != nil {
			option(result)
		}
	}
	return result
}

// Text 创建 text/plain 流式响应。
func Text(write WriterFunc, options ...Option) arkweb.Result {
	return New("text/plain; charset=utf-8", write, options...)
}

// Binary 创建 application/octet-stream 流式响应。
func Binary(write WriterFunc, options ...Option) arkweb.Result {
	return New("application/octet-stream", write, options...)
}

// Write 执行流式响应写出。
func (r *Result) Write(ctx *arkweb.Context) error {
	if ctx == nil || ctx.Response() == nil {
		return arkweb.ErrNilContext
	}
	if r == nil || r.write == nil {
		return ErrNilWriterFunc
	}
	if r.contentType != "" {
		ctx.Response().Header().Set("Content-Type", r.contentType)
	}
	for name, values := range r.headers {
		for _, value := range values {
			ctx.Response().Header().Add(name, value)
		}
	}
	ctx.Response().SetStatus(r.status)
	target, err := servletasync.NewStream(ctx.Response())
	if err != nil {
		return err
	}
	writeCtx := withJSONCodec(ctx.Context(), ctx.JSONCodec())
	writer := newWriter(writeCtx, target)
	if err := r.write(writeCtx, writer); err != nil {
		return err
	}
	return writer.Close()
}

func stringsTrimHeader(value string) string {
	values := cleanHeaderValues([]string{value})
	if len(values) == 0 {
		return ""
	}
	return values[0]
}
