package mvc

import (
	arkweb "goark.dev/arkarta/web"
	"goark.dev/goark/web/message"
)

// BindResultFunc 表示绑定请求体并传入 BindingResult 后返回普通值的处理函数。
type BindResultFunc[In any, Out any] func(ctx *arkweb.Context, input In, result BindingResult) (Out, error)

// BindJSONResult 绑定并验证 JSON 请求体，验证失败时由处理函数读取 BindingResult。
func BindJSONResult[In any, Out any](statusCode int, fn BindResultFunc[In, Out]) arkweb.Handler {
	return bindJSONResult(statusCode, fn, nil)
}

func bindJSONResult[In any, Out any](statusCode int, fn BindResultFunc[In, Out], groups []string) arkweb.Handler {
	validationGroups := cloneValidationGroups(groups)
	return arkweb.HandlerFunc(func(ctx *arkweb.Context) (arkweb.Result, error) {
		var input In
		if err := message.ReaderFromContext(ctx).Read(ctx, &input, message.MediaTypeJSON); err != nil {
			return nil, err
		}
		binding, err := validateBindingResult(ctx, &input, validationGroups)
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
