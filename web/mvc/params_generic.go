package mvc

import arkweb "goark.dev/arkarta/web"

// PathValueAs 将路径变量转换为目标类型。
func PathValueAs[T any](ctx *arkweb.Context, name string, options ...ParamOption) (T, error) {
	var zero T
	if ctx == nil {
		return zero, arkweb.ErrNilContext
	}
	value, ok := ctx.Param(name)
	value = stripMatrixSegment(value)
	paramOptions := newParamOptions(ctx, options)
	return resolveConvertedParameter("路径变量", name, value, ok, nil, paramOptions, paramTargetType[T](), convertParamValue[T](paramOptions))
}

// RequestParamAs 将请求参数转换为目标类型，参数视图包含 query 和 urlencoded form。
func RequestParamAs[T any](ctx *arkweb.Context, name string, options ...ParamOption) (T, error) {
	value, ok, err := formValue(ctx, name)
	paramOptions := newParamOptions(ctx, options)
	return resolveConvertedParameter("请求参数", name, value, ok, err, paramOptions, paramTargetType[T](), convertParamValue[T](paramOptions))
}

// RequestHeaderAs 将请求头转换为目标类型。
func RequestHeaderAs[T any](ctx *arkweb.Context, name string, options ...ParamOption) (T, error) {
	value, ok, err := headerValue(ctx, name)
	paramOptions := newParamOptions(ctx, options)
	return resolveConvertedParameter("请求头", name, value, ok, err, paramOptions, paramTargetType[T](), convertParamValue[T](paramOptions))
}

// CookieValueAs 将 Cookie 值转换为目标类型。
func CookieValueAs[T any](ctx *arkweb.Context, name string, options ...ParamOption) (T, error) {
	value, ok, err := cookieValue(ctx, name)
	paramOptions := newParamOptions(ctx, options)
	return resolveConvertedParameter("Cookie", name, value, ok, err, paramOptions, paramTargetType[T](), convertParamValue[T](paramOptions))
}

// MatrixVariableAs 将矩阵变量转换为目标类型。
func MatrixVariableAs[T any](ctx *arkweb.Context, name string, options ...ParamOption) (T, error) {
	value, ok, err := matrixValue(ctx, name)
	paramOptions := newParamOptions(ctx, options)
	return resolveConvertedParameter("矩阵变量", name, value, ok, err, paramOptions, paramTargetType[T](), convertParamValue[T](paramOptions))
}

// RequestAttributeAs 将请求属性转换为目标类型。
func RequestAttributeAs[T any](ctx *arkweb.Context, name string, options ...ParamOption) (T, error) {
	value, ok, err := requestAttributeValue(ctx, name)
	paramOptions := newParamOptions(ctx, options)
	return resolveConvertedParameter("请求属性", name, value, ok, err, paramOptions, paramTargetType[T](), convertParamValue[T](paramOptions))
}

// SessionAttributeAs 将 Session 属性转换为目标类型。
func SessionAttributeAs[T any](ctx *arkweb.Context, name string, options ...ParamOption) (T, error) {
	value, ok, err := sessionAttributeValue(ctx, name)
	paramOptions := newParamOptions(ctx, options)
	return resolveConvertedParameter("Session属性", name, value, ok, err, paramOptions, paramTargetType[T](), convertParamValue[T](paramOptions))
}
