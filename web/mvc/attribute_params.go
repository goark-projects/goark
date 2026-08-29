package mvc

import (
	"fmt"
	"strconv"

	arkweb "goark.dev/arkarta/web"
)

// RequestAttributeString 绑定字符串请求属性。
func RequestAttributeString(ctx *arkweb.Context, name string, options ...ParamOption) (string, error) {
	value, ok, err := requestAttributeValue(ctx, name)
	return resolveStringParameter("请求属性", name, value, ok, err, newParamOptions(ctx, options))
}

// RequestAttributeInt 绑定 int 请求属性。
func RequestAttributeInt(ctx *arkweb.Context, name string, options ...ParamOption) (int, error) {
	value, ok, err := requestAttributeValue(ctx, name)
	return resolveIntParameter("请求属性", name, value, ok, err, newParamOptions(ctx, options))
}

// RequestAttributeInt64 绑定 int64 请求属性。
func RequestAttributeInt64(ctx *arkweb.Context, name string, options ...ParamOption) (int64, error) {
	value, ok, err := requestAttributeValue(ctx, name)
	return resolveInt64Parameter("请求属性", name, value, ok, err, newParamOptions(ctx, options))
}

// RequestAttributeBool 绑定 bool 请求属性。
func RequestAttributeBool(ctx *arkweb.Context, name string, options ...ParamOption) (bool, error) {
	value, ok, err := requestAttributeValue(ctx, name)
	return resolveBoolParameter("请求属性", name, value, ok, err, newParamOptions(ctx, options))
}

// SessionAttributeString 绑定字符串 Session 属性。
func SessionAttributeString(ctx *arkweb.Context, name string, options ...ParamOption) (string, error) {
	value, ok, err := sessionAttributeValue(ctx, name)
	return resolveStringParameter("Session属性", name, value, ok, err, newParamOptions(ctx, options))
}

// SessionAttributeInt 绑定 int Session 属性。
func SessionAttributeInt(ctx *arkweb.Context, name string, options ...ParamOption) (int, error) {
	value, ok, err := sessionAttributeValue(ctx, name)
	return resolveIntParameter("Session属性", name, value, ok, err, newParamOptions(ctx, options))
}

// SessionAttributeInt64 绑定 int64 Session 属性。
func SessionAttributeInt64(ctx *arkweb.Context, name string, options ...ParamOption) (int64, error) {
	value, ok, err := sessionAttributeValue(ctx, name)
	return resolveInt64Parameter("Session属性", name, value, ok, err, newParamOptions(ctx, options))
}

// SessionAttributeBool 绑定 bool Session 属性。
func SessionAttributeBool(ctx *arkweb.Context, name string, options ...ParamOption) (bool, error) {
	value, ok, err := sessionAttributeValue(ctx, name)
	return resolveBoolParameter("Session属性", name, value, ok, err, newParamOptions(ctx, options))
}

// FlashAttributeString 绑定字符串 Flash 属性。
func FlashAttributeString(ctx *arkweb.Context, name string, options ...ParamOption) (string, error) {
	value, ok, err := flashAttributeValue(ctx, name)
	return resolveStringParameter("Flash属性", name, value, ok, err, newParamOptions(ctx, options))
}

// FlashAttributeInt 绑定 int Flash 属性。
func FlashAttributeInt(ctx *arkweb.Context, name string, options ...ParamOption) (int, error) {
	value, ok, err := flashAttributeValue(ctx, name)
	return resolveIntParameter("Flash属性", name, value, ok, err, newParamOptions(ctx, options))
}

// FlashAttributeInt64 绑定 int64 Flash 属性。
func FlashAttributeInt64(ctx *arkweb.Context, name string, options ...ParamOption) (int64, error) {
	value, ok, err := flashAttributeValue(ctx, name)
	return resolveInt64Parameter("Flash属性", name, value, ok, err, newParamOptions(ctx, options))
}

// FlashAttributeBool 绑定 bool Flash 属性。
func FlashAttributeBool(ctx *arkweb.Context, name string, options ...ParamOption) (bool, error) {
	value, ok, err := flashAttributeValue(ctx, name)
	return resolveBoolParameter("Flash属性", name, value, ok, err, newParamOptions(ctx, options))
}

func requestAttributeValue(ctx *arkweb.Context, name string) (string, bool, error) {
	value, ok, err := rawRequestAttributeValue(ctx, name)
	if err != nil || !ok {
		return "", ok, err
	}
	return attributeString(value), true, nil
}

func sessionAttributeValue(ctx *arkweb.Context, name string) (string, bool, error) {
	value, ok, err := rawSessionAttributeValue(ctx, name)
	if err != nil || !ok {
		return "", ok, err
	}
	return attributeString(value), true, nil
}

func attributeString(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case []byte:
		return string(typed)
	case bool:
		return strconv.FormatBool(typed)
	case int:
		return strconv.Itoa(typed)
	case int8:
		return strconv.FormatInt(int64(typed), 10)
	case int16:
		return strconv.FormatInt(int64(typed), 10)
	case int32:
		return strconv.FormatInt(int64(typed), 10)
	case int64:
		return strconv.FormatInt(typed, 10)
	case uint:
		return strconv.FormatUint(uint64(typed), 10)
	case uint8:
		return strconv.FormatUint(uint64(typed), 10)
	case uint16:
		return strconv.FormatUint(uint64(typed), 10)
	case uint32:
		return strconv.FormatUint(uint64(typed), 10)
	case uint64:
		return strconv.FormatUint(typed, 10)
	case fmt.Stringer:
		return typed.String()
	default:
		return fmt.Sprint(typed)
	}
}
