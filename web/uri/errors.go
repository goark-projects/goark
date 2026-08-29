package uri

import "errors"

var (
	// ErrInvalidURI 表示 URI 模板或基础地址非法。
	ErrInvalidURI = errors.New("goark/web/uri: invalid uri")
	// ErrMissingPathVariable 表示 URI 模板变量缺少替换值。
	ErrMissingPathVariable = errors.New("goark/web/uri: missing path variable")
)
