package mvc

import (
	"errors"
	"net/url"
	"sort"
	"strings"
	"time"

	servletmultipart "goark.dev/arkarta/servlet/multipart"

	arkweb "goark.dev/arkarta/web"

	"goark.dev/goark/core/convert"
)

func (b *DataBinder) applyFieldDefaults(values url.Values) url.Values {
	prefix := b.FieldDefaultPrefix()
	if prefix == "" || len(values) == 0 {
		return values
	}
	var out url.Values
	for name, list := range values {
		field, ok := strings.CutPrefix(name, prefix)
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

func sortedSuppressedFields(fields []string) []string {
	if len(fields) == 0 {
		return nil
	}
	sort.Strings(fields)
	out := fields[:0]
	for _, field := range fields {
		if field == "" {
			continue
		}
		if len(out) == 0 || out[len(out)-1] != field {
			out = append(out, field)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func (b *DataBinder) prepareModelAttributeValues(values url.Values) preparedModelAttributeValues {
	if b == nil {
		return preparedModelAttributeValues{values: values}
	}
	values = b.applyFieldDefaults(values)
	values, markers := b.extractFieldMarkers(values)
	values = b.adaptEmptyArrayIndices(values)
	var suppressed []string
	values, suppressed = b.filterModelAttributeValues(values)
	markers, suppressed = b.filterFieldMarkers(markers, suppressed)
	suppressed = sortedSuppressedFields(suppressed)
	return preparedModelAttributeValues{
		values:           values,
		fieldMarkers:     markers,
		suppressedFields: suppressed,
	}
}

func bindMultipartResult[In any, Out any](statusCode int, fn BindResultFunc[In, Out],
	groups []string, options ...servletmultipart.Option) arkweb.Handler {
	validationGroups := cloneValidationGroups(groups)
	multipartOptions := append([]servletmultipart.Option(nil), options...)
	return arkweb.HandlerFunc(func(ctx *arkweb.Context) (arkweb.Result, error) {
		input, binding, err := MultipartResultGroups[In](ctx, validationGroups, multipartOptions...)
		if err != nil {
			return nil, err
		}
		value, err := fn(ctx, input, binding)
		if err != nil {
			return nil, err
		}
		return jsonResult(ctx, statusCode, value), nil
	})
}

// PathVariableMap 绑定全部路径变量。
func PathVariableMap(ctx *arkweb.Context) (map[string]string, error) {
	if ctx == nil {
		return nil, arkweb.ErrNilContext
	}
	values := ctx.PathValues()
	if len(values) == 0 {
		return map[string]string{}, nil
	}
	out := make(map[string]string, len(values))
	for name, value := range values {
		out[name] = stripMatrixSegment(value)
	}
	return out, nil
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

func matrixValuesFor(ctx *arkweb.Context, name string, pathVariable string) ([]string, bool,
	error) {
	values, err := matrixValueListsForContext(ctx, pathVariable)
	if err != nil {
		return nil, false, err
	}
	list, ok := values[name]
	if !ok || len(list) == 0 {
		return nil, ok, err
	}
	return append([]string(nil), list...), true, nil
}
func newDataBinder(service *convert.Service) *DataBinder {
	if service == nil {
		service = DefaultConversionService()
	}
	return &DataBinder{
		conversionService:  service,
		fieldMarkerPrefix:  defaultFieldMarkerPrefix,
		fieldDefaultPrefix: defaultFieldDefaultPrefix,
	}
}

// WithConversionService 设置单次参数绑定使用的转换服务。
func WithConversionService(service *convert.Service) ParamOption {
	return func(options *paramOptions) {
		if service != nil {
			options.conversionService = service
		}
	}
}

func matrixValue(ctx *arkweb.Context, name string, pathVariable string) (string, bool, error) {
	if ctx == nil || ctx.Request() == nil {
		return "", false, arkweb.ErrNilContext
	}
	values := firstStringValueMap(matrixValueLists(ctx, pathVariable))
	value, ok := values[name]
	return value, ok, nil
}

func paramTimeLayouts(options paramOptions) []string {
	if len(options.timeLayouts) == 0 {
		return defaultParamTimeLayouts
	}
	layouts := make([]string, 0, len(options.timeLayouts)+len(defaultParamTimeLayouts))
	layouts = append(layouts, options.timeLayouts...)
	layouts = append(layouts, defaultParamTimeLayouts...)
	return layouts
}
func requestAttributeValue(ctx *arkweb.Context, name string) (string, bool, error) {
	value, ok, err := rawRequestAttributeValue(ctx, name)
	if err != nil || !ok {
		return "", ok, err
	}
	return attributeString(value), true, nil
}

// SetAllowedFields 设置允许绑定的字段模式；未设置时默认允许全部字段。
func (b *DataBinder) SetAllowedFields(fields ...string) error {
	if b == nil {
		return ErrNilDataBinder
	}
	b.allowedFields = normalizeBinderFieldPatterns(fields, false)
	return nil
}
func requestParameterValue(ctx *arkweb.Context, name string) (string, bool, error) {
	values, ok, err := requestParameterValues(ctx, name)
	if err != nil || !ok {
		return "", ok, err
	}
	return values[0], true, nil
}

// RequestHeaderMap 绑定全部请求头，每个名称取第一个值。
func RequestHeaderMap(ctx *arkweb.Context) (map[string]string, error) {
	values, err := requestHeaders(ctx)
	if err != nil {
		return nil, err
	}
	return firstStringValueMap(values), nil
}
func resolveTimeParameter(kind, name, value string, ok bool, err error, options paramOptions) (
	time.Time, error) {
	return resolveConvertedParameter(kind, name, value, ok, err, options, "time.Time", func(
		value string) (time.Time, error) {
		return parseParamTime(value, options)
	})
}

// AllowedFields 返回允许绑定字段模式快照。
func (b *DataBinder) AllowedFields() []string {
	if b == nil || len(b.allowedFields) == 0 {
		return nil
	}
	return append([]string(nil), b.allowedFields...)
}
func emptyArrayRequestParameterName(name string) string {
	if name == "" || strings.HasSuffix(name, "[]") {
		return name
	}
	return name + "[]"
}

// RequestParamAs 将请求参数转换为目标类型，参数视图包含 query 和 urlencoded form。
func RequestParamAs[T any](ctx *arkweb.Context, name string, options ...ParamOption) (T, error) {
	value, ok, err := formValue(ctx, name)
	paramOptions := newParamOptions(ctx, options)
	return resolveConvertedParameter("请求参数", name, value, ok, err, paramOptions,
		paramTargetType[T](), convertParamValue[T](paramOptions))
}

// MatrixVariableInt64s 绑定 int64 切片矩阵变量。
func MatrixVariableInt64s(ctx *arkweb.Context, name string, options ...ParamOption) ([]int64,
	error) {
	paramOptions := newParamOptions(ctx, options)
	values, ok, err := matrixValuesFor(ctx, name, paramOptions.matrixPathVariable)
	return resolveInt64SliceParameter("矩阵变量", name, values, ok, err, paramOptions)
}

// RequestAttributeString 绑定字符串请求属性。
func RequestAttributeString(ctx *arkweb.Context, name string, options ...ParamOption) (string,
	error) {
	value, ok, err := requestAttributeValue(ctx, name)
	return resolveStringParameter("请求属性", name, value, ok, err, newParamOptions(ctx, options))
}

// FlashAttributeBool 绑定 bool Flash 属性。
func FlashAttributeBool(ctx *arkweb.Context, name string, options ...ParamOption) (bool, error) {
	value, ok, err := flashAttributeValue(ctx, name)
	return resolveBoolParameter("Flash属性", name, value, ok, err, newParamOptions(ctx, options))
}

// MatrixVariableInt 绑定 int 矩阵变量。
func MatrixVariableInt(ctx *arkweb.Context, name string, options ...ParamOption) (int, error) {
	paramOptions := newParamOptions(ctx, options)
	value, ok, err := matrixValue(ctx, name, paramOptions.matrixPathVariable)
	return resolveIntParameter("矩阵变量", name, value, ok, err, paramOptions)
}

// RequestAttributeFloat64 绑定 float64 请求属性。
func RequestAttributeFloat64(ctx *arkweb.Context, name string, options ...ParamOption) (
	float64, error) {
	value, ok, err := requestAttributeValue(ctx, name)
	return resolveFloat64Parameter("请求属性", name, value, ok, err, newParamOptions(ctx, options))
}

// RequestParamFloat64s 绑定 float64 切片请求参数。
func RequestParamFloat64s(ctx *arkweb.Context, name string, options ...ParamOption) ([]float64,
	error) {
	values, ok, err := formValues(ctx, name)
	return resolveFloat64SliceParameter("请求参数", name, values, ok, err, newParamOptions(ctx, options))
}

// CookieValueStrings 绑定字符串切片 Cookie 值，支持逗号分隔值。
func CookieValueStrings(ctx *arkweb.Context, name string, options ...ParamOption) ([]string,
	error) {
	values, ok, err := cookieValues(ctx, name)
	return resolveStringSliceParameter("Cookie", name, values, ok, err, newParamOptions(ctx, options))
}

// RequestHeaderTime 绑定 time.Time 请求头。
func RequestHeaderTime(ctx *arkweb.Context, name string, options ...ParamOption) (time.Time,
	error) {
	value, ok, err := headerValue(ctx, name)
	return resolveTimeParameter("请求头", name, value, ok, err, newParamOptions(ctx, options))
}
func resolveInt64Parameter(kind, name, value string, ok bool, err error,
	options paramOptions) (int64, error) {
	return resolveConvertedParameter(kind, name, value, ok, err, options, "int64",
		convertParamValue[int64](options))
}

// SessionAttributeInt 绑定 int Session 属性。
func SessionAttributeInt(ctx *arkweb.Context, name string, options ...ParamOption) (int, error) {
	value, ok, err := sessionAttributeValue(ctx, name)
	return resolveIntParameter("Session属性", name, value, ok, err, newParamOptions(ctx, options))
}

// Err 将验证失败结果转换为 error。
func (r BindingResult) Err() error {
	return errors.Join(r.bindingError, r.result.Error())
}

// PathStrings 绑定字符串切片路径变量，支持逗号分隔值。
func PathStrings(ctx *arkweb.Context, name string, options ...ParamOption) ([]string, error) {
	values, ok, err := pathValues(ctx, name)
	return resolveStringSliceParameter("路径变量", name, values, ok, err, newParamOptions(ctx, options))
}

// RequestParamInt64s 绑定 int64 切片请求参数。
func RequestParamInt64s(ctx *arkweb.Context, name string, options ...ParamOption) ([]int64, error) {
	values, ok, err := formValues(ctx, name)
	return resolveInt64SliceParameter("请求参数", name, values, ok, err, newParamOptions(ctx, options))
}

// CookieValueBools 绑定 bool 切片 Cookie 值。
func CookieValueBools(ctx *arkweb.Context, name string, options ...ParamOption) ([]bool, error) {
	values, ok, err := cookieValues(ctx, name)
	return resolveBoolSliceParameter("Cookie", name, values, ok, err, newParamOptions(ctx, options))
}

// RequestParamBool 绑定 bool 请求参数，参数视图包含 query 和 urlencoded form。
func RequestParamBool(ctx *arkweb.Context, name string, options ...ParamOption) (bool, error) {
	value, ok, err := formValue(ctx, name)
	return resolveBoolParameter("请求参数", name, value, ok, err, newParamOptions(ctx, options))
}

// CookieValueInt 绑定 int Cookie 值。
func CookieValueInt(ctx *arkweb.Context, name string, options ...ParamOption) (int, error) {
	value, ok, err := cookieValue(ctx, name)
	return resolveIntParameter("Cookie", name, value, ok, err, newParamOptions(ctx, options))
}

// PathVariableTime 绑定 time.Time 路径变量。
func PathVariableTime(ctx *arkweb.Context, name string, options ...ParamOption) (time.Time, error) {
	return PathTime(ctx, name, options...)
}

// Valid 返回绑定对象是否通过验证。
func (r BindingResult) Valid() bool {
	return r.bindingError == nil && r.result.Valid()
}

// DefaultConversionService 创建 MVC 默认转换服务。
func DefaultConversionService() *convert.Service {
	return defaultMVCConversionService
}
func formValue(ctx *arkweb.Context, name string) (string, bool, error) {
	return requestParameterValue(ctx, name)
}
