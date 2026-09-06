package mvc

import (
	"errors"
	"net/url"
	"strings"
	"time"

	gowebfilter "goark.dev/goark/web/filter"

	arkweb "goark.dev/arkarta/web"

	"goark.dev/arkarta/servlet"
	"goark.dev/arkarta/validation"
	"goark.dev/goark/core/convert"
)

func matchBinderFieldWildcard(pattern string, name string) bool {
	patternIndex, nameIndex := 0, 0
	starIndex, matchIndex := -1, 0
	for nameIndex < len(name) {
		if patternIndex < len(pattern) && pattern[patternIndex] == '*' {
			starIndex = patternIndex
			matchIndex = nameIndex
			patternIndex++
			continue
		}
		if patternIndex < len(pattern) && pattern[patternIndex] == name[nameIndex] {
			patternIndex++
			nameIndex++
			continue
		}
		if starIndex >= 0 {
			patternIndex = starIndex + 1
			matchIndex++
			nameIndex = matchIndex
			continue
		}
		return false
	}
	for patternIndex < len(pattern) && pattern[patternIndex] == '*' {
		patternIndex++
	}
	return patternIndex == len(pattern)
}
func matrixSegmentValueLists(segment string) map[string][]string {
	out := make(map[string][]string)
	parts := strings.Split(segment, ";")
	if len(parts) < 2 {
		return out
	}
	for _, part := range parts[1:] {
		name, value, ok := strings.Cut(part, "=")
		if !ok {
			value = ""
		}
		name = pathUnescape(strings.TrimSpace(name))
		if name == "" {
			continue
		}
		out[name] = append(out[name], pathUnescape(strings.TrimSpace(value)))
	}
	return out
}

func appendFormContentValues(values url.Values, req *servlet.Request) url.Values {
	if !shouldAppendFormContent(req) {
		return values
	}
	form, ok := gowebfilter.FormContentValues(req)
	if !ok || len(form) == 0 {
		return values
	}
	if values == nil {
		values = url.Values{}
	}
	for name, list := range form {
		values[name] = append(values[name], list...)
	}
	return values
}

// ConversionServiceFromContext 返回当前请求绑定的转换服务；未绑定时返回默认服务。
func ConversionServiceFromContext(ctx *arkweb.Context) *convert.Service {
	if ctx == nil || ctx.Request() == nil {
		return DefaultConversionService()
	}
	value, ok := ctx.Request().Attribute(AttributeConversionService)
	if !ok {
		return DefaultConversionService()
	}
	service, ok := value.(*convert.Service)
	if !ok || service == nil {
		return DefaultConversionService()
	}
	return service
}
func requestParameterValues(ctx *arkweb.Context, name string) ([]string, bool, error) {
	values, err := requestParameters(ctx)
	if err != nil {
		return nil, false, err
	}
	list, ok := values[name]
	if !ok {
		list, ok = values[emptyArrayRequestParameterName(name)]
	}
	if !ok || len(list) == 0 {
		return nil, false, nil
	}
	return append([]string(nil), list...), true, nil
}

func firstStringValueMap(values map[string][]string) map[string]string {
	if len(values) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(values))
	for name, list := range values {
		if len(list) > 0 {
			out[name] = list[0]
		}
	}
	return out
}

func requestHeaders(ctx *arkweb.Context) (map[string][]string, error) {
	if ctx == nil || ctx.Request() == nil {
		return nil, arkweb.ErrNilContext
	}
	header := ctx.Request().Header()
	values := make(map[string][]string)
	for _, name := range servlet.HeaderNames(header) {
		values[name] = append([]string(nil), header.Values(name)...)
	}
	return values, nil
}

func binderFieldPatternMatches(pattern string, name string) bool {
	if pattern == "*" {
		return true
	}
	if !strings.Contains(pattern, "*") {
		return pattern == name
	}
	return matchBinderFieldWildcard(pattern, name)
}

// FieldError 返回指定字段的第一个验证失败项。
func (r BindingResult) FieldError(path string) (validation.Violation, bool) {
	for _, violation := range r.result.Violations() {
		if violation.Path() == path {
			return violation, true
		}
	}
	return validation.Violation{}, false
}

// MatrixVariableMap 绑定全部矩阵变量，每个名称取第一个值。
func MatrixVariableMap(ctx *arkweb.Context, options ...ParamOption) (map[string]string, error) {
	paramOptions := newParamOptions(ctx, options)
	values, err := matrixValueListsForContext(ctx, paramOptions.matrixPathVariable)
	if err != nil {
		return nil, err
	}
	return firstStringValueMap(values), nil
}

