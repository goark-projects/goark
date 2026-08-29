package message

import (
	"strings"

	arkjson "goark.dev/arkarta/json"
	arkweb "goark.dev/arkarta/web"
)

// ReadConverter 将指定 HTTP 媒体类型的请求体读取到 Go 目标对象。
type ReadConverter interface {
	MediaTypes() []string
	CanRead(target any, mediaType string) bool
	Read(ctx *arkweb.Context, target any, mediaType string) error
}

// Reader 按请求 Content-Type 选择 ReadConverter 读取请求体。
type Reader struct {
	converters []ReadConverter
}

// ReaderOption 定制请求体读取器。
type ReaderOption func(*Reader)

// WithReadConverters 覆盖请求体读取转换器列表。
func WithReadConverters(converters ...ReadConverter) ReaderOption {
	copied := append([]ReadConverter(nil), converters...)
	return func(reader *Reader) {
		reader.converters = cleanReadConverters(copied)
	}
}

// WithPrependedReadConverters 将读取转换器加入默认转换器之前，适合用户覆盖内置绑定逻辑。
func WithPrependedReadConverters(converters ...ReadConverter) ReaderOption {
	copied := append([]ReadConverter(nil), converters...)
	return func(reader *Reader) {
		cleaned := cleanReadConverters(copied)
		if len(cleaned) == 0 {
			return
		}
		current := reader.ReadConverters()
		merged := make([]ReadConverter, 0, len(cleaned)+len(current))
		merged = append(merged, cleaned...)
		merged = append(merged, current...)
		reader.converters = merged
	}
}

// WithAppendedReadConverters 将读取转换器加入当前转换器之后，适合补充低优先级兜底逻辑。
func WithAppendedReadConverters(converters ...ReadConverter) ReaderOption {
	copied := append([]ReadConverter(nil), converters...)
	return func(reader *Reader) {
		cleaned := cleanReadConverters(copied)
		if len(cleaned) == 0 {
			return
		}
		current := reader.ReadConverters()
		merged := make([]ReadConverter, 0, len(current)+len(cleaned))
		merged = append(merged, current...)
		merged = append(merged, cleaned...)
		reader.converters = merged
	}
}

// NewReader 创建请求体读取器。
func NewReader(options ...ReaderOption) Reader {
	reader := Reader{converters: DefaultReadConverters()}
	for _, option := range options {
		if option != nil {
			option(&reader)
		}
	}
	return reader
}

// ReadConverters 返回当前请求体读取转换器快照。
func (r Reader) ReadConverters() []ReadConverter {
	return append([]ReadConverter(nil), r.convertersOrDefault()...)
}

// DefaultReadConverters 返回 Spring Web 风格默认请求体转换器。
func DefaultReadConverters() []ReadConverter {
	return []ReadConverter{
		StringConverter{},
		BytesConverter{},
		JSONConverter{},
	}
}

// Read 基于 Content-Type 将请求体读取到目标对象。
func (r Reader) Read(ctx *arkweb.Context, target any, mediaTypes ...string) error {
	if err := ensureReadableContext(ctx); err != nil {
		return err
	}
	if nilTarget(target) {
		return arkjson.ErrNilTarget
	}
	contentType := strings.TrimSpace(ctx.Request().Header().Get("Content-Type"))
	if contentType == "" {
		selected, ok := r.defaultReadMediaType(target, mediaTypes)
		if !ok {
			return arkweb.ErrUnsupportedMediaType
		}
		return r.readWith(ctx, target, selected)
	}
	candidates := r.candidateMediaTypes(target, mediaTypes)
	if len(candidates) > 0 && !r.contentTypeAllowed(target, contentType, candidates) {
		return arkweb.ErrUnsupportedMediaType
	}
	return r.readWith(ctx, target, contentType)
}

func (r Reader) defaultReadMediaType(target any, mediaTypes []string) (string, bool) {
	for _, mediaType := range r.candidateMediaTypes(target, mediaTypes) {
		if _, ok := r.converterFor(target, mediaType); ok {
			return mediaType, true
		}
	}
	return "", false
}

func (r Reader) candidateMediaTypes(target any, mediaTypes []string) []string {
	if cleaned := cleanMediaTypes(mediaTypes); len(cleaned) > 0 {
		return cleaned
	}
	converters := r.convertersOrDefault()
	out := make([]string, 0, len(converters))
	for _, converter := range converters {
		for _, mediaType := range converter.MediaTypes() {
			if converter.CanRead(target, mediaType) {
				out = append(out, mediaType)
			}
		}
	}
	return cleanMediaTypes(out)
}

func (r Reader) contentTypeAllowed(target any, contentType string, candidates []string) bool {
	for _, candidate := range candidates {
		for _, converter := range r.convertersOrDefault() {
			if converter.CanRead(target, candidate) && converter.CanRead(target, contentType) {
				return true
			}
		}
	}
	return false
}

func (r Reader) readWith(ctx *arkweb.Context, target any, mediaType string) error {
	converter, ok := r.converterFor(target, mediaType)
	if !ok {
		return arkweb.ErrUnsupportedMediaType
	}
	return converter.Read(ctx, target, mediaType)
}

func (r Reader) converterFor(target any, mediaType string) (ReadConverter, bool) {
	for _, converter := range r.convertersOrDefault() {
		if converter.CanRead(target, mediaType) {
			return converter, true
		}
	}
	return nil, false
}

func (r Reader) convertersOrDefault() []ReadConverter {
	if len(r.converters) == 0 {
		return DefaultReadConverters()
	}
	return r.converters
}

func cleanReadConverters(converters []ReadConverter) []ReadConverter {
	if len(converters) == 0 {
		return nil
	}
	out := make([]ReadConverter, 0, len(converters))
	for _, converter := range converters {
		if converter != nil {
			out = append(out, converter)
		}
	}
	return out
}
