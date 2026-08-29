package mvc

import (
	"fmt"
	"net/http"

	"goark.dev/arkarta/servlet"
	arkweb "goark.dev/arkarta/web"
	"goark.dev/goark/core/convert"
)

// ParamOption 定制 MVC 参数绑定行为。
type ParamOption func(*paramOptions)

type paramOptions struct {
	required          bool
	hasDefault        bool
	defaultValue      string
	timeLayouts       []string
	conversionService *convert.Service
}

// WithRequired 设置参数是否必须存在。
func WithRequired(required bool) ParamOption {
	return func(options *paramOptions) {
		options.required = required
	}
}

// WithDefaultValue 设置参数缺失时使用的默认文本值。
func WithDefaultValue(value string) ParamOption {
	return func(options *paramOptions) {
		options.required = false
		options.hasDefault = true
		options.defaultValue = value
	}
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

// PathInt 绑定 int 路径变量。
func PathInt(ctx *arkweb.Context, name string, options ...ParamOption) (int, error) {
	if ctx == nil {
		return 0, arkweb.ErrNilContext
	}
	value, ok := ctx.Param(name)
	value = stripMatrixSegment(value)
	return resolveIntParameter("路径变量", name, value, ok, nil, newParamOptions(ctx, options))
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

// PathBool 绑定 bool 路径变量。
func PathBool(ctx *arkweb.Context, name string, options ...ParamOption) (bool, error) {
	if ctx == nil {
		return false, arkweb.ErrNilContext
	}
	value, ok := ctx.Param(name)
	value = stripMatrixSegment(value)
	return resolveBoolParameter("路径变量", name, value, ok, nil, newParamOptions(ctx, options))
}

// RequestParamString 绑定字符串请求参数，参数视图包含 query 和 urlencoded form。
func RequestParamString(ctx *arkweb.Context, name string, options ...ParamOption) (string, error) {
	value, ok, err := formValue(ctx, name)
	return resolveStringParameter("请求参数", name, value, ok, err, newParamOptions(ctx, options))
}

// RequestParamInt 绑定 int 请求参数，参数视图包含 query 和 urlencoded form。
func RequestParamInt(ctx *arkweb.Context, name string, options ...ParamOption) (int, error) {
	value, ok, err := formValue(ctx, name)
	return resolveIntParameter("请求参数", name, value, ok, err, newParamOptions(ctx, options))
}

// RequestParamInt64 绑定 int64 请求参数，参数视图包含 query 和 urlencoded form。
func RequestParamInt64(ctx *arkweb.Context, name string, options ...ParamOption) (int64, error) {
	value, ok, err := formValue(ctx, name)
	return resolveInt64Parameter("请求参数", name, value, ok, err, newParamOptions(ctx, options))
}

// RequestParamBool 绑定 bool 请求参数，参数视图包含 query 和 urlencoded form。
func RequestParamBool(ctx *arkweb.Context, name string, options ...ParamOption) (bool, error) {
	value, ok, err := formValue(ctx, name)
	return resolveBoolParameter("请求参数", name, value, ok, err, newParamOptions(ctx, options))
}

// RequestHeaderString 绑定字符串请求头。
func RequestHeaderString(ctx *arkweb.Context, name string, options ...ParamOption) (string, error) {
	value, ok, err := headerValue(ctx, name)
	return resolveStringParameter("请求头", name, value, ok, err, newParamOptions(ctx, options))
}

// RequestHeaderInt 绑定 int 请求头。
func RequestHeaderInt(ctx *arkweb.Context, name string, options ...ParamOption) (int, error) {
	value, ok, err := headerValue(ctx, name)
	return resolveIntParameter("请求头", name, value, ok, err, newParamOptions(ctx, options))
}

// RequestHeaderInt64 绑定 int64 请求头。
func RequestHeaderInt64(ctx *arkweb.Context, name string, options ...ParamOption) (int64, error) {
	value, ok, err := headerValue(ctx, name)
	return resolveInt64Parameter("请求头", name, value, ok, err, newParamOptions(ctx, options))
}

// RequestHeaderBool 绑定 bool 请求头。
func RequestHeaderBool(ctx *arkweb.Context, name string, options ...ParamOption) (bool, error) {
	value, ok, err := headerValue(ctx, name)
	return resolveBoolParameter("请求头", name, value, ok, err, newParamOptions(ctx, options))
}

// CookieValueString 绑定字符串 Cookie 值。
func CookieValueString(ctx *arkweb.Context, name string, options ...ParamOption) (string, error) {
	value, ok, err := cookieValue(ctx, name)
	return resolveStringParameter("Cookie", name, value, ok, err, newParamOptions(ctx, options))
}

// CookieValueInt 绑定 int Cookie 值。
func CookieValueInt(ctx *arkweb.Context, name string, options ...ParamOption) (int, error) {
	value, ok, err := cookieValue(ctx, name)
	return resolveIntParameter("Cookie", name, value, ok, err, newParamOptions(ctx, options))
}

// CookieValueInt64 绑定 int64 Cookie 值。
func CookieValueInt64(ctx *arkweb.Context, name string, options ...ParamOption) (int64, error) {
	value, ok, err := cookieValue(ctx, name)
	return resolveInt64Parameter("Cookie", name, value, ok, err, newParamOptions(ctx, options))
}

// CookieValueBool 绑定 bool Cookie 值。
func CookieValueBool(ctx *arkweb.Context, name string, options ...ParamOption) (bool, error) {
	value, ok, err := cookieValue(ctx, name)
	return resolveBoolParameter("Cookie", name, value, ok, err, newParamOptions(ctx, options))
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

func formValue(ctx *arkweb.Context, name string) (string, bool, error) {
	if ctx == nil {
		return "", false, arkweb.ErrNilContext
	}
	return ctx.FormValue(name)
}

func headerValue(ctx *arkweb.Context, name string) (string, bool, error) {
	if ctx == nil || ctx.Request() == nil {
		return "", false, arkweb.ErrNilContext
	}
	value, ok := ctx.Request().HeaderValue(name)
	return value, ok, nil
}

func cookieValue(ctx *arkweb.Context, name string) (string, bool, error) {
	if ctx == nil {
		return "", false, arkweb.ErrNilContext
	}
	cookie, err := ctx.Cookie(name)
	if err == nil {
		return cookie.Value, true, nil
	}
	if err == http.ErrNoCookie {
		return "", false, nil
	}
	return "", false, err
}

func resolveStringParameter(kind, name, value string, ok bool, err error, options paramOptions) (string, error) {
	value, ok, err = resolveRawParameter(kind, name, value, ok, err, options)
	if err != nil || !ok {
		return "", err
	}
	return value, nil
}

func resolveIntParameter(kind, name, value string, ok bool, err error, options paramOptions) (int, error) {
	return resolveConvertedParameter(kind, name, value, ok, err, options, "int", convertParamValue[int](options))
}

func resolveInt64Parameter(kind, name, value string, ok bool, err error, options paramOptions) (int64, error) {
	return resolveConvertedParameter(kind, name, value, ok, err, options, "int64", convertParamValue[int64](options))
}

func resolveBoolParameter(kind, name, value string, ok bool, err error, options paramOptions) (bool, error) {
	return resolveConvertedParameter(kind, name, value, ok, err, options, "bool", convertParamValue[bool](options))
}

func resolveRawParameter(kind, name, value string, ok bool, err error, options paramOptions) (string, bool, error) {
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

func missingParameterError(kind, name string) error {
	return servlet.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("缺少%s %q", kind, name), nil)
}

func invalidParameterError(name, value, targetType string, cause error) error {
	return &arkweb.ParameterError{
		Name:  name,
		Value: value,
		Type:  targetType,
		Cause: fmt.Errorf("%w: %v", arkweb.ErrInvalidParameter, cause),
	}
}
