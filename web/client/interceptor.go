package client

import (
	"context"
	"net/http"
)

// ExchangeFunc 执行一次原始 HTTP 交换。
type ExchangeFunc func(context.Context, *http.Request) (*http.Response, error)

// Interceptor 在请求发送前后扩展客户端行为。
type Interceptor interface {
	Intercept(ctx context.Context, req *http.Request, next ExchangeFunc) (*http.Response, error)
}

// InterceptorFunc 将函数适配为请求拦截器。
type InterceptorFunc func(context.Context, *http.Request, ExchangeFunc) (*http.Response, error)

// Intercept 执行函数型拦截器。
func (f InterceptorFunc) Intercept(ctx context.Context, req *http.Request, next ExchangeFunc) (*http.Response, error) {
	if f == nil {
		return next(ctx, req)
	}
	return f(ctx, req, next)
}

func (c *Client) exchangeFunc() ExchangeFunc {
	final := func(_ context.Context, req *http.Request) (*http.Response, error) {
		return c.httpClient.Do(req)
	}
	next := final
	for i := len(c.interceptors) - 1; i >= 0; i-- {
		interceptor := c.interceptors[i]
		if interceptor == nil {
			continue
		}
		currentInterceptor := interceptor
		currentNext := next
		next = func(ctx context.Context, req *http.Request) (*http.Response, error) {
			return currentInterceptor.Intercept(ctx, req, currentNext)
		}
	}
	return next
}
