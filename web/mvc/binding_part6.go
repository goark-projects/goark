package mvc

import (
	"net/url"
	"strings"
	"time"

	servletmultipart "goark.dev/arkarta/servlet/multipart"

	arkweb "goark.dev/arkarta/web"

	"goark.dev/arkarta/validation"
	"goark.dev/goark/core/convert"
)

func resolveAttributeValue[T any](kind, name string, value any, ok bool, err error,
	options paramOptions) (T, error) {
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

func requestCookieValues(ctx *arkweb.Context) (map[string][]string, error) {
	if ctx == nil || ctx.Request() == nil {
		return nil, arkweb.ErrNilContext
	}
	cookies := ctx.Request().Cookies()
	if len(cookies) == 0 {
		return map[string][]string{}, nil
	}
	out := make(map[string][]string, len(cookies))
	for _, cookie := range cookies {
		if cookie == nil {
			continue
		}
		name := strings.TrimSpace(cookie.Name)
		if name == "" {
			continue
		}
		out[name] = append(out[name], cookie.Value)
	}
	return out, nil
}

func (b *DataBinder) filterFieldMarkers(markers map[string]struct{}, suppressed []string) (
	map[string]struct{}, []string) {
	if len(markers) == 0 || !b.hasFieldRules() {
		return markers, suppressed
	}
	out := make(map[string]struct{}, len(markers))
	for name := range markers {
		if name != "" && b.isFieldAllowed(name) {
			out[name] = struct{}{}
			continue
		}
		suppressed = append(suppressed, name)
	}
	if len(out) == 0 {
		return nil, suppressed
	}
	return out, suppressed
}

func resolveRawParameter(kind, name, value string, ok bool, err error, options paramOptions) (
	string, bool, error) {
	if err != nil {
		return "", false, err
	}
	if ok {
		return value, true, nil
	}
	if options.hasDefault {
		return options.defaultValue, true, nil
	}
	if options.required {
		return "", false, missingParameterError(kind, name)
	}
	return "", false, nil
}

func resolveConvertedParameter[T any](kind, name, value string, ok bool, err error,
	options paramOptions, targetType string, convert func(string) (T, error)) (T, error) {
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

// MultipartResultGroups 绑定 multipart/form-data 请求体，并按显式分组返回绑定和验证结果。
func MultipartResultGroups[T any](ctx *arkweb.Context, groups []string,
	options ...servletmultipart.Option) (T, BindingResult, error) {
	var out T
	if ctx == nil {
		return out, BindingResult{}, arkweb.ErrNilContext
	}
	if err := ctx.BindMultipart(&out, options...); err != nil {
		return out, newBindingErrorResult(err), nil
	}
	result, err := validateBindingResult(ctx, &out, groups)
	return out, result, err
}

// PathValueAs 将路径变量转换为目标类型。
func PathValueAs[T any](ctx *arkweb.Context, name string, options ...ParamOption) (T, error) {
	var zero T
	if ctx == nil {
		return zero, arkweb.ErrNilContext
	}
	value, ok := ctx.Param(name)
	value = stripMatrixSegment(value)
	paramOptions := newParamOptions(ctx, options)
	return resolveConvertedParameter("路径变量", name, value, ok, nil, paramOptions,
		paramTargetType[T](), convertParamValue[T](paramOptions))
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

func matchesBinderFieldPattern(patterns []string, name string) bool {
	for _, pattern := range patterns {
		if binderFieldPatternMatches(pattern, name) {
			return true
		}
	}
	return false
}

// CookieValueValuesMap 绑定全部 Cookie，保留每个名称的全部值。
func CookieValueValuesMap(ctx *arkweb.Context) (map[string][]string, error) {
	values, err := requestCookieValues(ctx)
	if err != nil {
		return nil, err
	}
	return cloneStringValuesMap(values), nil
}

// PathInt 绑定 int 路径变量。
func PathInt(ctx *arkweb.Context, name string, options ...ParamOption) (int, error) {
	if ctx == nil {
		return 0, arkweb.ErrNilContext
	}
	value, ok := ctx.Param(name)
	value = stripMatrixSegment(value)
	return resolveIntParameter("路径变量", name, value, ok, nil, newParamOptions(ctx, options))
}
func rawRequestAttributeValue(ctx *arkweb.Context, name string) (any, bool, error) {
	if ctx == nil || ctx.Request() == nil {
		return nil, false, arkweb.ErrNilContext
	}
	value, ok := ctx.Request().Attribute(name)
	return value, ok, nil
}

// SetFieldMarkerPrefix 设置字段 marker 前缀；空字符串表示禁用 marker 处理。
func (b *DataBinder) SetFieldMarkerPrefix(prefix string) error {
	if b == nil {
		return ErrNilDataBinder
	}
	b.fieldMarkerPrefix = strings.TrimSpace(prefix)
	return nil
}

// RequestAttributeAs 将请求属性转换为目标类型。
func RequestAttributeAs[T any](ctx *arkweb.Context, name string, options ...ParamOption) (T,
	error) {
	value, ok, err := requestAttributeValue(ctx, name)
	paramOptions := newParamOptions(ctx, options)
	return resolveConvertedParameter("请求属性", name, value, ok, err, paramOptions,
		paramTargetType[T](), convertParamValue[T](paramOptions))
}

// CookieValueMap 绑定全部 Cookie，每个名称取第一个值。
func CookieValueMap(ctx *arkweb.Context) (map[string]string, error) {
	values, err := requestCookieValues(ctx)
	if err != nil {
		return nil, err
	}
	return firstStringValueMap(values), nil
}

// WithDefaultValue 设置参数缺失时使用的默认文本值。
func WithDefaultValue(value string) ParamOption {
	return func(options *paramOptions) {
		options.required = false
		options.hasDefault = true
		options.defaultValue = value
	}
}

// FieldMarkerPrefix 返回字段 marker 前缀。
func (b *DataBinder) FieldMarkerPrefix() string {
	if b == nil {
		return ""
	}
	return b.fieldMarkerPrefix
}
func stripMatrixSegment(value string) string {
	if base, _, ok := strings.Cut(value, ";"); ok {
		return base
	}
	return value
}

// CookieValueAs 将 Cookie 值转换为目标类型。
func CookieValueAs[T any](ctx *arkweb.Context, name string, options ...ParamOption) (T, error) {
	value, ok, err := cookieValue(ctx, name)
	paramOptions := newParamOptions(ctx, options)
	return resolveConvertedParameter("Cookie", name, value, ok, err, paramOptions,
		paramTargetType[T](), convertParamValue[T](paramOptions))
}

// CookieValueTimes 绑定 time.Time 切片 Cookie 值。
func CookieValueTimes(ctx *arkweb.Context, name string, options ...ParamOption) ([]time.Time,
	error) {
	values, ok, err := cookieValues(ctx, name)
	return resolveTimeSliceParameter("Cookie", name, values, ok, err, newParamOptions(ctx, options))
}

// SessionAttributeString 绑定字符串 Session 属性。
func SessionAttributeString(ctx *arkweb.Context, name string, options ...ParamOption) (string,
	error) {
	value, ok, err := sessionAttributeValue(ctx, name)
	return resolveStringParameter("Session属性", name, value, ok, err, newParamOptions(ctx, options))
}

// BindingResult 表示绑定成功后的结构体验证结果。
type BindingResult struct {
	result           validation.Result
	bindingError     error
	suppressedFields []string
}

// MatrixVariableBool 绑定 bool 矩阵变量。
func MatrixVariableBool(ctx *arkweb.Context, name string, options ...ParamOption) (bool, error) {
	paramOptions := newParamOptions(ctx, options)
	value, ok, err := matrixValue(ctx, name, paramOptions.matrixPathVariable)
	return resolveBoolParameter("矩阵变量", name, value, ok, err, paramOptions)
}

// MatrixVariableInts 绑定 int 切片矩阵变量。
func MatrixVariableInts(ctx *arkweb.Context, name string, options ...ParamOption) ([]int, error) {
	paramOptions := newParamOptions(ctx, options)
	values, ok, err := matrixValuesFor(ctx, name, paramOptions.matrixPathVariable)
	return resolveIntSliceParameter("矩阵变量", name, values, ok, err, paramOptions)
}

// RequestHeaderStrings 绑定字符串切片请求头，支持重复头和值内逗号分隔。
func RequestHeaderStrings(ctx *arkweb.Context, name string, options ...ParamOption) ([]string,
	error) {
	values, ok, err := headerValues(ctx, name)
	return resolveStringSliceParameter("请求头", name, values, ok, err, newParamOptions(ctx, options))
}
func resolveIntSliceParameter(kind, name string, values []string, ok bool, err error,
	options paramOptions) ([]int, error) {
	return resolveConvertedSliceParameter(kind, name, values, ok, err, options, "[]int",
		convertParamValue[int](options))
}

// SessionAttributeTime 绑定 time.Time Session 属性。
func SessionAttributeTime(ctx *arkweb.Context, name string, options ...ParamOption) (time.Time,
	error) {
	value, ok, err := sessionAttributeValue(ctx, name)
	return resolveTimeParameter("Session属性", name, value, ok, err, newParamOptions(ctx, options))
}

// PathVariableFloat64 绑定 float64 路径变量。
func PathVariableFloat64(ctx *arkweb.Context, name string, options ...ParamOption) (float64,
	error) {
	return PathFloat64(ctx, name, options...)
}

// FlashAttributeInt 绑定 int Flash 属性。
func FlashAttributeInt(ctx *arkweb.Context, name string, options ...ParamOption) (int, error) {
	value, ok, err := flashAttributeValue(ctx, name)
	return resolveIntParameter("Flash属性", name, value, ok, err, newParamOptions(ctx, options))
}

type preparedModelAttributeValues struct {
	values           url.Values
	fieldMarkers     map[string]struct{}
	suppressedFields []string
}

// PathInt64s 绑定 int64 切片路径变量。
func PathInt64s(ctx *arkweb.Context, name string, options ...ParamOption) ([]int64, error) {
	values, ok, err := pathValues(ctx, name)
	return resolveInt64SliceParameter("路径变量", name, values, ok, err, newParamOptions(ctx, options))
}

// RequestHeaderInts 绑定 int 切片请求头。
func RequestHeaderInts(ctx *arkweb.Context, name string, options ...ParamOption) ([]int, error) {
	values, ok, err := headerValues(ctx, name)
	return resolveIntSliceParameter("请求头", name, values, ok, err, newParamOptions(ctx, options))
}

// CookieValueTime 绑定 time.Time Cookie 值。
func CookieValueTime(ctx *arkweb.Context, name string, options ...ParamOption) (time.Time, error) {
	value, ok, err := cookieValue(ctx, name)
	return resolveTimeParameter("Cookie", name, value, ok, err, newParamOptions(ctx, options))
}

// RequestHeaderInt 绑定 int 请求头。
func RequestHeaderInt(ctx *arkweb.Context, name string, options ...ParamOption) (int, error) {
	value, ok, err := headerValue(ctx, name)
	return resolveIntParameter("请求头", name, value, ok, err, newParamOptions(ctx, options))
}

// PathVariableString 绑定字符串路径变量，对齐 Spring @PathVariable 命名。
func PathVariableString(ctx *arkweb.Context, name string, options ...ParamOption) (string, error) {
	return PathString(ctx, name, options...)
}

// PathVariableInts 绑定 int 切片路径变量。
func PathVariableInts(ctx *arkweb.Context, name string, options ...ParamOption) ([]int, error) {
	return PathInts(ctx, name, options...)
}

// Result 返回底层 Arkarta 验证结果。
func (r BindingResult) Result() validation.Result {
	return r.result
}
func (b *DataBinder) hasFieldRules() bool {
	return b != nil && (len(b.allowedFields) > 0 || len(b.disallowedFields) > 0)
}

// BindResultFunc 表示绑定请求体并传入 BindingResult 后返回普通值的处理函数。
type BindResultFunc[In any, Out any] func(ctx *arkweb.Context, input In,
	result BindingResult) (Out, error)

// Converter 描述 MVC 参数绑定可使用的类型转换器。
type Converter = convert.Converter
