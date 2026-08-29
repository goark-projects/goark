package mvc

import (
	servletmultipart "goark.dev/arkarta/servlet/multipart"
	arkweb "goark.dev/arkarta/web"
)

// Multipart 绑定 multipart/form-data 请求体并执行结构体验证。
func Multipart[T any](ctx *arkweb.Context, options ...servletmultipart.Option) (T, error) {
	return MultipartGroups[T](ctx, nil, options...)
}

// MultipartGroups 绑定 multipart/form-data 请求体并按显式分组执行结构体验证。
func MultipartGroups[T any](ctx *arkweb.Context, groups []string, options ...servletmultipart.Option) (T, error) {
	var out T
	if ctx == nil {
		return out, arkweb.ErrNilContext
	}
	if err := ctx.BindMultipart(&out, options...); err != nil {
		return out, err
	}
	return out, validateBound(ctx, &out, groups)
}