// PathTime 绑定 time.Time 路径变量。
func PathTime(ctx *arkweb.Context, name string, options ...ParamOption) (time.Time, error) {
	if ctx == nil {
		return time.Time{}, arkweb.ErrNilContext
	}
	value, ok := ctx.Param(name)
	value = stripMatrixSegment(value)
	return resolveTimeParameter("路径变量", name, value, ok, nil, newParamOptions(ctx, options))
}
func resolveStringParameter(kind, name, value string, ok bool, err error,
	options paramOptions) (string, error) {
	value, ok, err = resolveRawParameter(kind, name, value, ok, err, options)
	if err != nil || !ok {
		return "", err
	}
	return value, nil
}

// InitializeBinder 执行绑定器初始化函数。
func (f BinderInitializerFunc) InitializeBinder(ctx *arkweb.Context, binder *DataBinder) error {
	if f == nil {
		return ErrNilBinderInitializer
	}
	return f(ctx, binder)
}

// RequestParamValuesMap 绑定全部请求参数，保留每个名称的全部值。
func RequestParamValuesMap(ctx *arkweb.Context) (map[string][]string, error) {
	values, err := requestParameters(ctx)
	if err != nil {
		return nil, err
	}
	return cloneStringValuesMap(values), nil
}
func resolveTimeSliceParameter(kind, name string, values []string, ok bool, err error,
	options paramOptions) ([]time.Time, error) {
	return resolveConvertedSliceParameter(kind, name, values, ok, err, options, "[]time.Time",
		func(value string) (time.Time, error) {
			return parseParamTime(value, options)
		})
}

// SessionAttribute 读取 Session 属性，并转换为目标类型。
func SessionAttribute[T any](ctx *arkweb.Context, name string, options ...ParamOption) (T, error) {
	value, ok, err := rawSessionAttributeValue(ctx, name)
	paramOptions := newParamOptions(ctx, options)
	return resolveAttributeValue[T]("Session属性", name, value, ok, err, paramOptions)
}

// SuppressedFields 返回被字段绑定规则拒绝的字段名快照。
func (r BindingResult) SuppressedFields() []string {
	if len(r.suppressedFields) == 0 {
		return nil
	}
	return append([]string(nil), r.suppressedFields...)
}

// ConversionService 返回当前请求作用域的转换服务。
func (b *DataBinder) ConversionService() *convert.Service {
	if b == nil || b.conversionService == nil {
		return DefaultConversionService()
	}
	return b.conversionService
}

// FlashAttributeFloat64 绑定 float64 Flash 属性。
func FlashAttributeFloat64(ctx *arkweb.Context, name string, options ...ParamOption) (float64,
	error) {
	value, ok, err := flashAttributeValue(ctx, name)
	return resolveFloat64Parameter("Flash属性", name, value, ok, err, newParamOptions(ctx, options))
}

// MatrixVariableStrings 绑定字符串切片矩阵变量，支持逗号分隔值。
func MatrixVariableStrings(ctx *arkweb.Context, name string, options ...ParamOption) ([]string,
	error) {
	paramOptions := newParamOptions(ctx, options)
	values, ok, err := matrixValuesFor(ctx, name, paramOptions.matrixPathVariable)
	return resolveStringSliceParameter("矩阵变量", name, values, ok, err, paramOptions)
}

// FlashAttributeTime 绑定 time.Time Flash 属性。
func FlashAttributeTime(ctx *arkweb.Context, name string, options ...ParamOption) (time.Time,
	error) {
	value, ok, err := flashAttributeValue(ctx, name)
	return resolveTimeParameter("Flash属性", name, value, ok, err, newParamOptions(ctx, options))
}

// WithMatrixPathVariable 指定矩阵变量所属的路径变量段。
func WithMatrixPathVariable(pathVariable string) ParamOption {
	return func(options *paramOptions) {
		options.matrixPathVariable = strings.TrimSpace(pathVariable)
	}
}

// RequestHeaderFloat64 绑定 float64 请求头。
func RequestHeaderFloat64(ctx *arkweb.Context, name string, options ...ParamOption) (float64,
	error) {
	value, ok, err := headerValue(ctx, name)
	return resolveFloat64Parameter("请求头", name, value, ok, err, newParamOptions(ctx, options))
}

