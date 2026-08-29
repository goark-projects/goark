package mvc

import (
	"net/http"

	arkweb "goark.dev/arkarta/web"
	goweb "goark.dev/goark/web"
	"goark.dev/goark/web/mvc/view"
)

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

func controllerReturnValue(ctx *arkweb.Context, statusCode int, value any) arkweb.Result {
	return returnValueByKind(ctx, statusCode, value, ControllerKindFromContext(ctx))
}

func adviceReturnValue(ctx *arkweb.Context, statusCode int, value any) arkweb.Result {
	return returnValueByKind(ctx, statusCode, value, ControllerAdviceKindFromContext(ctx))
}

func returnValueByKind(ctx *arkweb.Context, statusCode int, value any, kind ControllerKind) arkweb.Result {
	switch typed := value.(type) {
	case ModelAndView:
		return typed
	case *ModelAndView:
		if typed == nil {
			return responseBodyResult(ctx, statusCode, nil)
		}
		return *typed
	case Model:
		return implicitModelResult(ctx, statusCode, typed)
	case *Model:
		if typed == nil {
			return responseBodyResult(ctx, statusCode, nil)
		}
		return implicitModelResult(ctx, statusCode, *typed)
	}
	if kind == ControllerKindREST {
		return responseBodyResult(ctx, statusCode, value)
	}
	if name, ok := value.(string); ok {
		return view.Render(name, nil, view.WithStatus(resolveResponseStatus(ctx, statusCode, http.StatusOK)))
	}
	return responseBodyResult(ctx, statusCode, value)
}

func responseBodyResult(ctx *arkweb.Context, statusCode int, value any) arkweb.Result {
	statusCode = resolveResponseStatus(ctx, statusCode, http.StatusOK)
	if mediaType, ok := selectedProducesMediaType(ctx); ok {
		return goweb.Message(statusCode, value, mediaType)
	}
	return goweb.Message(statusCode, value)
}
