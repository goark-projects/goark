package mvc

import (
	"goark.dev/arkarta/servlet"
	arkweb "goark.dev/arkarta/web"
	goweb "goark.dev/goark/web"
	"goark.dev/goark/web/message"
)

// RequestEntity 将请求体按 Content-Type 绑定，并返回请求实体元数据快照。
func RequestEntity[T any](ctx *arkweb.Context) (goweb.RequestEntity[T], error) {
	return requestEntity[T](ctx, nil, nil)
}

// RequestEntityWithMediaTypes 将请求体按指定媒体类型集合绑定，并返回请求实体元数据快照。
func RequestEntityWithMediaTypes[T any](ctx *arkweb.Context, mediaTypes ...string) (goweb.RequestEntity[T], error) {
	return requestEntity[T](ctx, nil, cleanRouteValues(mediaTypes))
}

// ValidatedRequestEntity 将请求体绑定到目标类型，并按可选分组执行结构体验证。
func ValidatedRequestEntity[T any](ctx *arkweb.Context, groups ...string) (goweb.RequestEntity[T], error) {
	return requestEntity[T](ctx, cloneValidationGroups(groups), nil)
}

// ValidatedRequestEntityWithMediaTypes 将请求体按指定媒体类型集合绑定，并执行结构体验证。
func ValidatedRequestEntityWithMediaTypes[T any](ctx *arkweb.Context, mediaTypes []string, groups ...string) (goweb.RequestEntity[T], error) {
	return requestEntity[T](ctx, cloneValidationGroups(groups), cleanRouteValues(mediaTypes))
}

func requestEntity[T any](ctx *arkweb.Context, groups []string, mediaTypes []string) (goweb.RequestEntity[T], error) {
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
		Headers:       request.Header(),
		ContentLength: request.ContentLength(),
	}, requestEntityHasBody(request), nil
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
