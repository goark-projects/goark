package view

import "errors"

var (
	// ErrNilResolver 表示视图结果没有可用解析器。
	ErrNilResolver = errors.New("goark/web/mvc/view: resolver is nil")
	// ErrNilView 表示解析器返回了空视图。
	ErrNilView = errors.New("goark/web/mvc/view: view is nil")
	// ErrViewNotFound 表示视图名没有匹配到模板或视图。
	ErrViewNotFound = errors.New("goark/web/mvc/view: view not found")
	// ErrInvalidViewName 表示视图名不满足安全路径约束。
	ErrInvalidViewName = errors.New("goark/web/mvc/view: invalid view name")
	// ErrNoTemplates 表示模板文件系统中没有可解析模板。
	ErrNoTemplates = errors.New("goark/web/mvc/view: no templates found")
)
