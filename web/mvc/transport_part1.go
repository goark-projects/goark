package mvc

import (
	"errors"
	"net/http"
	"reflect"

	servletmultipart "goark.dev/arkarta/servlet/multipart"

	arkweb "goark.dev/arkarta/web"
	"goark.dev/goark/core/convert"
	goweb "goark.dev/goark/web"

	"goark.dev/goark/web/message"
	"goark.dev/goark/web/mvc/view"
)

func returnValueByKind(ctx *arkweb.Context, statusCode int, value any,
	kind ControllerKind) arkweb.Result {
	switch typed := value.(type) {
	case ModelAndView:
		return mergeModelAndView(ctx, typed)
	case *ModelAndView:
		if typed == nil {
			return responseBodyResult(ctx, statusCode, nil)
		}
		return mergeModelAndView(ctx, *typed)
	case Model:
		return implicitModelResult(ctx, statusCode, mergeCurrentModel(ctx, typed))
	case *Model:
		if typed == nil {
			return responseBodyResult(ctx, statusCode, nil)
		}
		return implicitModelResult(ctx, statusCode, mergeCurrentModel(ctx, *typed))
	}
	if kind == ControllerKindREST {
		return responseBodyResult(ctx, statusCode, value)
	}
	if name, ok := value.(string); ok {
		if result, ok := forwardResultFromViewName(name); ok {
			return result
		}
		if result, ok := redirectResultFromViewName(ctx, statusCode, name); ok {
			return result
		}
		return view.Render(name, modelForView(ctx), view.WithStatus(resolveResponseStatus(ctx,
			statusCode, http.StatusOK)))
	}
	return responseBodyResult(ctx, statusCode, value)
}

// ResponseStatus 为处理器设置方法级默认 HTTP 状态码。
func ResponseStatus(statusCode int, handler arkweb.Handler) arkweb.Handler {
	if handler == nil {
		return nil
	}
	statusCode = normalizeResponseStatus(statusCode, http.StatusOK)
	return arkweb.HandlerFunc(func(ctx *arkweb.Context) (arkweb.Result, error) {
		restore := bindResponseStatus(ctx, statusCode)
		result, err := handler.Handle(ctx)
		restore()
		if err != nil {
			return nil, err
		}
		if result == nil {
			return responseStatusResult{statusCode: statusCode}, nil
		}
		return responseStatusResult{statusCode: statusCode, result: result}, nil
	})
}

func bindRequestEntity[In any, Out any](statusCode int, fn BindRequestEntityFunc[In, Out],
	groups []string, mediaTypes []string) arkweb.Handler {
	validationGroups := cloneValidationGroups(groups)
	readMediaTypes := cleanRouteValues(mediaTypes)
	return arkweb.HandlerFunc(func(ctx *arkweb.Context) (arkweb.Result, error) {
		input, err := requestEntity[In](ctx, validationGroups, readMediaTypes)
		if err != nil {
			return nil, err
		}
		value, err := fn(ctx, input)
		if err != nil {
			return nil, err
		}
		return jsonResult(ctx, statusCode, value), nil
	})
}

func supportsValidation(target any) bool {
	typ := reflect.TypeOf(target)
	if typ == nil {
		return false
	}
	typ = indirectValidationType(typ)
	switch typ.Kind() {
	case reflect.Struct:
		return true
	case reflect.Array, reflect.Slice:
		return indirectValidationType(typ.Elem()).Kind() == reflect.Struct
	default:
		return false
	}
}

var (
	// ErrNilModelAttributeInitializer 表示模型初始化器为空。
	ErrNilModelAttributeInitializer = errors.New("goark/web/mvc: model attribute initializer is nil")
	// ErrInvalidModelAttributeName 表示模型属性名非法。
	ErrInvalidModelAttributeName = errors.New("goark/web/mvc: invalid model attribute name")
	// ErrNilBinderInitializer 表示绑定器初始化器为空。
	ErrNilBinderInitializer = errors.New("goark/web/mvc: binder initializer is nil")
	// ErrNilDataBinder 表示数据绑定器为空。
	ErrNilDataBinder = errors.New("goark/web/mvc: data binder is nil")
	// ErrInvalidForwardLocation 表示 forward 目标路径非法。
	ErrInvalidForwardLocation = errors.New("goark/web/mvc: invalid forward location")
	// ErrForwardDispatcherUnavailable 表示当前请求无法获取 Servlet 分发器。
	ErrForwardDispatcherUnavailable = errors.New("goark/web/mvc: forward dispatcher unavailable")
)

