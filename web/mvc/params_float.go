package mvc

import (
	"strconv"

	arkweb "goark.dev/arkarta/web"
)

// PathFloat64 绑定 float64 路径变量。
func PathFloat64(ctx *arkweb.Context, name string, options ...ParamOption) (float64, error) {
	if ctx == nil {
		return 0, arkweb.ErrNilContext
	}
	value, ok := ctx.Param(name)
	value = stripMatrixSegment(value)
	return resolveFloat64Parameter("路径变量", name, value, ok, nil, newParamOptions(options))
}

// RequestParamFloat64 绑定 float64 请求参数，参数视图包含 query 和 urlencoded form。
func RequestParamFloat64(ctx *arkweb.Context, name string, options ...ParamOption) (float64, error) {
	value, ok, err := formValue(ctx, name)
	return resolveFloat64Parameter("请求参数", name, value, ok, err, newParamOptions(options))
}

// RequestHeaderFloat64 绑定 float64 请求头。
func RequestHeaderFloat64(ctx *arkweb.Context, name string, options ...ParamOption) (float64, error) {
	value, ok, err := headerValue(ctx, name)
	return resolveFloat64Parameter("请求头", name, value, ok, err, newParamOptions(options))
}

// CookieValueFloat64 绑定 float64 Cookie 值。
func CookieValueFloat64(ctx *arkweb.Context, name string, options ...ParamOption) (float64, error) {
	value, ok, err := cookieValue(ctx, name)
	return resolveFloat64Parameter("Cookie", name, value, ok, err, newParamOptions(options))
}

// MatrixVariableFloat64 绑定 float64 矩阵变量。
func MatrixVariableFloat64(ctx *arkweb.Context, name string, options ...ParamOption) (float64, error) {
	value, ok, err := matrixValue(ctx, name)
	return resolveFloat64Parameter("矩阵变量", name, value, ok, err, newParamOptions(options))
}

// RequestAttributeFloat64 绑定 float64 请求属性。
func RequestAttributeFloat64(ctx *arkweb.Context, name string, options ...ParamOption) (float64, error) {
	value, ok, err := requestAttributeValue(ctx, name)
	return resolveFloat64Parameter("请求属性", name, value, ok, err, newParamOptions(options))
}

// SessionAttributeFloat64 绑定 float64 Session 属性。
func SessionAttributeFloat64(ctx *arkweb.Context, name string, options ...ParamOption) (float64, error) {
	value, ok, err := sessionAttributeValue(ctx, name)
	return resolveFloat64Parameter("Session属性", name, value, ok, err, newParamOptions(options))
}

func resolveFloat64Parameter(kind, name, value string, ok bool, err error, options paramOptions) (float64, error) {
	return resolveConvertedParameter(kind, name, value, ok, err, options, "float64", func(value string) (float64, error) {
		return strconv.ParseFloat(value, 64)
	})
}
