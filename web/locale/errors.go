package locale

import "errors"

var (
	// ErrInvalidLocale 表示 Locale 为空或未经过标准化构造。
	ErrInvalidLocale = errors.New("goark/web/locale: invalid locale")
	// ErrNilResolver 表示 Locale 解析器为空。
	ErrNilResolver = errors.New("goark/web/locale: resolver is nil")
	// ErrInvalidParameterName 表示 Locale 切换参数名非法。
	ErrInvalidParameterName = errors.New("goark/web/locale: invalid parameter name")
	// ErrInvalidHTTPMethod 表示 Locale 切换 HTTP 方法非法。
	ErrInvalidHTTPMethod = errors.New("goark/web/locale: invalid http method")
)
