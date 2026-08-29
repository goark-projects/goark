package client

import (
	"context"
	"net/http"
	"reflect"
)

// StatusPredicate 判断响应是否需要进入状态处理器。
type StatusPredicate func(*Response) bool

// StatusHandler 处理指定响应状态。
type StatusHandler interface {
	HandleStatus(ctx context.Context, response *Response) error
}

// StatusHandlerFunc 将函数适配为响应状态处理器。
type StatusHandlerFunc func(context.Context, *Response) error

// HandleStatus 执行函数型响应状态处理器。
func (f StatusHandlerFunc) HandleStatus(ctx context.Context, response *Response) error {
	if f == nil {
		return ErrInvalidStatusHandler
	}
	return f(ctx, response)
}

type statusHandler struct {
	predicate StatusPredicate
	handler   StatusHandler
}

// StatusCode 匹配指定 HTTP 状态码。
func StatusCode(code int) StatusPredicate {
	return func(response *Response) bool {
		return response != nil && response.StatusCode() == code
	}
}

// StatusRange 匹配半开状态码区间 [min, max)。
func StatusRange(min int, max int) StatusPredicate {
	return func(response *Response) bool {
		if response == nil {
			return false
		}
		code := response.StatusCode()
		return code >= min && code < max
	}
}

// Is4xxStatus 判断响应是否为客户端错误。
func Is4xxStatus(response *Response) bool {
	return response != nil && response.StatusCode() >= http.StatusBadRequest && response.StatusCode() < http.StatusInternalServerError
}

// Is5xxStatus 判断响应是否为服务端错误。
func Is5xxStatus(response *Response) bool {
	return response != nil && response.StatusCode() >= http.StatusInternalServerError && response.StatusCode() < 600
}

// IsErrorStatus 判断响应是否为 HTTP 错误状态。
func IsErrorStatus(response *Response) bool {
	return Is4xxStatus(response) || Is5xxStatus(response)
}

// RaiseStatusError 将匹配响应转换为 StatusError。
func RaiseStatusError(ctx context.Context, response *Response) error {
	if ctx != nil {
		if err := ctx.Err(); err != nil {
			return err
		}
	}
	if response == nil {
		return ErrNilHTTPResponse
	}
	return response.EnsureSuccess()
}

func newStatusHandler(predicate StatusPredicate, handler StatusHandler) (statusHandler, error) {
	if predicate == nil || isNilStatusHandler(handler) {
		return statusHandler{}, ErrInvalidStatusHandler
	}
	return statusHandler{
		predicate: predicate,
		handler:   handler,
	}, nil
}

func isNilStatusHandler(handler StatusHandler) bool {
	if handler == nil {
		return true
	}
	value := reflect.ValueOf(handler)
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return value.IsNil()
	default:
		return false
	}
}

func applyStatusHandlers(ctx context.Context, response *Response, handlerGroups ...[]statusHandler) error {
	if response == nil {
		return ErrNilHTTPResponse
	}
	for _, handlers := range handlerGroups {
		for _, handler := range handlers {
			if handler.predicate == nil || handler.handler == nil || !handler.predicate(response) {
				continue
			}
			if err := handler.handler.HandleStatus(ctx, response); err != nil {
				return err
			}
		}
	}
	return nil
}
