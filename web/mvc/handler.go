package mvc

import (
	servletmultipart "goark.dev/arkarta/servlet/multipart"
	arkweb "goark.dev/arkarta/web"
	goweb "goark.dev/goark/web"
)

// ResultFunc 表示直接返回 Arkarta Web Result 的处理函数。
type ResultFunc func(ctx *arkweb.Context) (arkweb.Result, error)

// ValueFunc 表示返回普通值并由 MVC 写为 JSON 的处理函数。
type ValueFunc[T any] func(ctx *arkweb.Context) (T, error)

// EntityFunc 表示返回 Goark 响应实体的处理函数。
type EntityFunc[T any] func(ctx *arkweb.Context) (goweb.ResponseEntity[T], error)

// BindFunc 表示绑定并校验 JSON 请求体后返回普通值的处理函数。
type BindFunc[In any, Out any] func(ctx *arkweb.Context, input In) (Out, error)

// BindEntityFunc 表示绑定并校验 JSON 请求体后返回 Goark 响应实体的处理函数。
type BindEntityFunc[In any, Out any] func(ctx *arkweb.Context, input In) (goweb.ResponseEntity[Out], error)

// Handler 将 ResultFunc 适配为 Arkarta Web Handler。
func Handler(fn ResultFunc) arkweb.Handler {
	return arkweb.HandlerFunc(func(ctx *arkweb.Context) (arkweb.Result, error) {
		return fn(ctx)
	})
}

// Entity 将响应实体处理函数适配为 Arkarta Web Handler。
func Entity[T any](fn EntityFunc[T]) arkweb.Handler {
	return arkweb.HandlerFunc(func(ctx *arkweb.Context) (arkweb.Result, error) {
		entity, err := fn(ctx)
		if err != nil {
			return nil, err
		}
		return entity, nil
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
	return bindJSON(statusCode, fn, nil)
}

func bindJSON[In any, Out any](statusCode int, fn BindFunc[In, Out], groups []string) arkweb.Handler {
	validationGroups := cloneValidationGroups(groups)
	return arkweb.HandlerFunc(func(ctx *arkweb.Context) (arkweb.Result, error) {
		var input In
		if err := bindAndValidateJSON(ctx, &input, validationGroups); err != nil {
			return nil, err
		}
		value, err := fn(ctx, input)
		if err != nil {
			return nil, err
		}
		return arkweb.JSON(statusCode, value), nil
	})
}

// BindEntity 绑定并校验 JSON 请求体，再写出响应实体。
func BindEntity[In any, Out any](fn BindEntityFunc[In, Out]) arkweb.Handler {
	return bindEntity(fn, nil)
}

func bindEntity[In any, Out any](fn BindEntityFunc[In, Out], groups []string) arkweb.Handler {
	validationGroups := cloneValidationGroups(groups)
	return arkweb.HandlerFunc(func(ctx *arkweb.Context) (arkweb.Result, error) {
		var input In
		if err := bindAndValidateJSON(ctx, &input, validationGroups); err != nil {
			return nil, err
		}
		entity, err := fn(ctx, input)
		if err != nil {
			return nil, err
		}
		return entity, nil
	})
}

// BindMultipart 绑定并校验 multipart/form-data 请求体，再将返回值写为 JSON 响应。
func BindMultipart[In any, Out any](statusCode int, fn BindFunc[In, Out], options ...servletmultipart.Option) arkweb.Handler {
	return bindMultipart(statusCode, fn, nil, options...)
}

func bindMultipart[In any, Out any](statusCode int, fn BindFunc[In, Out], groups []string, options ...servletmultipart.Option) arkweb.Handler {
	validationGroups := cloneValidationGroups(groups)
	return arkweb.HandlerFunc(func(ctx *arkweb.Context) (arkweb.Result, error) {
		input, err := MultipartGroups[In](ctx, validationGroups, options...)
		if err != nil {
			return nil, err
		}
		value, err := fn(ctx, input)
		if err != nil {
			return nil, err
		}
		return arkweb.JSON(statusCode, value), nil
	})
}

// BindMultipartEntity 绑定并校验 multipart/form-data 请求体，再写出响应实体。
func BindMultipartEntity[In any, Out any](fn BindEntityFunc[In, Out], options ...servletmultipart.Option) arkweb.Handler {
	return bindMultipartEntity(fn, nil, options...)
}

func bindMultipartEntity[In any, Out any](fn BindEntityFunc[In, Out], groups []string, options ...servletmultipart.Option) arkweb.Handler {
	validationGroups := cloneValidationGroups(groups)
	return arkweb.HandlerFunc(func(ctx *arkweb.Context) (arkweb.Result, error) {
		input, err := MultipartGroups[In](ctx, validationGroups, options...)
		if err != nil {
			return nil, err
		}
		entity, err := fn(ctx, input)
		if err != nil {
			return nil, err
		}
		return entity, nil
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
