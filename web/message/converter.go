package message

import (
	"io"
	"net/http"

	arkjson "goark.dev/arkarta/json"
	"goark.dev/arkarta/servlet"
	arkweb "goark.dev/arkarta/web"
)

// Converter 将普通 Go 值写为指定 HTTP 媒体类型。
type Converter interface {
	MediaTypes() []string
	CanWrite(value any, mediaType string) bool
	Write(ctx *arkweb.Context, value any, mediaType string) error
}

// HTTPConverter 表示同时支持请求读取和响应写出的消息转换器。
type HTTPConverter interface {
	Converter
	ReadConverter
}

// JSONConverter 使用 Arkarta sonic JSON Codec 写出响应。
type JSONConverter struct{}

// MediaTypes 返回 JSON 兼容媒体类型。
func (JSONConverter) MediaTypes() []string {
	return []string{MediaTypeJSON}
}

// CanWrite 判断目标媒体类型是否可由 JSON 写出。
func (JSONConverter) CanWrite(_ any, mediaType string) bool {
	return mediaTypeMatches(mediaType, MediaTypeJSON) || structuredJSONType(mediaType)
}

// Write 写出 JSON 响应体。
func (JSONConverter) Write(ctx *arkweb.Context, value any, mediaType string) error {
	if err := ensureContext(ctx); err != nil {
		return err
	}
	if err := servlet.SetContentType(ctx.Response(), defaultMediaType(mediaType, MediaTypeJSON)); err != nil {
		return err
	}
	return arkjson.Encode(ctx.JSONCodec(), ctx.Response().BodyWriter(), value)
}

// StringConverter 写出字符串响应。
type StringConverter struct{}

// MediaTypes 返回文本媒体类型。
func (StringConverter) MediaTypes() []string {
	return []string{MediaTypeTextPlain}
}

// CanWrite 判断值是否为字符串。
func (StringConverter) CanWrite(value any, mediaType string) bool {
	_, ok := value.(string)
	return ok && mediaTypeMatches(mediaType, MediaTypeTextPlain)
}

// Write 写出字符串响应体。
func (StringConverter) Write(ctx *arkweb.Context, value any, mediaType string) error {
	if err := ensureContext(ctx); err != nil {
		return err
	}
	text, ok := value.(string)
	if !ok {
		return servlet.NewHTTPError(http.StatusInternalServerError, "message converter cannot write string", nil)
	}
	if err := servlet.SetContentType(ctx.Response(), defaultMediaType(mediaType, MediaTypeTextPlain)); err != nil {
		return err
	}
	_, err := ctx.Response().WriteString(text)
	return err
}

// BytesConverter 写出二进制响应。
type BytesConverter struct{}

// MediaTypes 返回二进制媒体类型。
func (BytesConverter) MediaTypes() []string {
	return []string{MediaTypeOctetStream}
}

// CanWrite 判断值是否为字节切片。
func (BytesConverter) CanWrite(value any, mediaType string) bool {
	_, ok := value.([]byte)
	return ok && mediaTypeMatches(mediaType, MediaTypeOctetStream)
}

// Write 写出字节响应体。
func (BytesConverter) Write(ctx *arkweb.Context, value any, mediaType string) error {
	if err := ensureContext(ctx); err != nil {
		return err
	}
	data, ok := value.([]byte)
	if !ok {
		return servlet.NewHTTPError(http.StatusInternalServerError, "message converter cannot write bytes", nil)
	}
	if err := servlet.SetContentType(ctx.Response(), defaultMediaType(mediaType, MediaTypeOctetStream)); err != nil {
		return err
	}
	_, err := ctx.Response().Write(data)
	return err
}

// ReaderConverter 流式写出 io.Reader 响应。
type ReaderConverter struct{}

// MediaTypes 返回默认流媒体类型。
func (ReaderConverter) MediaTypes() []string {
	return []string{MediaTypeOctetStream}
}

// CanWrite 判断值是否实现 io.Reader。
func (ReaderConverter) CanWrite(value any, mediaType string) bool {
	_, ok := value.(io.Reader)
	return ok && mediaTypeMatches(mediaType, MediaTypeOctetStream)
}

// Write 流式复制响应体。
func (ReaderConverter) Write(ctx *arkweb.Context, value any, mediaType string) (err error) {
	if err := ensureContext(ctx); err != nil {
		return err
	}
	reader, ok := value.(io.Reader)
	if !ok {
		return servlet.NewHTTPError(http.StatusInternalServerError, "message converter cannot write stream", nil)
	}
	if closer, ok := reader.(io.Closer); ok {
		defer func() {
			err = joinErrors(err, closer.Close())
		}()
	}
	if err := servlet.SetContentType(ctx.Response(), defaultMediaType(mediaType, MediaTypeOctetStream)); err != nil {
		return err
	}
	_, err = io.Copy(ctx.Response().BodyWriter(), reader)
	return err
}