// RequestParamStrings 绑定字符串切片请求参数，支持重复参数和逗号分隔值。
func RequestParamStrings(ctx *arkweb.Context, name string, options ...ParamOption) ([]string,
	error) {
	values, ok, err := formValues(ctx, name)
	return resolveStringSliceParameter("请求参数", name, values, ok, err, newParamOptions(ctx, options))
}

// RequestHeaderTimes 绑定 time.Time 切片请求头。
func RequestHeaderTimes(ctx *arkweb.Context, name string, options ...ParamOption) ([]time.Time,
	error) {
	values, ok, err := headerValues(ctx, name)
	return resolveTimeSliceParameter("请求头", name, values, ok, err, newParamOptions(ctx, options))
}
func resolveFloat64SliceParameter(kind, name string, values []string, ok bool, err error,
	options paramOptions) ([]float64, error) {
	return resolveConvertedSliceParameter(kind, name, values, ok, err, options, "[]float64",
		convertParamValue[float64](options))
}

func resolveIntParameter(kind, name, value string, ok bool, err error, options paramOptions) (
	int, error) {
	return resolveConvertedParameter(kind, name, value, ok, err, options, "int",
		convertParamValue[int](options))
}

// RequestAttributeInt 绑定 int 请求属性。
func RequestAttributeInt(ctx *arkweb.Context, name string, options ...ParamOption) (int, error) {
	value, ok, err := requestAttributeValue(ctx, name)
	return resolveIntParameter("请求属性", name, value, ok, err, newParamOptions(ctx, options))
}

// RequestAttributeBool 绑定 bool 请求属性。
func RequestAttributeBool(ctx *arkweb.Context, name string, options ...ParamOption) (bool, error) {
	value, ok, err := requestAttributeValue(ctx, name)
	return resolveBoolParameter("请求属性", name, value, ok, err, newParamOptions(ctx, options))
}

// NewBindingResult 创建绑定结果。
func NewBindingResult(result validation.Result, bindingErrors ...error) BindingResult {
	return BindingResult{result: result, bindingError: errors.Join(bindingErrors...)}
}

func resolveFloat64Parameter(kind, name, value string, ok bool, err error,
	options paramOptions) (float64, error) {
	return resolveConvertedParameter(kind, name, value, ok, err, options, "float64",
		convertParamValue[float64](options))
}

// RequestParamInts 绑定 int 切片请求参数。
func RequestParamInts(ctx *arkweb.Context, name string, options ...ParamOption) ([]int, error) {
	values, ok, err := formValues(ctx, name)
	return resolveIntSliceParameter("请求参数", name, values, ok, err, newParamOptions(ctx, options))
}

// CookieValueInt64s 绑定 int64 切片 Cookie 值。
func CookieValueInt64s(ctx *arkweb.Context, name string, options ...ParamOption) ([]int64, error) {
	values, ok, err := cookieValues(ctx, name)
	return resolveInt64SliceParameter("Cookie", name, values, ok, err, newParamOptions(ctx, options))
}

// RequestParamInt64 绑定 int64 请求参数，参数视图包含 query 和 urlencoded form。
func RequestParamInt64(ctx *arkweb.Context, name string, options ...ParamOption) (int64, error) {
	value, ok, err := formValue(ctx, name)
	return resolveInt64Parameter("请求参数", name, value, ok, err, newParamOptions(ctx, options))
}

// CookieValueString 绑定字符串 Cookie 值。
func CookieValueString(ctx *arkweb.Context, name string, options ...ParamOption) (string, error) {
	value, ok, err := cookieValue(ctx, name)
	return resolveStringParameter("Cookie", name, value, ok, err, newParamOptions(ctx, options))
}

// PathVariableBool 绑定 bool 路径变量。
func PathVariableBool(ctx *arkweb.Context, name string, options ...ParamOption) (bool, error) {
	return PathBool(ctx, name, options...)
}

// PathVariableTimes 绑定 time.Time 切片路径变量。
func PathVariableTimes(ctx *arkweb.Context, name string, options ...ParamOption) ([]time.Time,
	error) {
	return PathTimes(ctx, name, options...)
}
func formValues(ctx *arkweb.Context, name string) ([]string, bool, error) {
	return requestParameterValues(ctx, name)
}

// ParamOption 定制 MVC 参数绑定行为。
type ParamOption func(*paramOptions)

var defaultMVCConversionService = convert.DefaultService()
