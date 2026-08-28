package web

import "errors"

var (
	// ErrNilConfigurer 表示 Web 配置器为空。
	ErrNilConfigurer = errors.New("goark/web: configurer is nil")
	// ErrNilRegistry 表示 Web 注册表为空。
	ErrNilRegistry = errors.New("goark/web: registry is nil")
	// ErrInvalidRoute 表示路由描述非法。
	ErrInvalidRoute = errors.New("goark/web: invalid route")
	// ErrNilErrorMapper 表示错误映射器为空。
	ErrNilErrorMapper = errors.New("goark/web: error mapper is nil")
)
