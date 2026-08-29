package mvc

import (
	"strconv"
	"strings"
	"time"

	arkweb "goark.dev/arkarta/web"
)

// RequestParamStrings 绑定字符串切片请求参数，支持重复参数和逗号分隔值。
func RequestParamStrings(ctx *arkweb.Context, name string, options ...ParamOption) ([]string, error) {
	values, ok, err := formValues(ctx, name)
	return resolveStringSliceParameter("请求参数", name, values, ok, err, newParamOptions(options))
}

// RequestParamInts 绑定 int 切片请求参数。
func RequestParamInts(ctx *arkweb.Context, name string, options ...ParamOption) ([]int, error) {
	values, ok, err := formValues(ctx, name)
	return resolveIntSliceParameter("请求参数", name, values, ok, err, newParamOptions(options))
}

// RequestParamInt64s 绑定 int64 切片请求参数。
func RequestParamInt64s(ctx *arkweb.Context, name string, options ...ParamOption) ([]int64, error) {
	values, ok, err := formValues(ctx, name)
	return resolveInt64SliceParameter("请求参数", name, values, ok, err, newParamOptions(options))
}

// RequestParamBools 绑定 bool 切片请求参数。
func RequestParamBools(ctx *arkweb.Context, name string, options ...ParamOption) ([]bool, error) {
	values, ok, err := formValues(ctx, name)
	return resolveBoolSliceParameter("请求参数", name, values, ok, err, newParamOptions(options))
}

// RequestParamFloat64s 绑定 float64 切片请求参数。
func RequestParamFloat64s(ctx *arkweb.Context, name string, options ...ParamOption) ([]float64, error) {
	values, ok, err := formValues(ctx, name)
	return resolveFloat64SliceParameter("请求参数", name, values, ok, err, newParamOptions(options))
}

// RequestParamTimes 绑定 time.Time 切片请求参数。
func RequestParamTimes(ctx *arkweb.Context, name string, options ...ParamOption) ([]time.Time, error) {
	values, ok, err := formValues(ctx, name)
	return resolveTimeSliceParameter("请求参数", name, values, ok, err, newParamOptions(options))
}

// RequestHeaderStrings 绑定字符串切片请求头，支持重复头和值内逗号分隔。
func RequestHeaderStrings(ctx *arkweb.Context, name string, options ...ParamOption) ([]string, error) {
	values, ok, err := headerValues(ctx, name)
	return resolveStringSliceParameter("请求头", name, values, ok, err, newParamOptions(options))
}

// RequestHeaderInts 绑定 int 切片请求头。
func RequestHeaderInts(ctx *arkweb.Context, name string, options ...ParamOption) ([]int, error) {
	values, ok, err := headerValues(ctx, name)
	return resolveIntSliceParameter("请求头", name, values, ok, err, newParamOptions(options))
}

// RequestHeaderInt64s 绑定 int64 切片请求头。
func RequestHeaderInt64s(ctx *arkweb.Context, name string, options ...ParamOption) ([]int64, error) {
	values, ok, err := headerValues(ctx, name)
	return resolveInt64SliceParameter("请求头", name, values, ok, err, newParamOptions(options))
}

// RequestHeaderBools 绑定 bool 切片请求头。
func RequestHeaderBools(ctx *arkweb.Context, name string, options ...ParamOption) ([]bool, error) {
	values, ok, err := headerValues(ctx, name)
	return resolveBoolSliceParameter("请求头", name, values, ok, err, newParamOptions(options))
}

// RequestHeaderFloat64s 绑定 float64 切片请求头。
func RequestHeaderFloat64s(ctx *arkweb.Context, name string, options ...ParamOption) ([]float64, error) {
	values, ok, err := headerValues(ctx, name)
	return resolveFloat64SliceParameter("请求头", name, values, ok, err, newParamOptions(options))
}

// RequestHeaderTimes 绑定 time.Time 切片请求头。
func RequestHeaderTimes(ctx *arkweb.Context, name string, options ...ParamOption) ([]time.Time, error) {
	values, ok, err := headerValues(ctx, name)
	return resolveTimeSliceParameter("请求头", name, values, ok, err, newParamOptions(options))
}

// CookieValueStrings 绑定字符串切片 Cookie 值，支持逗号分隔值。
func CookieValueStrings(ctx *arkweb.Context, name string, options ...ParamOption) ([]string, error) {
	values, ok, err := cookieValues(ctx, name)
	return resolveStringSliceParameter("Cookie", name, values, ok, err, newParamOptions(options))
}

// CookieValueInts 绑定 int 切片 Cookie 值。
func CookieValueInts(ctx *arkweb.Context, name string, options ...ParamOption) ([]int, error) {
	values, ok, err := cookieValues(ctx, name)
	return resolveIntSliceParameter("Cookie", name, values, ok, err, newParamOptions(options))
}

// CookieValueInt64s 绑定 int64 切片 Cookie 值。
func CookieValueInt64s(ctx *arkweb.Context, name string, options ...ParamOption) ([]int64, error) {
	values, ok, err := cookieValues(ctx, name)
	return resolveInt64SliceParameter("Cookie", name, values, ok, err, newParamOptions(options))
}

