package flash

import "errors"

var (
	// ErrNilAccessor 表示 Flash 过滤器缺少 Session 访问器。
	ErrNilAccessor = errors.New("goark/web/mvc/flash: session accessor is nil")
)
