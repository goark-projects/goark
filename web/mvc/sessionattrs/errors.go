package sessionattrs

import "errors"

var (
	// ErrNilAccessor 表示 Session 访问器为空。
	ErrNilAccessor = errors.New("goark/web/mvc/sessionattrs: session accessor is nil")
)