// CookieValueBools 绑定 bool 切片 Cookie 值。
func CookieValueBools(ctx *arkweb.Context, name string, options ...ParamOption) ([]bool, error) {
	values, ok, err := cookieValues(ctx, name)
	return resolveBoolSliceParameter("Cookie", name, values, ok, err, newParamOptions(options))
}

// CookieValueFloat64s 绑定 float64 切片 Cookie 值。
func CookieValueFloat64s(ctx *arkweb.Context, name string, options ...ParamOption) ([]float64, error) {
	values, ok, err := cookieValues(ctx, name)
	return resolveFloat64SliceParameter("Cookie", name, values, ok, err, newParamOptions(options))
}

// CookieValueTimes 绑定 time.Time 切片 Cookie 值。
func CookieValueTimes(ctx *arkweb.Context, name string, options ...ParamOption) ([]time.Time, error) {
	values, ok, err := cookieValues(ctx, name)
	return resolveTimeSliceParameter("Cookie", name, values, ok, err, newParamOptions(options))
}

func formValues(ctx *arkweb.Context, name string) ([]string, bool, error) {
	if ctx == nil {
		return nil, false, arkweb.ErrNilContext
	}
	return ctx.FormValues(name)
}

func headerValues(ctx *arkweb.Context, name string) ([]string, bool, error) {
	if ctx == nil || ctx.Request() == nil {
		return nil, false, arkweb.ErrNilContext
	}
	values := ctx.Request().Headers(name)
	if len(values) == 0 {
		return nil, false, nil
	}
	return values, true, nil
}

func cookieValues(ctx *arkweb.Context, name string) ([]string, bool, error) {
	value, ok, err := cookieValue(ctx, name)
	if err != nil || !ok {
		return nil, ok, err
	}
	return []string{value}, true, nil
}

func resolveConvertedParameter[T any](kind, name, value string, ok bool, err error, options paramOptions, targetType string, convert func(string) (T, error)) (T, error) {
	var zero T
	value, ok, err = resolveRawParameter(kind, name, value, ok, err, options)
	if err != nil || !ok {
		return zero, err
	}
	parsed, parseErr := convert(strings.TrimSpace(value))
	if parseErr != nil {
		return zero, invalidParameterError(name, value, targetType, parseErr)
	}
	return parsed, nil
}

func resolveStringSliceParameter(kind, name string, values []string, ok bool, err error, options paramOptions) ([]string, error) {
	values, ok, err = resolveRawValuesParameter(kind, name, values, ok, err, options)
	if err != nil || !ok {
		return nil, err
	}
	return splitParamValues(values), nil
}

func resolveIntSliceParameter(kind, name string, values []string, ok bool, err error, options paramOptions) ([]int, error) {
	return resolveConvertedSliceParameter(kind, name, values, ok, err, options, "[]int", strconv.Atoi)
}

func resolveInt64SliceParameter(kind, name string, values []string, ok bool, err error, options paramOptions) ([]int64, error) {
	return resolveConvertedSliceParameter(kind, name, values, ok, err, options, "[]int64", func(value string) (int64, error) {
		return strconv.ParseInt(value, 10, 64)
	})
}

func resolveBoolSliceParameter(kind, name string, values []string, ok bool, err error, options paramOptions) ([]bool, error) {
	return resolveConvertedSliceParameter(kind, name, values, ok, err, options, "[]bool", strconv.ParseBool)
}

func resolveFloat64SliceParameter(kind, name string, values []string, ok bool, err error, options paramOptions) ([]float64, error) {
	return resolveConvertedSliceParameter(kind, name, values, ok, err, options, "[]float64", func(value string) (float64, error) {
		return strconv.ParseFloat(value, 64)
	})
}

func resolveTimeSliceParameter(kind, name string, values []string, ok bool, err error, options paramOptions) ([]time.Time, error) {
	return resolveConvertedSliceParameter(kind, name, values, ok, err, options, "[]time.Time", func(value string) (time.Time, error) {
		return parseParamTime(value, options)
	})
}

func resolveConvertedSliceParameter[T any](kind, name string, values []string, ok bool, err error, options paramOptions, targetType string, convert func(string) (T, error)) ([]T, error) {
	values, ok, err = resolveRawValuesParameter(kind, name, values, ok, err, options)
	if err != nil || !ok {
		return nil, err
	}
	items := splitParamValues(values)
	out := make([]T, 0, len(items))
	for _, item := range items {
		parsed, parseErr := convert(item)
		if parseErr != nil {
			return nil, invalidParameterError(name, item, targetType, parseErr)
		}
		out = append(out, parsed)
	}
	return out, nil
}

func resolveRawValuesParameter(kind, name string, values []string, ok bool, err error, options paramOptions) ([]string, bool, error) {
	if err != nil {
		return nil, false, err
	}
	if ok {
		return append([]string(nil), values...), true, nil
	}
	if options.hasDefault {
		return []string{options.defaultValue}, true, nil
	}
	if options.required {
		return nil, false, missingParameterError(kind, name)
	}
	return nil, false, nil
}

func splitParamValues(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		for _, item := range strings.Split(value, ",") {
			item = strings.TrimSpace(item)
			if item != "" {
				out = append(out, item)
			}
		}
	}
	return out
}
