package mvc

import (
	arkweb "goark.dev/arkarta/web"
	"goark.dev/goark/web/message"
)

// RequestBody 将请求体按 Content-Type 绑定到目标类型，不执行结构体验证。
func RequestBody[T any](ctx *arkweb.Context) (T, error) {
	var out T
	if err := message.ReaderFromContext(ctx).Read(ctx, &out); err != nil {
		return out, err
	}
	return out, nil
}

// RequestBodyWithMediaTypes 将请求体按指定媒体类型集合绑定到目标类型。
func RequestBodyWithMediaTypes[T any](ctx *arkweb.Context, mediaTypes ...string) (T, error) {
	var out T
	if err := message.ReaderFromContext(ctx).Read(ctx, &out, mediaTypes...); err != nil {
		return out, err
	}
	return out, nil
}

// ValidatedRequestBody 将请求体绑定到目标类型，并按可选分组执行结构体验证。
func ValidatedRequestBody[T any](ctx *arkweb.Context, groups ...string) (T, error) {
	var out T
	if err := bindAndValidateBody(ctx, &out, groups, nil); err != nil {
		return out, err
	}
	return out, nil
}

// ValidatedRequestBodyResult 将请求体绑定到目标类型，并返回可由调用方处理的验证结果。
func ValidatedRequestBodyResult[T any](ctx *arkweb.Context, groups ...string) (T, BindingResult, error) {
	return validatedRequestBodyResult[T](ctx, nil, groups)
}

// ValidatedRequestBodyWithMediaTypes 将请求体按指定媒体类型集合绑定，并执行结构体验证。
func ValidatedRequestBodyWithMediaTypes[T any](ctx *arkweb.Context, mediaTypes []string, groups ...string) (T, error) {
	var out T
	if err := bindAndValidateBody(ctx, &out, groups, mediaTypes); err != nil {
		return out, err
	}
	return out, nil
}

// ValidatedRequestBodyResultWithMediaTypes 将请求体按指定媒体类型集合绑定，并返回验证结果。
func ValidatedRequestBodyResultWithMediaTypes[T any](ctx *arkweb.Context, mediaTypes []string, groups ...string) (T, BindingResult, error) {
	return validatedRequestBodyResult[T](ctx, mediaTypes, groups)
}

func validatedRequestBodyResult[T any](ctx *arkweb.Context, mediaTypes []string, groups []string) (T, BindingResult, error) {
	var out T
	if err := message.ReaderFromContext(ctx).Read(ctx, &out, mediaTypes...); err != nil {
		return out, BindingResult{}, err
	}
	result, err := validateBindingResult(ctx, &out, groups)
	return out, result, err
}
