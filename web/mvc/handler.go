package mvc

import arkweb "goark.dev/arkarta/web"

// ResultFunc 表示直接返回 Arkarta Web Result 的处理函数。
type ResultFunc func(ctx *arkweb.Context) (arkweb.Result, error)

// ValueFunc 表示返回普通值并由 MVC 写为 JSON 的处理函数。
type ValueFunc[T any] func(ctx *arkweb.Context) (T, error)

// BindFunc 表示绑定并校验 JSON 请求体后返回普通值的处理函数。
type BindFunc[In any, Out any] func(ctx *arkweb.Context, input In) (Out, error)

// Handler 将 ResultFunc 适配为 Arkarta Web Handler。
func Handler(fn ResultFunc) arkweb.Handler {
	return arkweb.HandlerFunc(func(ctx *arkweb.Context) (arkweb.Result, error) {
		return fn(ctx)
	})
}

// JSON 将普通返回值写为 JSON 响应。
func JSON[T any](statusCode int, fn ValueFunc[T]) arkweb.Handler {
	return arkweb.HandlerFunc(func(ctx *arkweb.Context) (arkweb.Result, error) {
		value, err := fn(ctx)
		if err != nil {
			return nil, err
		}
		return arkweb.JSON(statusCode, value), nil
	})
}

// BindJSON 绑定并校验 JSON 请求体，再将返回值写为 JSON 响应。
func BindJSON[In any, Out any](statusCode int, fn BindFunc[In, Out]) arkweb.Handler {
	return arkweb.HandlerFunc(func(ctx *arkweb.Context) (arkweb.Result, error) {
		var input In
		if err := ctx.BindAndValidateJSON(&input); err != nil {
			return nil, err
		}
		value, err := fn(ctx, input)
		if err != nil {
			return nil, err
		}
		return arkweb.JSON(statusCode, value), nil
	})
}

// Text 将字符串写为文本响应。
func Text(statusCode int, fn func(ctx *arkweb.Context) (string, error)) arkweb.Handler {
	return arkweb.HandlerFunc(func(ctx *arkweb.Context) (arkweb.Result, error) {
		value, err := fn(ctx)
		if err != nil {
			return nil, err
		}
		return arkweb.Text(statusCode, value), nil
	})
}

// NoContent 执行无响应体处理函数。
func NoContent(fn func(ctx *arkweb.Context) error) arkweb.Handler {
	return arkweb.HandlerFunc(func(ctx *arkweb.Context) (arkweb.Result, error) {
		if err := fn(ctx); err != nil {
			return nil, err
		}
		return arkweb.NoContent(), nil
	})
}
