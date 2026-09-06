package mvc

import (
	"errors"
	"mime"
	"net/http"
	"strings"

	servletmultipart "goark.dev/arkarta/servlet/multipart"

	arkweb "goark.dev/arkarta/web"
	goweb "goark.dev/goark/web"

	"goark.dev/arkarta/servlet"
	"goark.dev/goark/web/message"
)

// RequestPartsByName 返回 multipart/form-data 中指定字段名的全部文件段。
func RequestPartsByName(ctx *arkweb.Context, name string, options ...ParamOption) (
	[]servletmultipart.Part, error) {
	if ctx == nil || ctx.Request() == nil {
		return nil, arkweb.ErrNilContext
	}
	paramOptions := newParamOptions(ctx, options)
	parts, err := servletmultipart.Parts(ctx.Request(), servletmultipart.NewParser())
	if err != nil {
		return nil, err
	}
	matched := make([]servletmultipart.Part, 0, len(parts))
	for _, part := range parts {
		if part.Name() == name {
			matched = append(matched, part)
		}
	}
	if len(matched) > 0 {
		return matched, nil
	}
	if paramOptions.required {
		return nil, missingParameterError("请求Part", name)
	}
	return nil, nil
}

func requestPart(ctx *arkweb.Context, name string, options ...ParamOption) (
	servletmultipart.Part, bool, error) {
	var zero servletmultipart.Part
	if ctx == nil || ctx.Request() == nil {
		return zero, false, arkweb.ErrNilContext
	}
	paramOptions := newParamOptions(ctx, options)
	parser := servletmultipart.NewParser()
	part, ok, err := servletmultipart.RequestPart(ctx.Request(), name, parser)
	if err != nil {
		return zero, false, err
	}
	if ok {
		return part, true, nil
	}
	if paramOptions.required {
		return zero, false, missingParameterError("请求Part", name)
	}
	return zero, false, nil
}

func ensureJSONPart(contentType string) error {
	contentType = strings.TrimSpace(contentType)
	if contentType == "" {
		return nil
	}
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return arkweb.ErrUnsupportedMediaType
	}
	parts := strings.SplitN(strings.ToLower(mediaType), "/", 2)
	if len(parts) != 2 {
		return arkweb.ErrUnsupportedMediaType
	}
	if parts[0] == "application" && (parts[1] == "json" || strings.HasSuffix(parts[1], "+json")) {
		return nil
	}
	return arkweb.ErrUnsupportedMediaType
}

func bindResponseStatus(ctx *arkweb.Context, statusCode int) func() {
	if ctx == nil || ctx.Request() == nil {
		return func() {}
	}
	request := ctx.Request()
	previous, existed := request.Attribute(AttributeResponseStatus)
	request.SetAttribute(AttributeResponseStatus, statusCode)
	return func() {
		if existed {
			request.SetAttribute(AttributeResponseStatus, previous)
			return
		}
		request.SetAttribute(AttributeResponseStatus, nil)
	}
}

func requestEntityMetadata(ctx *arkweb.Context) (goweb.RequestMetadata, bool, error) {
	if ctx == nil || ctx.Request() == nil {
		return goweb.RequestMetadata{}, false, arkweb.ErrNilContext
	}
	request := ctx.Request()
	return goweb.RequestMetadata{
		Method:        request.Method(),
		URL:           requestEntityURL(request),
		RequestURI:    request.RequestURI(),
		Path:          request.Path(),
		Headers:       snapshotRequestHeaders(request.Header()),
		ContentLength: request.ContentLength(),
	}, requestEntityHasBody(request), nil
}

func responseStatusFromContext(ctx *arkweb.Context) (int, bool) {
	if ctx == nil || ctx.Request() == nil {
		return 0, false
	}
	value, ok := ctx.Request().Attribute(AttributeResponseStatus)
	if !ok {
		return 0, false
	}
	statusCode, ok := value.(int)
	if !ok {
		return 0, false
	}
	return normalizeResponseStatus(statusCode, http.StatusOK), true
}

// ExceptionHandlerAs 使用 errors.As 匹配指定错误类型。
func ExceptionHandlerAs[E error](fn ExceptionResultFunc[E]) goweb.ErrorMapper {
	return ExceptionHandler(func(ctx *arkweb.Context, err error) (arkweb.Result, bool) {
		if fn == nil || err == nil {
			return nil, false
		}
		var target E
		if !errors.As(err, &target) {
			return nil, false
		}
		return fn(ctx, target), true
	})
}

func requestEntityHasBody(request *servlet.Request) bool {
	if request == nil || request.Body() == nil {
		return false
	}
	if request.ContentLength() != 0 {
		return true
	}
	httpRequest := request.HTTPRequest()
	return httpRequest != nil && len(httpRequest.TransferEncoding) > 0
}

// ExceptionResponseBodyAs 强制将普通异常返回值写为响应体。
func ExceptionResponseBodyAs[E error, T any](statusCode int, fn ExceptionValueFunc[E,
	T]) goweb.ErrorMapper {
	return ExceptionHandlerAs(func(ctx *arkweb.Context, err E) arkweb.Result {
		if fn == nil {
			return nil
		}
		return responseBodyResult(ctx, statusCode, fn(ctx, err))
	})
}

