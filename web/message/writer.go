package message

import (
	"net/http"

	"goark.dev/arkarta/servlet"
	arkweb "goark.dev/arkarta/web"
)

// Writer 按媒体类型协商选择 Converter 写出响应。
type Writer struct {
	converters []Converter
}

// Option 定制消息写出器。
type Option func(*Writer)

// WithConverters 覆盖消息转换器列表。
func WithConverters(converters ...Converter) Option {
	copied := append([]Converter(nil), converters...)
	return func(writer *Writer) {
		writer.converters = cleanConverters(copied)
	}
}

// WithPrependedConverters 将消息转换器加入默认转换器之前，适合用户覆盖内置写出逻辑。
func WithPrependedConverters(converters ...Converter) Option {
	copied := append([]Converter(nil), converters...)
	return func(writer *Writer) {
		cleaned := cleanConverters(copied)
		if len(cleaned) == 0 {
			return
		}
		current := writer.Converters()
		merged := make([]Converter, 0, len(cleaned)+len(current))
		merged = append(merged, cleaned...)
		merged = append(merged, current...)
		writer.converters = merged
	}
}

// WithAppendedConverters 将消息转换器加入当前转换器之后，适合补充低优先级兜底逻辑。
func WithAppendedConverters(converters ...Converter) Option {
	copied := append([]Converter(nil), converters...)
	return func(writer *Writer) {
		cleaned := cleanConverters(copied)
		if len(cleaned) == 0 {
			return
		}
		current := writer.Converters()
		merged := make([]Converter, 0, len(current)+len(cleaned))
		merged = append(merged, current...)
		merged = append(merged, cleaned...)
		writer.converters = merged
	}
}

// NewWriter 创建消息写出器。
func NewWriter(options ...Option) Writer {
	writer := Writer{converters: DefaultConverters()}
	for _, option := range options {
		if option != nil {
			option(&writer)
		}
	}
	return writer
}

// Converters 返回当前消息转换器快照。
func (w Writer) Converters() []Converter {
	return append([]Converter(nil), w.convertersOrDefault()...)
}

// DefaultConverters 返回 Spring Web 风格默认转换器。
func DefaultConverters() []Converter {
	return []Converter{
		BytesConverter{},
		StringConverter{},
		ReaderConverter{},
		FormConverter{},
		JSONConverter{},
	}
}

// Write 协商媒体类型并写出响应。
func (w Writer) Write(ctx *arkweb.Context, statusCode int, value any, mediaTypes ...string) error {
	if err := ensureContext(ctx); err != nil {
		return err
	}
	candidates := w.candidateMediaTypes(value, mediaTypes)
	selected, ok := NegotiateContentType(ctx.Request(), candidates...)
	if !ok {
		return servlet.NewHTTPError(http.StatusNotAcceptable, http.StatusText(http.StatusNotAcceptable), nil)
	}
	converter, ok := w.converterFor(value, selected)
	if !ok {
		return servlet.NewHTTPError(http.StatusUnsupportedMediaType, http.StatusText(http.StatusUnsupportedMediaType), nil)
	}
	ctx.Response().SetStatus(normalizeStatus(statusCode, http.StatusOK))
	return converter.Write(ctx, value, selected)
}

func (w Writer) candidateMediaTypes(value any, mediaTypes []string) []string {
	if cleaned := cleanMediaTypes(mediaTypes); len(cleaned) > 0 {
		return cleaned
	}
	converters := w.convertersOrDefault()
	out := make([]string, 0, len(converters))
	for _, converter := range converters {
		for _, mediaType := range converter.MediaTypes() {
			if converter.CanWrite(value, mediaType) {
				out = append(out, mediaType)
			}
		}
	}
	return cleanMediaTypes(out)
}

func (w Writer) converterFor(value any, mediaType string) (Converter, bool) {
	for _, converter := range w.convertersOrDefault() {
		if converter.CanWrite(value, mediaType) {
			return converter, true
		}
	}
	return nil, false
}

func (w Writer) convertersOrDefault() []Converter {
	if len(w.converters) == 0 {
		return DefaultConverters()
	}
	return w.converters
}

func cleanConverters(converters []Converter) []Converter {
	if len(converters) == 0 {
		return nil
	}
	out := make([]Converter, 0, len(converters))
	for _, converter := range converters {
		if converter != nil {
			out = append(out, converter)
		}
	}
	return out
}
