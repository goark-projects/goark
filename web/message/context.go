package message

import arkweb "goark.dev/arkarta/web"

const (
	// AttributeReader 是请求属性中保存 Reader 的键。
	AttributeReader = "goark.dev/goark/web/message.reader"
	// AttributeWriter 是请求属性中保存 Writer 的键。
	AttributeWriter = "goark.dev/goark/web/message.writer"
)

// BindContext 将当前请求使用的消息读写器绑定到 Arkarta Web 上下文。
func BindContext(ctx *arkweb.Context, reader Reader, writer Writer) error {
	if ctx == nil || ctx.Request() == nil {
		return arkweb.ErrNilContext
	}
	ctx.Request().SetAttribute(AttributeReader, reader)
	ctx.Request().SetAttribute(AttributeWriter, writer)
	return nil
}

// ReaderFromContext 返回当前请求绑定的读取器；未绑定时返回默认读取器。
func ReaderFromContext(ctx *arkweb.Context) Reader {
	if ctx == nil || ctx.Request() == nil {
		return NewReader()
	}
	value, ok := ctx.Request().Attribute(AttributeReader)
	if !ok {
		return NewReader()
	}
	switch reader := value.(type) {
	case Reader:
		return reader
	case *Reader:
		if reader != nil {
			return *reader
		}
	}
	return NewReader()
}

// WriterFromContext 返回当前请求绑定的写出器；未绑定时返回默认写出器。
func WriterFromContext(ctx *arkweb.Context) Writer {
	if ctx == nil || ctx.Request() == nil {
		return NewWriter()
	}
	value, ok := ctx.Request().Attribute(AttributeWriter)
	if !ok {
		return NewWriter()
	}
	switch writer := value.(type) {
	case Writer:
		return writer
	case *Writer:
		if writer != nil {
			return *writer
		}
	}
	return NewWriter()
}

// ContextInterceptor 在请求进入 MVC 处理前绑定消息读写器。
func ContextInterceptor(reader Reader, writer Writer) arkweb.Interceptor {
	return arkweb.InterceptorFunc(func(ctx *arkweb.Context, next arkweb.Handler) (arkweb.Result, error) {
		if ctx != nil && ctx.Request() != nil {
			ctx.Request().SetAttribute(AttributeReader, reader)
			ctx.Request().SetAttribute(AttributeWriter, writer)
		}
		return next.Handle(ctx)
	})
}
