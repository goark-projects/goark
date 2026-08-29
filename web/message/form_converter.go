package message

import (
	"net/http"
	"net/url"

	arkjson "goark.dev/arkarta/json"
	"goark.dev/arkarta/servlet"
	arkweb "goark.dev/arkarta/web"
)

// FormConverter 读写 application/x-www-form-urlencoded 表单。
type FormConverter struct{}

// MediaTypes 返回 URL 编码表单媒体类型。
func (FormConverter) MediaTypes() []string {
	return []string{MediaTypeFormURLEncoded}
}

// CanRead 判断目标对象是否可接收 URL 编码表单。
func (FormConverter) CanRead(target any, mediaType string) bool {
	value, ok := target.(*url.Values)
	return ok && value != nil && mediaTypeMatches(mediaType, MediaTypeFormURLEncoded)
}

// Read 将 URL 编码表单请求体读取为 url.Values。
func (FormConverter) Read(ctx *arkweb.Context, target any, _ string) error {
	if err := ensureReadableContext(ctx); err != nil {
		return err
	}
	value, ok := target.(*url.Values)
	if !ok || value == nil {
		return arkjson.ErrNilTarget
	}
	data, err := readBoundedBody(ctx.Request())
	if err != nil {
		return err
	}
	values, err := url.ParseQuery(string(data))
	if err != nil {
		return servlet.NewHTTPError(http.StatusBadRequest, "表单请求体解析失败", err)
	}
	*value = values
	return nil
}

// CanWrite 判断值是否可写为 URL 编码表单。
func (FormConverter) CanWrite(value any, mediaType string) bool {
	if !mediaTypeMatches(mediaType, MediaTypeFormURLEncoded) {
		return false
	}
	switch typed := value.(type) {
	case url.Values:
		return true
	case *url.Values:
		return typed != nil
	default:
		return false
	}
}

// Write 将 url.Values 写为 URL 编码表单。
func (FormConverter) Write(ctx *arkweb.Context, value any, mediaType string) error {
	if err := ensureContext(ctx); err != nil {
		return err
	}
	values, ok := formValues(value)
	if !ok {
		return servlet.NewHTTPError(http.StatusInternalServerError, "message converter cannot write form", nil)
	}
	if err := servlet.SetContentType(ctx.Response(), defaultMediaType(mediaType, MediaTypeFormURLEncoded)); err != nil {
		return err
	}
	_, err := ctx.Response().WriteString(values.Encode())
	return err
}

func formValues(value any) (url.Values, bool) {
	switch typed := value.(type) {
	case url.Values:
		return typed, true
	case *url.Values:
		if typed != nil {
			return *typed, true
		}
	}
	return nil, false
}