// OptionalValidatedRequestBodyWithMediaTypes 按指定媒体类型可选绑定请求体并执行结构体验证。
func OptionalValidatedRequestBodyWithMediaTypes[T any](ctx *arkweb.Context,
	mediaTypes []string, groups ...string) (T, bool, error) {
	var out T
	present, err := requestBodyPresent(ctx)
	if err != nil || !present {
		return out, false, err
	}
	if err := bindAndValidateBody(ctx, &out, groups, mediaTypes); err != nil {
		return out, false, err
	}
	return out, true, nil
}

// RequestParts 返回 multipart/form-data 中的全部文件段。
func RequestParts(ctx *arkweb.Context) ([]servletmultipart.Part, error) {
	if ctx == nil || ctx.Request() == nil {
		return nil, arkweb.ErrNilContext
	}
	parts, err := servletmultipart.Parts(ctx.Request(), servletmultipart.NewParser())
	if err != nil {
		return nil, err
	}
	return parts, nil
}

// ExceptionHandlerIf 使用谓词匹配哨兵错误或自定义错误条件。
func ExceptionHandlerIf(match ExceptionPredicate, fn func(ctx *arkweb.Context,
	err error) arkweb.Result) goweb.ErrorMapper {
	return ExceptionHandler(func(ctx *arkweb.Context, err error) (arkweb.Result, bool) {
		if match == nil || fn == nil || !match(err) {
			return nil, false
		}
		return fn(ctx, err), true
	})
}

func (r responseStatusResult) Write(ctx *arkweb.Context) error {
	if ctx == nil || ctx.Response() == nil {
		return arkweb.ErrNilContext
	}
	ctx.Response().SetStatus(r.statusCode)
	if r.result == nil {
		return nil
	}
	return r.result.Write(ctx)
}

func bindAndValidateBody(ctx *arkweb.Context, target any, groups []string,
	mediaTypes []string) error {
	if err := message.ReaderFromContext(ctx).Read(ctx, target, mediaTypes...); err != nil {
		return err
	}
	if !supportsValidation(target) {
		return nil
	}
	return validateBound(ctx, target, groups)
}

func validatedRequestBodyResult[T any](ctx *arkweb.Context, mediaTypes []string,
	groups []string) (T, BindingResult, error) {
	var out T
	if err := message.ReaderFromContext(ctx).Read(ctx, &out, mediaTypes...); err != nil {
		return out, BindingResult{}, err
	}
	result, err := validateBindingResult(ctx, &out, groups)
	return out, result, err
}

func normalizeResponseStatus(statusCode int, fallback int) int {
	if statusCode == 0 {
		return fallback
	}
	if statusCode < 100 || statusCode > 999 {
		return http.StatusInternalServerError
	}
	return statusCode
}

// RequestBodyWithMediaTypes 将请求体按指定媒体类型集合绑定到目标类型。
func RequestBodyWithMediaTypes[T any](ctx *arkweb.Context, mediaTypes ...string) (T, error) {
	var out T
	if err := message.ReaderFromContext(ctx).Read(ctx, &out, mediaTypes...); err != nil {
		return out, err
	}
	return out, nil
}

func responseBodyResult(ctx *arkweb.Context, statusCode int, value any) arkweb.Result {
	statusCode = resolveResponseStatus(ctx, statusCode, http.StatusOK)
	if mediaType, ok := selectedProducesMediaType(ctx); ok {
		return goweb.Message(statusCode, value, mediaType)
	}
	return goweb.Message(statusCode, value)
}

// ValidatedRequestBodyResult 将请求体绑定到目标类型，并返回可由调用方处理的验证结果。
func ValidatedRequestBodyResult[T any](ctx *arkweb.Context, groups ...string) (T,
	BindingResult, error) {
	return validatedRequestBodyResult[T](ctx, nil, groups)
}

