package webtest

import "errors"

var (
	// ErrNilHTTPHandler 表示测试客户端缺少 HTTP 处理器。
	ErrNilHTTPHandler = errors.New("goark/web/test: http handler is nil")
	// ErrNilServletHandler 表示测试客户端缺少 Servlet 处理器。
	ErrNilServletHandler = errors.New("goark/web/test: servlet handler is nil")
	// ErrNilRouter 表示测试客户端缺少 Web Router。
	ErrNilRouter = errors.New("goark/web/test: router is nil")
	// ErrNilRequest 表示待执行请求为空。
	ErrNilRequest = errors.New("goark/web/test: request is nil")
	// ErrInvalidHeader 表示请求头名称或值非法。
	ErrInvalidHeader = errors.New("goark/web/test: invalid header")
	// ErrNilResponse 表示响应断言缺少响应对象。
	ErrNilResponse = errors.New("goark/web/test: response is nil")
	// ErrNilJSONTarget 表示 JSON 解码目标为空。
	ErrNilJSONTarget = errors.New("goark/web/test: json target is nil")
	// ErrInvalidJSONPath 表示 JSON 路径表达式非法。
	ErrInvalidJSONPath = errors.New("goark/web/test: invalid json path")
)