// ResponseBody 将普通返回值通过消息转换器写为响应体。
func ResponseBody[T any](statusCode int, fn ValueFunc[T]) arkweb.Handler {
	return arkweb.HandlerFunc(func(ctx *arkweb.Context) (arkweb.Result, error) {
		value, err := fn(ctx)
		if err != nil {
			return nil, err
		}
		return responseBodyResult(ctx, statusCode, value), nil
	})
}

// ValidatedRequestBodyWithMediaTypes 将请求体按指定媒体类型集合绑定，并执行结构体验证。
func ValidatedRequestBodyWithMediaTypes[T any](ctx *arkweb.Context, mediaTypes []string,
	groups ...string) (T, error) {
	var out T
	if err := bindAndValidateBody(ctx, &out, groups, mediaTypes); err != nil {
		return out, err
	}
	return out, nil
}

func resolveResponseStatus(ctx *arkweb.Context, statusCode int, fallback int) int {
	if statusCode != 0 {
		return normalizeResponseStatus(statusCode, fallback)
	}
	if statusCode, ok := responseStatusFromContext(ctx); ok {
		return statusCode
	}
	return normalizeResponseStatus(fallback, http.StatusOK)
}

// RequestBody 将请求体按 Content-Type 绑定到目标类型，不执行结构体验证。
func RequestBody[T any](ctx *arkweb.Context) (T, error) {
	var out T
	if err := message.ReaderFromContext(ctx).Read(ctx, &out); err != nil {
		return out, err
	}
	return out, nil
}

// ValidatedRequestBody 将请求体绑定到目标类型，并按可选分组执行结构体验证。
func ValidatedRequestBody[T any](ctx *arkweb.Context, groups ...string) (T, error) {
	var out T
	if err := bindAndValidateBody(ctx, &out, groups, nil); err != nil {
		return out, err
	}
	return out, nil
}

func validateBound(ctx *arkweb.Context, target any, groups []string) error {
	result, err := validateBindingResult(ctx, target, groups)
	if err != nil {
		return err
	}
	return result.Err()
}

// BindRequestEntityWithMediaTypes 按指定媒体类型集合绑定请求实体，再将返回值写为 JSON 响应。
func BindRequestEntityWithMediaTypes[In any, Out any](statusCode int,
	fn BindRequestEntityFunc[In, Out], mediaTypes ...string) arkweb.Handler {
	return bindRequestEntity(statusCode, fn, nil, mediaTypes)
}

// RequestEntityWithMediaTypes 将请求体按指定媒体类型集合绑定，并返回请求实体元数据快照。
func RequestEntityWithMediaTypes[T any](ctx *arkweb.Context, mediaTypes ...string) (
	goweb.RequestEntity[T], error) {
	return requestEntity[T](ctx, nil, cleanRouteValues(mediaTypes))
}

// ValidatedRequestPartJSON 将指定 multipart 文件段按 JSON 绑定到目标类型，并按可选分组执行结构体验证。
func ValidatedRequestPartJSON[T any](ctx *arkweb.Context, name string, groups []string,
	options ...ParamOption) (T, error) {
	return requestPartJSON[T](ctx, name, groups, true, options...)
}

// BindBodyGroups 绑定请求体，并按显式校验分组写出 JSON 响应。
func BindBodyGroups[In any, Out any](statusCode int, fn BindFunc[In, Out],
	groups ...string) arkweb.Handler {
	return bindBody(statusCode, fn, groups, nil)
}

// BindRequestEntityGroups 绑定请求实体，并按显式校验分组写出 JSON 响应。
func BindRequestEntityGroups[In any, Out any](statusCode int, fn BindRequestEntityFunc[In,
	Out], groups ...string) arkweb.Handler {
	return bindRequestEntity(statusCode, fn, groups, nil)
}

// BindMultipartResultGroups 绑定 multipart/form-data 请求体，并按显式校验分组传入 BindingResult。
func BindMultipartResultGroups[In any, Out any](statusCode int, fn BindResultFunc[In, Out],
	groups []string, options ...servletmultipart.Option) arkweb.Handler {
	return bindMultipartResult(statusCode, fn, groups, options...)
}

// OptionalValidatedRequestBody 可选绑定请求体；请求体存在时执行结构体验证。
func OptionalValidatedRequestBody[T any](ctx *arkweb.Context, groups ...string) (T, bool, error) {
	return OptionalValidatedRequestBodyWithMediaTypes[T](ctx, nil, groups...)
}

const (
	// AttributeResponseStatus 保存 MVC 方法级默认响应状态码。
	AttributeResponseStatus = "goark.web.mvc.response.status"
)

// BindRequestEntityFunc 表示绑定请求实体后返回普通值的处理函数。
type BindRequestEntityFunc[In any, Out any] func(ctx *arkweb.Context,
	input goweb.RequestEntity[In]) (Out, error)

// BindRequestEntityResponseFunc 表示绑定请求实体后返回响应实体的处理函数。
type BindRequestEntityResponseFunc[In any, Out any] func(ctx *arkweb.Context,
	input goweb.RequestEntity[In]) (goweb.ResponseEntity[Out], error)

// ExceptionFunc 表示可决定是否处理当前错误的 MVC 异常函数。
type ExceptionFunc func(ctx *arkweb.Context, err error) (arkweb.Result, bool)

// ExceptionValueFunc 将指定错误转换为普通返回值。
type ExceptionValueFunc[E error, T any] func(ctx *arkweb.Context, err E) T
