package mvc

import (
	arkweb "goark.dev/arkarta/web"
	"goark.dev/goark/core/util"
)

// PreHandleFunc 在处理器执行前决定请求是否继续。
type PreHandleFunc func(ctx *arkweb.Context) (bool, error)

// PostHandleFunc 在处理器成功返回后调整响应结果。
type PostHandleFunc func(ctx *arkweb.Context, result arkweb.Result) (arkweb.Result, error)

// AfterCompletionFunc 在处理器链完成后接收处理错误。
type AfterCompletionFunc func(ctx *arkweb.Context, err error)

// HandlerInterceptor 提供 Spring MVC HandlerInterceptor 的 Go 化生命周期。
type HandlerInterceptor interface {
	PreHandle(ctx *arkweb.Context) (bool, error)
	PostHandle(ctx *arkweb.Context, result arkweb.Result) (arkweb.Result, error)
	AfterCompletion(ctx *arkweb.Context, err error)
}

// HandlerInterceptorFuncs 用函数组快速声明 HandlerInterceptor。
type HandlerInterceptorFuncs struct {
	PreHandleFunc       PreHandleFunc
	PostHandleFunc      PostHandleFunc
	AfterCompletionFunc AfterCompletionFunc
}

// NewHandlerInterceptor 创建函数型 HandlerInterceptor。
func NewHandlerInterceptor(pre PreHandleFunc, post PostHandleFunc, after AfterCompletionFunc) HandlerInterceptor {
	return HandlerInterceptorFuncs{
		PreHandleFunc:       pre,
		PostHandleFunc:      post,
		AfterCompletionFunc: after,
	}
}

// PreHandle 执行前置处理；未设置时默认继续。
func (f HandlerInterceptorFuncs) PreHandle(ctx *arkweb.Context) (bool, error) {
	if f.PreHandleFunc == nil {
		return true, nil
	}
	return f.PreHandleFunc(ctx)
}

// PostHandle 执行后置处理；未设置时保留原响应结果。
func (f HandlerInterceptorFuncs) PostHandle(ctx *arkweb.Context, result arkweb.Result) (arkweb.Result, error) {
	if f.PostHandleFunc == nil {
		return result, nil
	}
	return f.PostHandleFunc(ctx, result)
}

// AfterCompletion 执行完成回调；未设置时无操作。
func (f HandlerInterceptorFuncs) AfterCompletion(ctx *arkweb.Context, err error) {
	if f.AfterCompletionFunc != nil {
		f.AfterCompletionFunc(ctx, err)
	}
}

// HandlerInterceptorAdapter 将 MVC HandlerInterceptor 适配为底层 Web Interceptor。
func HandlerInterceptorAdapter(interceptor HandlerInterceptor) arkweb.Interceptor {
	if util.IsNil(interceptor) {
		return nil
	}
	return arkweb.InterceptorFunc(func(ctx *arkweb.Context, next arkweb.Handler) (result arkweb.Result, err error) {
		proceed, err := interceptor.PreHandle(ctx)
		if err != nil || !proceed {
			return nil, err
		}
		defer func() {
			interceptor.AfterCompletion(ctx, err)
		}()
		result, err = next.Handle(ctx)
		if err != nil {
			return nil, err
		}
		return interceptor.PostHandle(ctx, result)
	})
}
