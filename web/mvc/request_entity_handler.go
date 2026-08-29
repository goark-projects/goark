package mvc

import (
	arkweb "goark.dev/arkarta/web"
	goweb "goark.dev/goark/web"
)

// BindRequestEntityFunc 表示绑定请求实体后返回普通值的处理函数。
type BindRequestEntityFunc[In any, Out any] func(ctx *arkweb.Context, input goweb.RequestEntity[In]) (Out, error)

// BindRequestEntityResponseFunc 表示绑定请求实体后返回响应实体的处理函数。
type BindRequestEntityResponseFunc[In any, Out any] func(ctx *arkweb.Context, input goweb.RequestEntity[In]) (goweb.ResponseEntity[Out], error)

// BindRequestEntity 绑定并校验请求实体，再将返回值写为 JSON 响应。
func BindRequestEntity[In any, Out any](statusCode int, fn BindRequestEntityFunc[In, Out]) arkweb.Handler {
	return bindRequestEntity(statusCode, fn, nil, nil)
}

// BindRequestEntityWithMediaTypes 按指定媒体类型集合绑定请求实体，再将返回值写为 JSON 响应。
func BindRequestEntityWithMediaTypes[In any, Out any](statusCode int, fn BindRequestEntityFunc[In, Out], mediaTypes ...string) arkweb.Handler {
	return bindRequestEntity(statusCode, fn, nil, mediaTypes)
}

func bindRequestEntity[In any, Out any](statusCode int, fn BindRequestEntityFunc[In, Out], groups []string, mediaTypes []string) arkweb.Handler {
	validationGroups := cloneValidationGroups(groups)
	readMediaTypes := cleanRouteValues(mediaTypes)
	return arkweb.HandlerFunc(func(ctx *arkweb.Context) (arkweb.Result, error) {
		input, err := requestEntity[In](ctx, validationGroups, readMediaTypes)
		if err != nil {
			return nil, err
		}
		value, err := fn(ctx, input)
		if err != nil {
			return nil, err
		}
		return jsonResult(ctx, statusCode, value), nil
	})
}

// BindRequestEntityEntity 绑定并校验请求实体，再写出响应实体。
func BindRequestEntityEntity[In any, Out any](fn BindRequestEntityResponseFunc[In, Out]) arkweb.Handler {
	return bindRequestEntityEntity(fn, nil, nil)
}

// BindRequestEntityEntityWithMediaTypes 按指定媒体类型集合绑定请求实体，再写出响应实体。
func BindRequestEntityEntityWithMediaTypes[In any, Out any](fn BindRequestEntityResponseFunc[In, Out], mediaTypes ...string) arkweb.Handler {
	return bindRequestEntityEntity(fn, nil, mediaTypes)
}

func bindRequestEntityEntity[In any, Out any](fn BindRequestEntityResponseFunc[In, Out], groups []string, mediaTypes []string) arkweb.Handler {
	validationGroups := cloneValidationGroups(groups)
	readMediaTypes := cleanRouteValues(mediaTypes)
	return arkweb.HandlerFunc(func(ctx *arkweb.Context) (arkweb.Result, error) {
		input, err := requestEntity[In](ctx, validationGroups, readMediaTypes)
		if err != nil {
			return nil, err
		}
		entity, err := fn(ctx, input)
		if err != nil {
			return nil, err
		}
		return entityResult(ctx, entity), nil
	})
}
