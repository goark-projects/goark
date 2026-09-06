// Package mvc 提供 Goark Web MVC 的显式路由装配、请求参数绑定、聚合表单绑定和异常映射能力。
//
// 第一版采用描述符和生成代码友好的注册模型，不做 Java 风格运行期扫描。
// RequestMapping 提供对齐 Spring MVC 的无 method 限定路由展开能力。
// Controller 级 method 和条件用于表达类型级 RequestMapping 的条件继承语义。
package mvc

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	servletmultipart "goark.dev/arkarta/servlet/multipart"

	arkweb "goark.dev/arkarta/web"

	"goark.dev/arkarta/servlet/session"
	"goark.dev/arkarta/validation"
	"goark.dev/goark/core/convert"
)

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
func (b *DataBinder) filterModelAttributeValues(values url.Values) (url.Values, []string) {
	if b == nil || len(values) == 0 || !b.hasFieldRules() {
		return values, nil
	}
	out := make(url.Values, len(values))
	suppressed := make([]string, 0)
	for name, list := range values {
		if name == "" || len(list) == 0 {
			continue
		}
		if !b.isFieldAllowed(name) {
			suppressed = append(suppressed, name)
			continue
		}
		out[name] = append([]string(nil), list...)
	}
	return out, suppressed
}