// ValidatedRequestBodyResultWithMediaTypes 将请求体按指定媒体类型集合绑定，并返回验证结果。
func ValidatedRequestBodyResultWithMediaTypes[T any](ctx *arkweb.Context, mediaTypes []string,
	groups ...string) (T, BindingResult, error) {
	return validatedRequestBodyResult[T](ctx, mediaTypes, groups)
}

// BindRequestEntityEntity 绑定并校验请求实体，再写出响应实体。
func BindRequestEntityEntity[In any, Out any](fn BindRequestEntityResponseFunc[In,
	Out]) arkweb.Handler {
	return bindRequestEntityEntity(fn, nil, nil)
}

// ValidatedRequestEntity 将请求体绑定到目标类型，并按可选分组执行结构体验证。
func ValidatedRequestEntity[T any](ctx *arkweb.Context, groups ...string) (
	goweb.RequestEntity[T], error) {
	return requestEntity[T](ctx, cloneValidationGroups(groups), nil)
}

// BindJSONGroups 绑定 JSON 请求体，并按显式校验分组写出 JSON 响应。
func BindJSONGroups[In any, Out any](statusCode int, fn BindFunc[In, Out],
	groups ...string) arkweb.Handler {
	return bindJSON(statusCode, fn, groups)
}

// BindEntityGroups 绑定 JSON 请求体，并按显式校验分组写出响应实体。
func BindEntityGroups[In any, Out any](fn BindEntityFunc[In, Out],
	groups ...string) arkweb.Handler {
	return bindEntity(fn, groups)
}

// BindRequestEntityEntityGroups 绑定请求实体，并按显式校验分组写出响应实体。
func BindRequestEntityEntityGroups[In any, Out any](fn BindRequestEntityResponseFunc[In, Out],
	groups ...string) arkweb.Handler {
	return bindRequestEntityEntity(fn, groups, nil)
}

// BindMultipartEntityGroups 绑定 multipart/form-data 请求体，并按显式校验分组写出响应实体。
func BindMultipartEntityGroups[In any, Out any](fn BindEntityFunc[In, Out], groups []string,
	options ...servletmultipart.Option) arkweb.Handler {
	return bindMultipartEntity(fn, groups, options...)
}

// RequestPartJSON 将指定 multipart 文件段按 JSON 绑定到目标类型，不执行结构体验证。
func RequestPartJSON[T any](ctx *arkweb.Context, name string, options ...ParamOption) (T, error) {
	return requestPartJSON[T](ctx, name, nil, false, options...)
}

func indirectValidationType(typ reflect.Type) reflect.Type {
	for typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}
	return typ
}
func adviceReturnValue(ctx *arkweb.Context, statusCode int, value any) arkweb.Result {
	return returnValueByKind(ctx, statusCode, value, ControllerAdviceKindFromContext(ctx))
}

// ExceptionResultFunc 将指定错误转换为 Web 响应。
type ExceptionResultFunc[E error] func(ctx *arkweb.Context, err E) arkweb.Result

// BinderInitializer 表示控制器级 @InitBinder 的 Go 化初始化器。
type BinderInitializer interface {
	InitializeBinder(ctx *arkweb.Context, binder *DataBinder) error
}

// BinderInitializerFunc 将函数适配为 BinderInitializer。
type BinderInitializerFunc func(ctx *arkweb.Context, binder *DataBinder) error

// ConverterFunc 将类型安全函数适配为 MVC 转换器。
type ConverterFunc[S any, T any] = convert.ConverterFunc[S, T]

// BindingError 返回绑定阶段错误；没有绑定错误时返回 nil。
func (r BindingResult) BindingError() error {
	return r.bindingError
}

// PathVariableAs 将路径变量转换为目标类型。
func PathVariableAs[T any](ctx *arkweb.Context, name string, options ...ParamOption) (T, error) {
	return PathValueAs[T](ctx, name, options...)
}

// HasErrors 返回绑定对象是否存在验证错误。
func (r BindingResult) HasErrors() bool {
	return !r.Valid()
}

func newBindingErrorResult(err error) BindingResult {
	return BindingResult{bindingError: err}
}
