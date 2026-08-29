package mvc

import "errors"

var (
	// ErrNilModelAttributeInitializer 表示模型初始化器为空。
	ErrNilModelAttributeInitializer = errors.New("goark/web/mvc: model attribute initializer is nil")
	// ErrInvalidModelAttributeName 表示模型属性名非法。
	ErrInvalidModelAttributeName = errors.New("goark/web/mvc: invalid model attribute name")
)
