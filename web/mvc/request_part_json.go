package mvc

import (
	"mime"
	"net/http"
	"strings"

	arkjson "goark.dev/arkarta/json"
	"goark.dev/arkarta/servlet"
	arkweb "goark.dev/arkarta/web"
)

// RequestPartJSON 将指定 multipart 文件段按 JSON 绑定到目标类型，不执行结构体验证。
func RequestPartJSON[T any](ctx *arkweb.Context, name string, options ...ParamOption) (T, error) {
	return requestPartJSON[T](ctx, name, nil, false, options...)
}

// ValidatedRequestPartJSON 将指定 multipart 文件段按 JSON 绑定到目标类型，并按可选分组执行结构体验证。
func ValidatedRequestPartJSON[T any](ctx *arkweb.Context, name string, groups []string, options ...ParamOption) (T, error) {
	return requestPartJSON[T](ctx, name, groups, true, options...)
}

func requestPartJSON[T any](ctx *arkweb.Context, name string, groups []string, validate bool, options ...ParamOption) (T, error) {
	var out T
	part, ok, err := requestPart(ctx, name, options...)
	if err != nil || !ok {
		return out, err
	}
	if err := ensureJSONPart(part.Header().Get("Content-Type")); err != nil {
		return out, err
	}
	reader, err := part.Open()
	if err != nil {
		return out, err
	}
	defer reader.Close()
	if err := arkjson.Decode(ctx.JSONCodec(), reader, &out); err != nil {
		return out, servlet.NewHTTPError(http.StatusBadRequest, "请求Part格式非法", err)
	}
	if validate {
		return out, validateBound(ctx, &out, groups)
	}
	return out, nil
}

func ensureJSONPart(contentType string) error {
	contentType = strings.TrimSpace(contentType)
	if contentType == "" {
		return nil
	}
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return arkweb.ErrUnsupportedMediaType
	}
	parts := strings.SplitN(strings.ToLower(mediaType), "/", 2)
	if len(parts) != 2 {
		return arkweb.ErrUnsupportedMediaType
	}
	if parts[0] == "application" && (parts[1] == "json" || strings.HasSuffix(parts[1], "+json")) {
		return nil
	}
	return arkweb.ErrUnsupportedMediaType
}
