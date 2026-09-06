package mvc

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	servletmultipart "goark.dev/arkarta/servlet/multipart"

	arkweb "goark.dev/arkarta/web"

	"goark.dev/arkarta/servlet"
	"goark.dev/arkarta/validation"
	"goark.dev/goark/core/convert"
	"goark.dev/goark/core/lang"
)

func (b *DataBinder) adaptEmptyArrayIndices(values url.Values) url.Values {
	if len(values) == 0 {
		return values
	}
	var out url.Values
	for name, list := range values {
		field, ok := strings.CutSuffix(name, "[]")
		if !ok {
			continue
		}
		out = ensureClonedModelAttributeValues(out, values)
		delete(out, name)
		if field == "" {
			continue
		}
		if _, exists := out[field]; !exists {
			out[field] = append([]string(nil), list...)
		}
	}
	if out == nil {
		return values
	}
	return out
}

func normalizeBinderFieldPatterns(fields []string, foldCase bool) []string {
	if len(fields) == 0 {
		return nil
	}
	out := make([]string, 0, len(fields))
	for _, field := range fields {
		field = strings.TrimSpace(field)
		if field == "" {
			continue
		}
		if foldCase {
			field = strings.ToLower(field)
		}
		out = append(out, field)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func resolveConvertedSliceParameter[T any](kind, name string, values []string, ok bool,
	err error, options paramOptions, targetType string, convert func(string) (T, error)) ([]T, error) {
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

func newParamOptions(ctx *arkweb.Context, options []ParamOption) paramOptions {
	out := paramOptions{
		required:          true,
		conversionService: ConversionServiceFromContext(ctx),
	}
	for _, option := range options {
		if option != nil {
			option(&out)
		}
	}
	if out.conversionService == nil {
		out.conversionService = DefaultConversionService()
	}
	return out
}

// FieldErrors 返回指定字段的全部验证失败项。
func (r BindingResult) FieldErrors(path string) []validation.Violation {
	violations := r.result.Violations()
	if len(violations) == 0 {
		return nil
	}
	matched := make([]validation.Violation, 0, 1)
	for _, violation := range violations {
		if violation.Path() == path {
			matched = append(matched, violation)
		}
	}
	return matched
}
func (b *DataBinder) isFieldAllowed(name string) bool {
	if b == nil {
		return true
	}
	if len(b.allowedFields) > 0 && !matchesBinderFieldPattern(b.allowedFields, name) {
		return false
	}
	if len(b.disallowedFields) == 0 {
		return true
	}
	return !matchesBinderFieldPattern(b.disallowedFields, strings.ToLower(name))
}

// MultipartGroups 绑定 multipart/form-data 请求体并按显式分组执行结构体验证。
func MultipartGroups[T any](ctx *arkweb.Context, groups []string,
	options ...servletmultipart.Option) (T, error) {
	var out T
	if ctx == nil {
		return out, arkweb.ErrNilContext
	}
	if err := ctx.BindMultipart(&out, options...); err != nil {
		return out, err
	}
	return out, validateBound(ctx, &out, groups)
}

// MatrixVariableValuesMap 绑定全部矩阵变量，保留每个名称的全部值。
func MatrixVariableValuesMap(ctx *arkweb.Context, options ...ParamOption) (map[string][]string,
	error) {
	paramOptions := newParamOptions(ctx, options)
	values, err := matrixValueListsForContext(ctx, paramOptions.matrixPathVariable)
	if err != nil {
		return nil, err
	}
	return cloneStringValuesMap(values), nil
}

func modelAttributeValuesForCurrentBinder(ctx *arkweb.Context,
	values url.Values) preparedModelAttributeValues {
	binder, ok := dataBinderFromContext(ctx)
	if !ok {
		binder = newDataBinder(nil)
	}
	return binder.prepareModelAttributeValues(values)
}

// PathFloat64 绑定 float64 路径变量。
func PathFloat64(ctx *arkweb.Context, name string, options ...ParamOption) (float64, error) {
	if ctx == nil {
		return 0, arkweb.ErrNilContext
	}
	value, ok := ctx.Param(name)
	value = stripMatrixSegment(value)
	return resolveFloat64Parameter("路径变量", name, value, ok, nil, newParamOptions(ctx, options))
}

// PathString 绑定字符串路径变量。
func PathString(ctx *arkweb.Context, name string, options ...ParamOption) (string, error) {
	if ctx == nil {
		return "", arkweb.ErrNilContext
	}
	value, ok := ctx.Param(name)
	value = stripMatrixSegment(value)
	return resolveStringParameter("路径变量", name, value, ok, nil, newParamOptions(ctx, options))
}
func sessionAttributeValue(ctx *arkweb.Context, name string) (string, bool, error) {
	value, ok, err := rawSessionAttributeValue(ctx, name)
	if err != nil || !ok {
		return "", ok, err
	}
	return attributeString(value), true, nil
}

// SetDisallowedFields 设置拒绝绑定的字段模式；拒绝规则优先于允许规则。
func (b *DataBinder) SetDisallowedFields(fields ...string) error {
	if b == nil {
		return ErrNilDataBinder
	}
	b.disallowedFields = normalizeBinderFieldPatterns(fields, true)
	return nil
}
func matrixValueListsForContext(ctx *arkweb.Context, pathVariable string) (map[string][]string,
	error) {
	if ctx == nil || ctx.Request() == nil {
		return nil, arkweb.ErrNilContext
	}
	return matrixValueLists(ctx, pathVariable), nil
}

// RequestHeaderValuesMap 绑定全部请求头，保留每个名称的全部值。
func RequestHeaderValuesMap(ctx *arkweb.Context) (map[string][]string, error) {
	values, err := requestHeaders(ctx)
	if err != nil {
		return nil, err
	}
	return cloneStringValuesMap(values), nil
}

type paramOptions struct {
	required           bool
	hasDefault         bool
	defaultValue       string
	timeLayouts        []string
	matrixPathVariable string
	conversionService  *convert.Service
}

// DisallowedFields 返回拒绝绑定字段模式快照。
func (b *DataBinder) DisallowedFields() []string {
	if b == nil || len(b.disallowedFields) == 0 {
		return nil
	}
	return append([]string(nil), b.disallowedFields...)
}

// MatrixVariableString 绑定字符串矩阵变量。
func MatrixVariableString(ctx *arkweb.Context, name string, options ...ParamOption) (string,
	error) {
	paramOptions := newParamOptions(ctx, options)
	value, ok, err := matrixValue(ctx, name, paramOptions.matrixPathVariable)
	return resolveStringParameter("矩阵变量", name, value, ok, err, paramOptions)
}

// RequestHeaderAs 将请求头转换为目标类型。
func RequestHeaderAs[T any](ctx *arkweb.Context, name string, options ...ParamOption) (T, error) {
	value, ok, err := headerValue(ctx, name)
	paramOptions := newParamOptions(ctx, options)
	return resolveConvertedParameter("请求头", name, value, ok, err, paramOptions,
		paramTargetType[T](), convertParamValue[T](paramOptions))
}

// MatrixVariableFloat64s 绑定 float64 切片矩阵变量。
func MatrixVariableFloat64s(ctx *arkweb.Context, name string, options ...ParamOption) (
	[]float64, error) {
	paramOptions := newParamOptions(ctx, options)
	values, ok, err := matrixValuesFor(ctx, name, paramOptions.matrixPathVariable)
	return resolveFloat64SliceParameter("矩阵变量", name, values, ok, err, paramOptions)
}

// RequestAttributeInt64 绑定 int64 请求属性。
func RequestAttributeInt64(ctx *arkweb.Context, name string, options ...ParamOption) (int64,
	error) {
	value, ok, err := requestAttributeValue(ctx, name)
	return resolveInt64Parameter("请求属性", name, value, ok, err, newParamOptions(ctx, options))
}

// RequestAttribute 读取请求属性，并转换为目标类型。
func RequestAttribute[T any](ctx *arkweb.Context, name string, options ...ParamOption) (T, error) {
	value, ok, err := rawRequestAttributeValue(ctx, name)
	paramOptions := newParamOptions(ctx, options)
	return resolveAttributeValue[T]("请求属性", name, value, ok, err, paramOptions)
}

// MatrixVariableInt64 绑定 int64 矩阵变量。
func MatrixVariableInt64(ctx *arkweb.Context, name string, options ...ParamOption) (int64, error) {
	paramOptions := newParamOptions(ctx, options)
	value, ok, err := matrixValue(ctx, name, paramOptions.matrixPathVariable)
	return resolveInt64Parameter("矩阵变量", name, value, ok, err, paramOptions)
}

// SessionAttributeFloat64 绑定 float64 Session 属性。
func SessionAttributeFloat64(ctx *arkweb.Context, name string, options ...ParamOption) (
	float64, error) {
	value, ok, err := sessionAttributeValue(ctx, name)
	return resolveFloat64Parameter("Session属性", name, value, ok, err, newParamOptions(ctx, options))
}

// RequestParamTimes 绑定 time.Time 切片请求参数。
func RequestParamTimes(ctx *arkweb.Context, name string, options ...ParamOption) ([]time.Time,
	error) {
	values, ok, err := formValues(ctx, name)
	return resolveTimeSliceParameter("请求参数", name, values, ok, err, newParamOptions(ctx, options))
}

// CookieValueFloat64s 绑定 float64 切片 Cookie 值。
func CookieValueFloat64s(ctx *arkweb.Context, name string, options ...ParamOption) ([]float64,
	error) {
	values, ok, err := cookieValues(ctx, name)
	return resolveFloat64SliceParameter("Cookie", name, values, ok, err, newParamOptions(ctx, options))
}

// RequestAttributeTime 绑定 time.Time 请求属性。
func RequestAttributeTime(ctx *arkweb.Context, name string, options ...ParamOption) (time.Time,
	error) {
	value, ok, err := requestAttributeValue(ctx, name)
	return resolveTimeParameter("请求属性", name, value, ok, err, newParamOptions(ctx, options))
}
func resolveBoolParameter(kind, name, value string, ok bool, err error, options paramOptions) (
	bool, error) {
	return resolveConvertedParameter(kind, name, value, ok, err, options, "bool",
		convertParamValue[bool](options))
}

// SessionAttributeBool 绑定 bool Session 属性。
func SessionAttributeBool(ctx *arkweb.Context, name string, options ...ParamOption) (bool, error) {
	value, ok, err := sessionAttributeValue(ctx, name)
	return resolveBoolParameter("Session属性", name, value, ok, err, newParamOptions(ctx, options))
}

const (
	// AttributeConversionService 是请求属性中保存 MVC 转换服务的键。
	AttributeConversionService = "goark.dev/goark/web/mvc.conversionService"
)

// PathInts 绑定 int 切片路径变量。
func PathInts(ctx *arkweb.Context, name string, options ...ParamOption) ([]int, error) {
	values, ok, err := pathValues(ctx, name)
	return resolveIntSliceParameter("路径变量", name, values, ok, err, newParamOptions(ctx, options))
}

// RequestParamBools 绑定 bool 切片请求参数。
func RequestParamBools(ctx *arkweb.Context, name string, options ...ParamOption) ([]bool, error) {
	values, ok, err := formValues(ctx, name)
	return resolveBoolSliceParameter("请求参数", name, values, ok, err, newParamOptions(ctx, options))
}

// RequestParamTime 绑定 time.Time 请求参数，参数视图包含 query 和 urlencoded form。
func RequestParamTime(ctx *arkweb.Context, name string, options ...ParamOption) (time.Time, error) {
	value, ok, err := formValue(ctx, name)
	return resolveTimeParameter("请求参数", name, value, ok, err, newParamOptions(ctx, options))
}

// RequestHeaderString 绑定字符串请求头。
func RequestHeaderString(ctx *arkweb.Context, name string, options ...ParamOption) (string, error) {
	value, ok, err := headerValue(ctx, name)
	return resolveStringParameter("请求头", name, value, ok, err, newParamOptions(ctx, options))
}

// CookieValueInt64 绑定 int64 Cookie 值。
func CookieValueInt64(ctx *arkweb.Context, name string, options ...ParamOption) (int64, error) {
	value, ok, err := cookieValue(ctx, name)
	return resolveInt64Parameter("Cookie", name, value, ok, err, newParamOptions(ctx, options))
}

func paramTargetType[T any]() string {
	return fmt.Sprint(lang.TypeOf[T]())
}

func missingParameterError(kind, name string) error {
	return servlet.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("缺少%s %q", kind, name), nil)
}