func resolveRawValuesParameter(kind, name string, values []string, ok bool, err error,
	options paramOptions) ([]string, bool, error) {
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

func dataBinderFromContext(ctx *arkweb.Context) (*DataBinder, bool) {
	if ctx == nil || ctx.Request() == nil {
		return nil, false
	}
	value, ok := ctx.Request().Attribute(attributeDataBinder)
	if !ok {
		return nil, false
	}
	binder, ok := value.(*DataBinder)
	if !ok || binder == nil {
		return nil, false
	}
	return binder, true
}

// ConversionInterceptor 在请求进入 MVC 处理前绑定转换服务。
func ConversionInterceptor(service *convert.Service) arkweb.Interceptor {
	if service == nil {
		service = DefaultConversionService()
	}
	return arkweb.InterceptorFunc(func(ctx *arkweb.Context, next arkweb.Handler) (arkweb.Result,
		error) {
		if ctx != nil && ctx.Request() != nil {
			ctx.Request().SetAttribute(AttributeConversionService, service)
		}
		return next.Handle(ctx)
	})
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

func requestParameters(ctx *arkweb.Context) (url.Values, error) {
	if ctx == nil || ctx.Request() == nil {
		return nil, arkweb.ErrNilContext
	}
	values, err := ctx.Request().Parameters()
	if err != nil {
		return nil, err
	}
	return appendFormContentValues(values, ctx.Request()), nil
}

func cloneStringValuesMap(values map[string][]string) map[string][]string {
	if len(values) == 0 {
		return map[string][]string{}
	}
	out := make(map[string][]string, len(values))
	for name, list := range values {
		out[name] = append([]string(nil), list...)
	}
	return out
}

// DataBinder 封装控制器本地的数据绑定扩展点。
type DataBinder struct {
	conversionService  *convert.Service
	allowedFields      []string
	disallowedFields   []string
	fieldMarkerPrefix  string
	fieldDefaultPrefix string
}

func resolveStringSliceParameter(kind, name string, values []string, ok bool, err error,
	options paramOptions) ([]string, error) {
	values, ok, err = resolveRawValuesParameter(kind, name, values, ok, err, options)
	if err != nil || !ok {
		return nil, err
	}
	return splitParamValues(values), nil
}

// PathInt64 绑定 int64 路径变量。
func PathInt64(ctx *arkweb.Context, name string, options ...ParamOption) (int64, error) {
	if ctx == nil {
		return 0, arkweb.ErrNilContext
	}
	value, ok := ctx.Param(name)
	value = stripMatrixSegment(value)
	return resolveInt64Parameter("路径变量", name, value, ok, nil, newParamOptions(ctx, options))
}
func convertDefaultAttributeValue[T any](name string, options paramOptions) (T, error) {
	var zero T
	converted, err := convert.Convert[T](options.conversionService, options.defaultValue)
	if err != nil {
		return zero, invalidParameterError(name, options.defaultValue, paramTargetType[T](), err)
	}
	return converted, nil
}

// SetFieldDefaultPrefix 设置字段默认值前缀；空字符串表示禁用默认值处理。
func (b *DataBinder) SetFieldDefaultPrefix(prefix string) error {
	if b == nil {
		return ErrNilDataBinder
	}
	b.fieldDefaultPrefix = strings.TrimSpace(prefix)
	return nil
}

// SessionAttributeAs 将 Session 属性转换为目标类型。
func SessionAttributeAs[T any](ctx *arkweb.Context, name string, options ...ParamOption) (T,
	error) {
	value, ok, err := sessionAttributeValue(ctx, name)
	paramOptions := newParamOptions(ctx, options)
	return resolveConvertedParameter("Session属性", name, value, ok, err, paramOptions,
		paramTargetType[T](), convertParamValue[T](paramOptions))
}

// MatrixVariableTimes 绑定 time.Time 切片矩阵变量。
func MatrixVariableTimes(ctx *arkweb.Context, name string, options ...ParamOption) (
	[]time.Time, error) {
	paramOptions := newParamOptions(ctx, options)
	values, ok, err := matrixValuesFor(ctx, name, paramOptions.matrixPathVariable)
	return resolveTimeSliceParameter("矩阵变量", name, values, ok, err, paramOptions)
}

func headerValue(ctx *arkweb.Context, name string) (string, bool, error) {
	if ctx == nil || ctx.Request() == nil {
		return "", false, arkweb.ErrNilContext
	}
	value, ok := ctx.Request().HeaderValue(name)
	return value, ok, nil
}

func ensureClonedModelAttributeValues(out url.Values, values url.Values) url.Values {
	if out != nil {
		return out
	}
	return url.Values(cloneStringValuesMap(values))
}

func pathUnescape(value string) string {
	out, err := url.PathUnescape(value)
	if err != nil {
		return value
	}
	return out
}

// MatrixVariableAs 将矩阵变量转换为目标类型。
func MatrixVariableAs[T any](ctx *arkweb.Context, name string, options ...ParamOption) (T, error) {
	paramOptions := newParamOptions(ctx, options)
	value, ok, err := matrixValue(ctx, name, paramOptions.matrixPathVariable)
	return resolveConvertedParameter("矩阵变量", name, value, ok, err, paramOptions,
		paramTargetType[T](), convertParamValue[T](paramOptions))
}

var defaultParamTimeLayouts = []string{
	time.RFC3339Nano,
	time.RFC3339,
	http.TimeFormat,
	"2006-01-02",
	"2006-01-02 15:04:05",
}

// SessionAttributeInt64 绑定 int64 Session 属性。
func SessionAttributeInt64(ctx *arkweb.Context, name string, options ...ParamOption) (int64,
	error) {
	value, ok, err := sessionAttributeValue(ctx, name)
	return resolveInt64Parameter("Session属性", name, value, ok, err, newParamOptions(ctx, options))
}
func convertParamValue[T any](options paramOptions) func(string) (T, error) {
	return func(value string) (T, error) {
		return convert.Convert[T](options.conversionService, value)
	}
}

// BindMultipartResult 绑定 multipart/form-data 请求体，验证失败时由处理函数读取 BindingResult。
func BindMultipartResult[In any, Out any](statusCode int, fn BindResultFunc[In, Out],
	options ...servletmultipart.Option) arkweb.Handler {
	return bindMultipartResult(statusCode, fn, nil, options...)
}

// MatrixVariableBools 绑定 bool 切片矩阵变量。
func MatrixVariableBools(ctx *arkweb.Context, name string, options ...ParamOption) ([]bool, error) {
	paramOptions := newParamOptions(ctx, options)
	values, ok, err := matrixValuesFor(ctx, name, paramOptions.matrixPathVariable)
	return resolveBoolSliceParameter("矩阵变量", name, values, ok, err, paramOptions)
}

// RequestHeaderInt64s 绑定 int64 切片请求头。
func RequestHeaderInt64s(ctx *arkweb.Context, name string, options ...ParamOption) ([]int64,
	error) {
	values, ok, err := headerValues(ctx, name)
	return resolveInt64SliceParameter("请求头", name, values, ok, err, newParamOptions(ctx, options))
}
func resolveInt64SliceParameter(kind, name string, values []string, ok bool, err error,
	options paramOptions) ([]int64, error) {
	return resolveConvertedSliceParameter(kind, name, values, ok, err, options, "[]int64",
		convertParamValue[int64](options))
}

// WithRequired 设置参数是否必须存在。
func WithRequired(required bool) ParamOption {
	return func(options *paramOptions) {
		options.required = required
	}
}

// PathVariableStrings 绑定字符串切片路径变量。
func PathVariableStrings(ctx *arkweb.Context, name string, options ...ParamOption) ([]string,
	error) {
	return PathStrings(ctx, name, options...)
}

// FlashAttributeInt64 绑定 int64 Flash 属性。
func FlashAttributeInt64(ctx *arkweb.Context, name string, options ...ParamOption) (int64, error) {
	value, ok, err := flashAttributeValue(ctx, name)
	return resolveInt64Parameter("Flash属性", name, value, ok, err, newParamOptions(ctx, options))
}

// MultipartResult 绑定 multipart/form-data 请求体，并返回可由调用方处理的绑定和验证结果。
func MultipartResult[T any](ctx *arkweb.Context, options ...servletmultipart.Option) (T,
	BindingResult, error) {
	return MultipartResultGroups[T](ctx, nil, options...)
}

// PathBools 绑定 bool 切片路径变量。
func PathBools(ctx *arkweb.Context, name string, options ...ParamOption) ([]bool, error) {
	values, ok, err := pathValues(ctx, name)
	return resolveBoolSliceParameter("路径变量", name, values, ok, err, newParamOptions(ctx, options))
}

// RequestHeaderBools 绑定 bool 切片请求头。
func RequestHeaderBools(ctx *arkweb.Context, name string, options ...ParamOption) ([]bool, error) {
	values, ok, err := headerValues(ctx, name)
	return resolveBoolSliceParameter("请求头", name, values, ok, err, newParamOptions(ctx, options))
}

// RequestParamString 绑定字符串请求参数，参数视图包含 query 和 urlencoded form。
func RequestParamString(ctx *arkweb.Context, name string, options ...ParamOption) (string, error) {
	value, ok, err := formValue(ctx, name)
	return resolveStringParameter("请求参数", name, value, ok, err, newParamOptions(ctx, options))
}

// RequestHeaderInt64 绑定 int64 请求头。
func RequestHeaderInt64(ctx *arkweb.Context, name string, options ...ParamOption) (int64, error) {
	value, ok, err := headerValue(ctx, name)
	return resolveInt64Parameter("请求头", name, value, ok, err, newParamOptions(ctx, options))
}

// PathVariableInt 绑定 int 路径变量。
func PathVariableInt(ctx *arkweb.Context, name string, options ...ParamOption) (int, error) {
	return PathInt(ctx, name, options...)
}

// PathVariableInt64s 绑定 int64 切片路径变量。
func PathVariableInt64s(ctx *arkweb.Context, name string, options ...ParamOption) ([]int64, error) {
	return PathInt64s(ctx, name, options...)
}

// Violations 返回全部验证失败项。
func (r BindingResult) Violations() []validation.Violation {
	return r.result.Violations()
}
