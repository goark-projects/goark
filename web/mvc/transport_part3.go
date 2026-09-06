package mvc

import (
	"net/http"

	arkjson "goark.dev/arkarta/json"
	servletmultipart "goark.dev/arkarta/servlet/multipart"

	arkweb "goark.dev/arkarta/web"
	goweb "goark.dev/goark/web"

	"goark.dev/arkarta/servlet"
	"goark.dev/arkarta/validation"
	"goark.dev/goark/web/message"
)

func requestPartJSON[T any](ctx *arkweb.Context, name string, groups []string, validate bool,
	options ...ParamOption) (T, error) {
	var out T
	part, ok, err := requestPart(ctx, name, options...)
	if err != nil || !ok {
		return out, err
	}
	if err := ensureJSONPart(part.Header().Get("Content-Type")); err != nil {
		return out, err
	}
	reader, err := part.Open()
	if err != nil {
		return out, err
	}
	defer reader.Close()
	if err := arkjson.Decode(ctx.JSONCodec(), reader, &out); err != nil {
		return out, servlet.NewHTTPError(http.StatusBadRequest, "请求Part格式非法", err)
	}
	if validate {
		return out, validateBound(ctx, &out, groups)
	}
	return out, nil
}

func requestEntity[T any](ctx *arkweb.Context, groups []string, mediaTypes []string) (
	goweb.RequestEntity[T], error) {
	metadata, hasBody, err := requestEntityMetadata(ctx)
	if err != nil {
		return goweb.RequestEntity[T]{}, err
	}
	var body T
	if !hasBody {
		return goweb.NewRequestEntity(metadata, body, false), nil
	}
	if err := message.ReaderFromContext(ctx).Read(ctx, &body, mediaTypes...); err != nil {
		return goweb.RequestEntity[T]{}, err
	}
	if supportsValidation(&body) {
		if err := validateBound(ctx, &body, groups); err != nil {
			return goweb.RequestEntity[T]{}, err
		}
	}
	return goweb.NewRequestEntity(metadata, body, true), nil
}

func validateBindingResult(ctx *arkweb.Context, target any, groups []string) (BindingResult,
	error) {
	if !supportsValidation(target) {
		return BindingResult{}, nil
	}
	var (
		result validation.Result
		err    error
	)
	if len(groups) == 0 {
		result, err = ctx.Validate(target)
	} else {
		result, err = ctx.ValidateGroups(target, groups...)
	}
	if err != nil {
		return BindingResult{}, err
	}
	return NewBindingResult(result), nil
}

func bindRequestEntityEntity[In any, Out any](fn BindRequestEntityResponseFunc[In, Out],
	groups []string, mediaTypes []string) arkweb.Handler {
	validationGroups := cloneValidationGroups(groups)
	readMediaTypes := cleanRouteValues(mediaTypes)
	return arkweb.HandlerFunc(func(ctx *arkweb.Context) (arkweb.Result, error) {
		input, err := requestEntity[In](ctx, validationGroups, readMediaTypes)
		if err != nil {
			return nil, err
		}
		entity, err := fn(ctx, input)
		if err != nil {
			return nil, err
		}
		return entityResult(ctx, entity), nil
	})
}

func snapshotRequestHeaders(header servlet.Header) http.Header {
	if header == nil {
		return nil
	}
	result := make(http.Header)
	header.Visit(func(name, value string) bool {
		result.Add(name, value)
		return true
	})
	if len(result) == 0 {
		return nil
	}
	return result
}

// OptionalRequestBodyWithMediaTypes 按指定媒体类型可选绑定请求体。
func OptionalRequestBodyWithMediaTypes[T any](ctx *arkweb.Context, mediaTypes ...string) (T,
	bool, error) {
	var out T
	present, err := requestBodyPresent(ctx)
	if err != nil || !present {
		return out, false, err
	}
	if err := message.ReaderFromContext(ctx).Read(ctx, &out, mediaTypes...); err != nil {
		return out, false, err
	}
	return out, true, nil
}

// ExceptionHandler 将 MVC 异常函数适配为 Web 错误映射器。
func ExceptionHandler(fn ExceptionFunc) goweb.ErrorMapper {
	return goweb.ErrorMapperFunc(func(ctx *arkweb.Context, err error) arkweb.Result {
		if fn == nil {
			return nil
		}
		result, handled := fn(ctx, err)
		if !handled {
			return nil
		}
		return result
	})
}

func requestEntityURL(request *servlet.Request) string {
	if request == nil {
		return ""
	}
	rawURL := request.RequestURL()
	if query := request.QueryString(); query != "" {
		return rawURL + "?" + query
	}
	return rawURL
}

// ExceptionReturnAs 使用 advice 默认策略写出普通异常返回值。
func ExceptionReturnAs[E error, T any](statusCode int, fn ExceptionValueFunc[E,
	T]) goweb.ErrorMapper {
	return ExceptionHandlerAs(func(ctx *arkweb.Context, err E) arkweb.Result {
		if fn == nil {
			return nil
		}
		return adviceReturnValue(ctx, statusCode, fn(ctx, err))
	})
}

