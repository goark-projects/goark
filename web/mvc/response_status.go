package mvc

import (
	"net/http"

	arkweb "goark.dev/arkarta/web"
)

const (
	// AttributeResponseStatus 保存 MVC 方法级默认响应状态码。
	AttributeResponseStatus = "goark.web.mvc.response.status"
)

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

type responseStatusResult struct {
	statusCode int
	result     arkweb.Result
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

func resolveResponseStatus(ctx *arkweb.Context, statusCode int, fallback int) int {
	if statusCode != 0 {
		return normalizeResponseStatus(statusCode, fallback)
	}
	if statusCode, ok := responseStatusFromContext(ctx); ok {
		return statusCode
	}
	return normalizeResponseStatus(fallback, http.StatusOK)
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
