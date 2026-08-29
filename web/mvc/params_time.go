package mvc

import (
	"net/http"
	"strings"
	"time"

	arkweb "goark.dev/arkarta/web"
)

var defaultParamTimeLayouts = []string{
	time.RFC3339Nano,
	time.RFC3339,
	http.TimeFormat,
	"2006-01-02",
	"2006-01-02 15:04:05",
}

// WithTimeLayout 追加时间参数解析布局，优先于内置布局尝试。
func WithTimeLayout(layout string) ParamOption {
	layout = strings.TrimSpace(layout)
	return func(options *paramOptions) {
		if layout != "" {
			options.timeLayouts = append(options.timeLayouts, layout)
		}
	}
}

// PathTime 绑定 time.Time 路径变量。
func PathTime(ctx *arkweb.Context, name string, options ...ParamOption) (time.Time, error) {
	if ctx == nil {
		return time.Time{}, arkweb.ErrNilContext
	}
	value, ok := ctx.Param(name)
	value = stripMatrixSegment(value)
	return resolveTimeParameter("路径变量", name, value, ok, nil, newParamOptions(ctx, options))
}

// RequestParamTime 绑定 time.Time 请求参数，参数视图包含 query 和 urlencoded form。
func RequestParamTime(ctx *arkweb.Context, name string, options ...ParamOption) (time.Time, error) {
	value, ok, err := formValue(ctx, name)
	return resolveTimeParameter("请求参数", name, value, ok, err, newParamOptions(ctx, options))
}

// RequestHeaderTime 绑定 time.Time 请求头。
func RequestHeaderTime(ctx *arkweb.Context, name string, options ...ParamOption) (time.Time, error) {
	value, ok, err := headerValue(ctx, name)
	return resolveTimeParameter("请求头", name, value, ok, err, newParamOptions(ctx, options))
}

// CookieValueTime 绑定 time.Time Cookie 值。
func CookieValueTime(ctx *arkweb.Context, name string, options ...ParamOption) (time.Time, error) {
	value, ok, err := cookieValue(ctx, name)
	return resolveTimeParameter("Cookie", name, value, ok, err, newParamOptions(ctx, options))
}

// MatrixVariableTime 绑定 time.Time 矩阵变量。
func MatrixVariableTime(ctx *arkweb.Context, name string, options ...ParamOption) (time.Time, error) {
	paramOptions := newParamOptions(ctx, options)
	value, ok, err := matrixValue(ctx, name, paramOptions.matrixPathVariable)
	return resolveTimeParameter("矩阵变量", name, value, ok, err, paramOptions)
}

// RequestAttributeTime 绑定 time.Time 请求属性。
func RequestAttributeTime(ctx *arkweb.Context, name string, options ...ParamOption) (time.Time, error) {
	value, ok, err := requestAttributeValue(ctx, name)
	return resolveTimeParameter("请求属性", name, value, ok, err, newParamOptions(ctx, options))
}

// SessionAttributeTime 绑定 time.Time Session 属性。
func SessionAttributeTime(ctx *arkweb.Context, name string, options ...ParamOption) (time.Time, error) {
	value, ok, err := sessionAttributeValue(ctx, name)
	return resolveTimeParameter("Session属性", name, value, ok, err, newParamOptions(ctx, options))
}

func resolveTimeParameter(kind, name, value string, ok bool, err error, options paramOptions) (time.Time, error) {
	return resolveConvertedParameter(kind, name, value, ok, err, options, "time.Time", func(value string) (time.Time, error) {
		return parseParamTime(value, options)
	})
}

func parseParamTime(value string, options paramOptions) (time.Time, error) {
	layouts := paramTimeLayouts(options)
	var lastErr error
	for _, layout := range layouts {
		parsed, err := time.Parse(layout, value)
		if err == nil {
			return parsed, nil
		}
		lastErr = err
	}
	return time.Time{}, lastErr
}

func paramTimeLayouts(options paramOptions) []string {
	if len(options.timeLayouts) == 0 {
		return defaultParamTimeLayouts
	}
	layouts := make([]string, 0, len(options.timeLayouts)+len(defaultParamTimeLayouts))
	layouts = append(layouts, options.timeLayouts...)
	layouts = append(layouts, defaultParamTimeLayouts...)
	return layouts
}
