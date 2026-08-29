package mvc

import (
	servletmultipart "goark.dev/arkarta/servlet/multipart"
	arkweb "goark.dev/arkarta/web"
)

// BindMultipartResult 绑定 multipart/form-data 请求体，验证失败时由处理函数读取 BindingResult。
func BindMultipartResult[In any, Out any](statusCode int, fn BindResultFunc[In, Out], options ...servletmultipart.Option) arkweb.Handler {
	return bindMultipartResult(statusCode, fn, nil, options...)
}

func bindMultipartResult[In any, Out any](statusCode int, fn BindResultFunc[In, Out], groups []string, options ...servletmultipart.Option) arkweb.Handler {
	validationGroups := cloneValidationGroups(groups)
	multipartOptions := append([]servletmultipart.Option(nil), options...)
	return arkweb.HandlerFunc(func(ctx *arkweb.Context) (arkweb.Result, error) {
		input, binding, err := MultipartResultGroups[In](ctx, validationGroups, multipartOptions...)
		if err != nil {
			return nil, err
		}
		value, err := fn(ctx, input, binding)
		if err != nil {
			return nil, err
		}
		return jsonResult(ctx, statusCode, value), nil
	})
}
