package message

import (
	"io"
	"net/http"
	"strings"

	arkjson "goark.dev/arkarta/json"
	"goark.dev/arkarta/servlet"
	arkweb "goark.dev/arkarta/web"
)

// CanRead 判断目标对象和媒体类型是否可由 JSON 读取。
func (JSONConverter) CanRead(target any, mediaType string) bool {
	return !nilTarget(target) && (mediaTypeMatches(mediaType, MediaTypeJSON) || structuredJSONType(mediaType))
}

// Read 将 JSON 请求体读取到目标对象。
func (JSONConverter) Read(ctx *arkweb.Context, target any, _ string) error {
	if err := ensureReadableContext(ctx); err != nil {
		return err
	}
	return ctx.BindJSON(target)
}

// CanRead 判断目标对象和媒体类型是否可由字符串读取。
func (StringConverter) CanRead(target any, mediaType string) bool {
	value, ok := target.(*string)
	return ok && value != nil && textMediaType(mediaType)
}

// Read 将文本请求体读取为字符串。
func (StringConverter) Read(ctx *arkweb.Context, target any, _ string) error {
	if err := ensureReadableContext(ctx); err != nil {
		return err
	}
	value, ok := target.(*string)
	if !ok || value == nil {
		return arkjson.ErrNilTarget
	}
	data, err := readBoundedBody(ctx.Request())
	if err != nil {
		return err
	}
	*value = string(data)
	return nil
}

// CanRead 判断目标对象和媒体类型是否可由字节切片读取。
func (BytesConverter) CanRead(target any, mediaType string) bool {
	value, ok := target.(*[]byte)
	return ok && value != nil && validMediaType(mediaType)
}

// Read 将请求体读取为字节切片。
func (BytesConverter) Read(ctx *arkweb.Context, target any, _ string) error {
	if err := ensureReadableContext(ctx); err != nil {
		return err
	}
	value, ok := target.(*[]byte)
	if !ok || value == nil {
		return arkjson.ErrNilTarget
	}
	data, err := readBoundedBody(ctx.Request())
	if err != nil {
		return err
	}
	*value = data
	return nil
}

func readBoundedBody(request *servlet.Request) ([]byte, error) {
	if request == nil || request.Body() == nil {
		return nil, arkjson.ErrNilReader
	}
	if length := request.ContentLength(); length > arkjson.DefaultMaxBytes {
		return nil, arkjson.ErrPayloadTooLarge
	}
	reader := io.LimitReader(request.Body(), arkjson.DefaultMaxBytes+1)
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, servlet.NewHTTPError(http.StatusBadRequest, "请求体读取失败", err)
	}
	if int64(len(data)) > arkjson.DefaultMaxBytes {
		return nil, arkjson.ErrPayloadTooLarge
	}
	return data, nil
}

func textMediaType(mediaType string) bool {
	typ, _, ok := parseMediaType(mediaType)
	if !ok {
		return false
	}
	return typ == "text/plain" || strings.HasPrefix(typ, "text/")
}

func validMediaType(mediaType string) bool {
	_, _, ok := parseMediaType(mediaType)
	return ok
}
