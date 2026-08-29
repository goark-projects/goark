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

// MultipartResult 绑定 multipart/form-data 请求体，并返回可由调用方处理的绑定和验证结果。
func MultipartResult[T any](ctx *arkweb.Context, options ...servletmultipart.Option) (T, BindingResult, error) {
	return MultipartResultGroups[T](ctx, nil, options...)
}

// MultipartResultGroups 绑定 multipart/form-data 请求体，并按显式分组返回绑定和验证结果。
func MultipartResultGroups[T any](ctx *arkweb.Context, groups []string, options ...servletmultipart.Option) (T, BindingResult, error) {
	var out T
	if ctx == nil {
		return out, BindingResult{}, arkweb.ErrNilContext
	}
	if err := ctx.BindMultipart(&out, options...); err != nil {
		return out, newBindingErrorResult(err), nil
	}
	result, err := validateBindingResult(ctx, &out, groups)
	return out, result, err
}
