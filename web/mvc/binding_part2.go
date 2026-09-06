package mvc

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	servletmultipart "goark.dev/arkarta/servlet/multipart"

	arkweb "goark.dev/arkarta/web"
	goweb "goark.dev/goark/web"

	"goark.dev/arkarta/servlet"
	"goark.dev/goark/container"
	"goark.dev/goark/core/convert"
	"goark.dev/goark/web/message"
)

func (b *DataBinder) extractFieldMarkers(values url.Values) (url.Values, map[string]struct{}) {
	prefix := b.FieldMarkerPrefix()
	if prefix == "" || len(values) == 0 {
		return values, nil
	}
	var out url.Values
	var markers map[string]struct{}
	for name := range values {
		field, ok := strings.CutPrefix(name, prefix)
		if !ok {
			continue
		}
		out = ensureClonedModelAttributeValues(out, values)
		delete(out, name)
		if field == "" {
			continue
		}
		if _, exists := out[field]; exists {
			continue
		}
		if markers == nil {
			markers = make(map[string]struct{})
		}
		markers[field] = struct{}{}
	}
	if out == nil {
		return values, markers
	}
	return out, markers
}

func bindJSONResult[In any, Out any](statusCode int, fn BindResultFunc[In, Out],
	groups []string) arkweb.Handler {
	validationGroups := cloneValidationGroups(groups)
	return arkweb.HandlerFunc(func(ctx *arkweb.Context) (arkweb.Result, error) {
		var input In
		if err := message.ReaderFromContext(ctx).Read(ctx, &input, message.MediaTypeJSON); err != nil {
			return nil, err
		}
		binding, err := validateBindingResult(ctx, &input, validationGroups)
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
func matrixValueLists(ctx *arkweb.Context, pathVariable string) map[string][]string {
	if pathVariable != "" {
		segment, ok := ctx.Param(pathVariable)
		if !ok {
			return map[string][]string{}
		}
		return matrixSegmentValueLists(segment)
	}
	out := make(map[string][]string)
	for _, segment := range strings.Split(strings.Trim(ctx.Request().Path(), "/"), "/") {
		for name, values := range matrixSegmentValueLists(segment) {
			out[name] = append(out[name], values...)
		}
	}
	return out
}

// RegisterConversionService 注册 MVC 转换服务贡献点。
func RegisterConversionService(registry *container.Registry, name string,
	service *convert.Service, options ...container.Option) error {
	return goweb.RegisterConfigurer(registry, name, goweb.ConfigurerFunc(func(ctx context.Context,
		webRegistry *goweb.Registry) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if webRegistry == nil {
			return goweb.ErrNilRegistry
		}
		webRegistry.Use(ConversionInterceptor(service))
		return nil
	}), options...)
}
func cookieValue(ctx *arkweb.Context, name string) (string, bool, error) {
	if ctx == nil {
		return "", false, arkweb.ErrNilContext
	}
	cookie, err := ctx.Cookie(name)
	if err == nil {
		return cookie.Value, true, nil
	}
	if errors.Is(err, servlet.ErrNoCookie) {
		return "", false, nil
	}
	return "", false, err
}

func parseParamTime(value string, options paramOptions) (time.Time, error) {
	layouts := paramTimeLayouts(options)
	var lastErr error
	for _, layout := range layouts {
		parsed, err := time.Parse(layout, value)
		if err == nil {
			return parsed, nil
		}
		lastErr = err
	}
	return time.Time{}, lastErr
}

// BindConversionService 将 MVC 转换服务绑定到当前请求上下文。
func BindConversionService(ctx *arkweb.Context, service *convert.Service) error {
	if ctx == nil || ctx.Request() == nil {
		return arkweb.ErrNilContext
	}
	if service == nil {
		service = DefaultConversionService()
	}
	ctx.Request().SetAttribute(AttributeConversionService, service)
	return nil
}
func (b *DataBinder) inheritFieldRules(parent *DataBinder) {
	if b == nil || parent == nil {
		return
	}
	b.allowedFields = parent.AllowedFields()
	b.disallowedFields = parent.DisallowedFields()
	b.fieldMarkerPrefix = parent.FieldMarkerPrefix()
	b.fieldDefaultPrefix = parent.FieldDefaultPrefix()
}

func pathValues(ctx *arkweb.Context, name string) ([]string, bool, error) {
	if ctx == nil {
		return nil, false, arkweb.ErrNilContext
	}
	value, ok := ctx.Param(name)
	if !ok {
		return nil, false, nil
	}
	return []string{stripMatrixSegment(value)}, true, nil
}

// WithTimeLayout 追加时间参数解析布局，优先于内置布局尝试。
func WithTimeLayout(layout string) ParamOption {
	layout = strings.TrimSpace(layout)
	return func(options *paramOptions) {
		if layout != "" {
			options.timeLayouts = append(options.timeLayouts, layout)
		}
	}
}

// PathBool 绑定 bool 路径变量。
func PathBool(ctx *arkweb.Context, name string, options ...ParamOption) (bool, error) {
	if ctx == nil {
		return false, arkweb.ErrNilContext
	}
	value, ok := ctx.Param(name)
	value = stripMatrixSegment(value)
	return resolveBoolParameter("路径变量", name, value, ok, nil, newParamOptions(ctx, options))
}
func (r BindingResult) withSuppressedFields(fields []string) BindingResult {
	if len(fields) == 0 {
		return r
	}
	r.suppressedFields = append([]string(nil), fields...)
	return r
}

// FieldDefaultPrefix 返回字段默认值前缀。
func (b *DataBinder) FieldDefaultPrefix() string {
	if b == nil {
		return ""
	}
	return b.fieldDefaultPrefix
}

// RequestParamMap 绑定全部请求参数，每个名称取第一个值。
func RequestParamMap(ctx *arkweb.Context) (map[string]string, error) {
	values, err := requestParameters(ctx)
	if err != nil {
		return nil, err
	}
	return firstStringValueMap(values), nil
}
func cookieValues(ctx *arkweb.Context, name string) ([]string, bool, error) {
	value, ok, err := cookieValue(ctx, name)
	if err != nil || !ok {
		return nil, ok, err
	}
	return []string{value}, true, nil
}

func invalidParameterError(name, value, targetType string, cause error) error {
	return &arkweb.ParameterError{
		Name:  name,
		Value: value,
		Type:  targetType,
		Cause: fmt.Errorf("%w: %v", arkweb.ErrInvalidParameter, cause),
	}
}

// AddConverter 向当前请求作用域的转换服务注册转换器。
func (b *DataBinder) AddConverter(converter Converter) error {
	if b == nil || b.conversionService == nil {
		return ErrNilDataBinder
	}
	return b.conversionService.Register(converter)
}

// MatrixVariableFloat64 绑定 float64 矩阵变量。
func MatrixVariableFloat64(ctx *arkweb.Context, name string, options ...ParamOption) (float64,
	error) {
	paramOptions := newParamOptions(ctx, options)
	value, ok, err := matrixValue(ctx, name, paramOptions.matrixPathVariable)
	return resolveFloat64Parameter("矩阵变量", name, value, ok, err, paramOptions)
}

// FlashAttributeAs 将 Flash 属性转换为目标类型。
func FlashAttributeAs[T any](ctx *arkweb.Context, name string, options ...ParamOption) (T, error) {
	value, ok, err := flashAttributeValue(ctx, name)
	paramOptions := newParamOptions(ctx, options)
	return resolveConvertedParameter("Flash属性", name, value, ok, err, paramOptions,
		paramTargetType[T](), convertParamValue[T](paramOptions))
}

// MatrixVariableTime 绑定 time.Time 矩阵变量。
func MatrixVariableTime(ctx *arkweb.Context, name string, options ...ParamOption) (time.Time,
	error) {
	paramOptions := newParamOptions(ctx, options)
	value, ok, err := matrixValue(ctx, name, paramOptions.matrixPathVariable)
	return resolveTimeParameter("矩阵变量", name, value, ok, err, paramOptions)
}

// FlashAttributeString 绑定字符串 Flash 属性。
func FlashAttributeString(ctx *arkweb.Context, name string, options ...ParamOption) (string,
	error) {
	value, ok, err := flashAttributeValue(ctx, name)
	return resolveStringParameter("Flash属性", name, value, ok, err, newParamOptions(ctx, options))
}

const (
	attributeDataBinder       = "goark.dev/goark/web/mvc.dataBinder"
	defaultFieldMarkerPrefix  = "_"
	defaultFieldDefaultPrefix = "!"
)

// RequestParamFloat64 绑定 float64 请求参数，参数视图包含 query 和 urlencoded form。
func RequestParamFloat64(ctx *arkweb.Context, name string, options ...ParamOption) (float64,
	error) {
	value, ok, err := formValue(ctx, name)
	return resolveFloat64Parameter("请求参数", name, value, ok, err, newParamOptions(ctx, options))
}

// PathTimes 绑定 time.Time 切片路径变量。
func PathTimes(ctx *arkweb.Context, name string, options ...ParamOption) ([]time.Time, error) {
	values, ok, err := pathValues(ctx, name)
	return resolveTimeSliceParameter("路径变量", name, values, ok, err, newParamOptions(ctx, options))
}

// RequestHeaderFloat64s 绑定 float64 切片请求头。
func RequestHeaderFloat64s(ctx *arkweb.Context, name string, options ...ParamOption) (
	[]float64, error) {
	values, ok, err := headerValues(ctx, name)
	return resolveFloat64SliceParameter("请求头", name, values, ok, err, newParamOptions(ctx, options))
}
func resolveBoolSliceParameter(kind, name string, values []string, ok bool, err error,
	options paramOptions) ([]bool, error) {
	return resolveConvertedSliceParameter(kind, name, values, ok, err, options, "[]bool",
		convertParamValue[bool](options))
}

// CookieValueBool 绑定 bool Cookie 值。
func CookieValueBool(ctx *arkweb.Context, name string, options ...ParamOption) (bool, error) {
	value, ok, err := cookieValue(ctx, name)
	return resolveBoolParameter("Cookie", name, value, ok, err, newParamOptions(ctx, options))
}

// PathVariableFloat64s 绑定 float64 切片路径变量。
func PathVariableFloat64s(ctx *arkweb.Context, name string, options ...ParamOption) ([]float64,
	error) {
	return PathFloat64s(ctx, name, options...)
}

// BindJSONResult 绑定并验证 JSON 请求体，验证失败时由处理函数读取 BindingResult。
func BindJSONResult[In any, Out any](statusCode int, fn BindResultFunc[In, Out]) arkweb.Handler {
	return bindJSONResult(statusCode, fn, nil)
}

// CookieValueFloat64 绑定 float64 Cookie 值。
func CookieValueFloat64(ctx *arkweb.Context, name string, options ...ParamOption) (float64, error) {
	value, ok, err := cookieValue(ctx, name)
	return resolveFloat64Parameter("Cookie", name, value, ok, err, newParamOptions(ctx, options))
}

// PathFloat64s 绑定 float64 切片路径变量。
func PathFloat64s(ctx *arkweb.Context, name string, options ...ParamOption) ([]float64, error) {
	values, ok, err := pathValues(ctx, name)
	return resolveFloat64SliceParameter("路径变量", name, values, ok, err, newParamOptions(ctx, options))
}

// CookieValueInts 绑定 int 切片 Cookie 值。
func CookieValueInts(ctx *arkweb.Context, name string, options ...ParamOption) ([]int, error) {
	values, ok, err := cookieValues(ctx, name)
	return resolveIntSliceParameter("Cookie", name, values, ok, err, newParamOptions(ctx, options))
}

// RequestParamInt 绑定 int 请求参数，参数视图包含 query 和 urlencoded form。
func RequestParamInt(ctx *arkweb.Context, name string, options ...ParamOption) (int, error) {
	value, ok, err := formValue(ctx, name)
	return resolveIntParameter("请求参数", name, value, ok, err, newParamOptions(ctx, options))
}

// RequestHeaderBool 绑定 bool 请求头。
func RequestHeaderBool(ctx *arkweb.Context, name string, options ...ParamOption) (bool, error) {
	value, ok, err := headerValue(ctx, name)
	return resolveBoolParameter("请求头", name, value, ok, err, newParamOptions(ctx, options))
}

// PathVariableInt64 绑定 int64 路径变量。
func PathVariableInt64(ctx *arkweb.Context, name string, options ...ParamOption) (int64, error) {
	return PathInt64(ctx, name, options...)
}

// PathVariableBools 绑定 bool 切片路径变量。
func PathVariableBools(ctx *arkweb.Context, name string, options ...ParamOption) ([]bool, error) {
	return PathBools(ctx, name, options...)
}

// Multipart 绑定 multipart/form-data 请求体并执行结构体验证。
func Multipart[T any](ctx *arkweb.Context, options ...servletmultipart.Option) (T, error) {
	return MultipartGroups[T](ctx, nil, options...)
}
func shouldAppendFormContent(req *servlet.Request) bool {
	return req != nil && strings.EqualFold(req.Method(), http.MethodDelete)
}
