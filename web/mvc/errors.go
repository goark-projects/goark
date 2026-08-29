package mvc

import "errors"

var (
	// ErrNilModelAttributeInitializer 表示模型初始化器为空。
	ErrNilModelAttributeInitializer = errors.New("goark/web/mvc: model attribute initializer is nil")
	// ErrInvalidModelAttributeName 表示模型属性名非法。
	ErrInvalidModelAttributeName = errors.New("goark/web/mvc: invalid model attribute name")
	// ErrNilBinderInitializer 表示绑定器初始化器为空。
	ErrNilBinderInitializer = errors.New("goark/web/mvc: binder initializer is nil")
	// ErrNilDataBinder 表示数据绑定器为空。
	ErrNilDataBinder = errors.New("goark/web/mvc: data binder is nil")
	// ErrInvalidForwardLocation 表示 forward 目标路径非法。
	ErrInvalidForwardLocation = errors.New("goark/web/mvc: invalid forward location")
	// ErrForwardDispatcherUnavailable 表示当前请求无法获取 Servlet 分发器。
	ErrForwardDispatcherUnavailable = errors.New("goark/web/mvc: forward dispatcher unavailable")
)
