package mvc

import (
	arkweb "goark.dev/arkarta/web"
	"goark.dev/goark/web/message"
)

// OptionalRequestBody 可选绑定请求体；请求体不存在时返回 present=false。
func OptionalRequestBody[T any](ctx *arkweb.Context) (T, bool, error) {
	return OptionalRequestBodyWithMediaTypes[T](ctx)
}

// OptionalRequestBodyWithMediaTypes 按指定媒体类型可选绑定请求体。
func OptionalRequestBodyWithMediaTypes[T any](ctx *arkweb.Context, mediaTypes ...string) (T, bool, error) {
	var out T
	present, err := requestBodyPresent(ctx)
	if err != nil || !present {
		return out, false, err
	}
	if err := message.ReaderFromContext(ctx).Read(ctx, &out, mediaTypes...); err != nil {
		return out, false, err
	}
	return out, true, nil
}

// OptionalValidatedRequestBody 可选绑定请求体；请求体存在时执行结构体验证。
func OptionalValidatedRequestBody[T any](ctx *arkweb.Context, groups ...string) (T, bool, error) {
	return OptionalValidatedRequestBodyWithMediaTypes[T](ctx, nil, groups...)
}

// OptionalValidatedRequestBodyWithMediaTypes 按指定媒体类型可选绑定请求体并执行结构体验证。
func OptionalValidatedRequestBodyWithMediaTypes[T any](ctx *arkweb.Context, mediaTypes []string, groups ...string) (T, bool, error) {
	var out T
	present, err := requestBodyPresent(ctx)
	if err != nil || !present {
		return out, false, err
	}
	if err := bindAndValidateBody(ctx, &out, groups, mediaTypes); err != nil {
		return out, false, err
	}
	return out, true, nil
}

func requestBodyPresent(ctx *arkweb.Context) (bool, error) {
	if ctx == nil || ctx.Request() == nil {
		return false, arkweb.ErrNilContext
	}
	if ctx.Request().Body() == nil {
		return false, nil
	}
	return ctx.Request().ContentLength() != 0, nil
}