// Return 将普通返回值按当前控制器默认策略写出。
func Return[T any](statusCode int, fn ValueFunc[T]) arkweb.Handler {
	return arkweb.HandlerFunc(func(ctx *arkweb.Context) (arkweb.Result, error) {
		value, err := fn(ctx)
		if err != nil {
			return nil, err
		}
		return controllerReturnValue(ctx, statusCode, value), nil
	})
}

func requestBodyPresent(ctx *arkweb.Context) (bool, error) {
	if ctx == nil || ctx.Request() == nil {
		return false, arkweb.ErrNilContext
	}
	if ctx.Request().Body() == nil {
		return false, nil
	}
	return ctx.Request().ContentLength() != 0, nil
}

// ExceptionEntityAs 使用响应实体完整控制异常响应状态、头和响应体。
func ExceptionEntityAs[E error, T any](fn ExceptionEntityFunc[E, T]) goweb.ErrorMapper {
	return ExceptionHandlerAs(func(ctx *arkweb.Context, err E) arkweb.Result {
		if fn == nil {
			return nil
		}
		return entityResult(ctx, fn(ctx, err))
	})
}

func bindAndValidateJSON(ctx *arkweb.Context, target any, groups []string) error {
	if err := message.ReaderFromContext(ctx).Read(ctx, target, message.MediaTypeJSON); err != nil {
		return err
	}
	if !supportsValidation(target) {
		return nil
	}
	return validateBound(ctx, target, groups)
}

func cloneValidationGroups(groups []string) []string {
	if len(groups) == 0 {
		return nil
	}
	cloned := make([]string, len(groups))
	copy(cloned, groups)
	return cloned
}

// RequestPart 绑定 multipart/form-data 中的指定文件段。
func RequestPart(ctx *arkweb.Context, name string, options ...ParamOption) (
	servletmultipart.Part, error) {
	part, _, err := requestPart(ctx, name, options...)
	return part, err
}

// BindRequestEntity 绑定并校验请求实体，再将返回值写为 JSON 响应。
func BindRequestEntity[In any, Out any](statusCode int, fn BindRequestEntityFunc[In,
	Out]) arkweb.Handler {
	return bindRequestEntity(statusCode, fn, nil, nil)
}

// BindRequestEntityEntityWithMediaTypes 按指定媒体类型集合绑定请求实体，再写出响应实体。
func BindRequestEntityEntityWithMediaTypes[In any, Out any](
	fn BindRequestEntityResponseFunc[In, Out], mediaTypes ...string) arkweb.Handler {
	return bindRequestEntityEntity(fn, nil, mediaTypes)
}

// ValidatedRequestEntityWithMediaTypes 将请求体按指定媒体类型集合绑定，并执行结构体验证。
func ValidatedRequestEntityWithMediaTypes[T any](ctx *arkweb.Context, mediaTypes []string,
	groups ...string) (goweb.RequestEntity[T], error) {
	return requestEntity[T](ctx, cloneValidationGroups(groups), cleanRouteValues(mediaTypes))
}

// BindJSONResultGroups 绑定 JSON 请求体，并按显式校验分组传入 BindingResult。
func BindJSONResultGroups[In any, Out any](statusCode int, fn BindResultFunc[In, Out],
	groups ...string) arkweb.Handler {
	return bindJSONResult(statusCode, fn, groups)
}

// BindBodyEntityGroups 绑定请求体，并按显式校验分组写出响应实体。
func BindBodyEntityGroups[In any, Out any](fn BindEntityFunc[In, Out],
	groups ...string) arkweb.Handler {
	return bindBodyEntity(fn, groups, nil)
}

// BindMultipartGroups 绑定 multipart/form-data 请求体，并按显式校验分组写出 JSON 响应。
func BindMultipartGroups[In any, Out any](statusCode int, fn BindFunc[In, Out],
	groups []string, options ...servletmultipart.Option) arkweb.Handler {
	return bindMultipart(statusCode, fn, groups, options...)
}

// OptionalRequestBody 可选绑定请求体；请求体不存在时返回 present=false。
func OptionalRequestBody[T any](ctx *arkweb.Context) (T, bool, error) {
	return OptionalRequestBodyWithMediaTypes[T](ctx)
}

// RequestEntity 将请求体按 Content-Type 绑定，并返回请求实体元数据快照。
func RequestEntity[T any](ctx *arkweb.Context) (goweb.RequestEntity[T], error) {
	return requestEntity[T](ctx, nil, nil)
}

type responseStatusResult struct {
	statusCode int
	result     arkweb.Result
}

func controllerReturnValue(ctx *arkweb.Context, statusCode int, value any) arkweb.Result {
	return returnValueByKind(ctx, statusCode, value, ControllerKindFromContext(ctx))
}

// ExceptionPredicate 判断当前错误是否应由异常处理器处理。
type ExceptionPredicate func(err error) bool

// ExceptionEntityFunc 将指定错误转换为响应实体。
type ExceptionEntityFunc[E error, T any] func(ctx *arkweb.Context, err E) goweb.ResponseEntity[T]
