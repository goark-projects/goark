package mvc

import (
	"goark.dev/arkarta/servlet/session"
	arkweb "goark.dev/arkarta/web"
	"goark.dev/goark/core/convert"
)

// RequestAttribute 读取请求属性，并转换为目标类型。
func RequestAttribute[T any](ctx *arkweb.Context, name string, options ...ParamOption) (T, error) {
	value, ok, err := rawRequestAttributeValue(ctx, name)
	paramOptions := newParamOptions(ctx, options)
	return resolveAttributeValue[T]("请求属性", name, value, ok, err, paramOptions)
}

// SessionAttribute 读取 Session 属性，并转换为目标类型。
func SessionAttribute[T any](ctx *arkweb.Context, name string, options ...ParamOption) (T, error) {
	value, ok, err := rawSessionAttributeValue(ctx, name)
	paramOptions := newParamOptions(ctx, options)
	return resolveAttributeValue[T]("Session属性", name, value, ok, err, paramOptions)
}

func rawRequestAttributeValue(ctx *arkweb.Context, name string) (any, bool, error) {
	if ctx == nil || ctx.Request() == nil {
		return nil, false, arkweb.ErrNilContext
	}
	value, ok := ctx.Request().Attribute(name)
	return value, ok, nil
}

func rawSessionAttributeValue(ctx *arkweb.Context, name string) (any, bool, error) {
	if ctx == nil || ctx.Request() == nil {
		return nil, false, arkweb.ErrNilContext
	}
	current, ok := session.Current(ctx.Request())
	if !ok {
		return nil, false, nil
	}
	value, ok := current.Attribute(name)
	return value, ok, nil
}

func resolveAttributeValue[T any](kind, name string, value any, ok bool, err error, options paramOptions) (T, error) {
	var zero T
	if err != nil {
		return zero, err
	}
	if !ok {
		if options.hasDefault {
			return convertDefaultAttributeValue[T](name, options)
		}
		if options.required {
			return zero, missingParameterError(kind, name)
		}
		return zero, nil
	}
	converted, err := convert.Convert[T](options.conversionService, value)
	if err != nil {
		return zero, invalidParameterError(name, attributeString(value), paramTargetType[T](), err)
	}
	return converted, nil
}

func convertDefaultAttributeValue[T any](name string, options paramOptions) (T, error) {
	var zero T
	converted, err := convert.Convert[T](options.conversionService, options.defaultValue)
	if err != nil {
		return zero, invalidParameterError(name, options.defaultValue, paramTargetType[T](), err)
	}
	return converted, nil
}
