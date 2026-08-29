package mvc

import (
	arkweb "goark.dev/arkarta/web"
	"goark.dev/goark/core/convert"
)

// DataBinder 封装控制器本地的数据绑定扩展点。
type DataBinder struct {
	conversionService *convert.Service
}

func newDataBinder(service *convert.Service) *DataBinder {
	if service == nil {
		service = DefaultConversionService()
	}
	return &DataBinder{conversionService: service}
}

// AddConverter 向当前请求作用域的转换服务注册转换器。
func (b *DataBinder) AddConverter(converter Converter) error {
	if b == nil || b.conversionService == nil {
		return ErrNilDataBinder
	}
	return b.conversionService.Register(converter)
}

// ConversionService 返回当前请求作用域的转换服务。
func (b *DataBinder) ConversionService() *convert.Service {
	if b == nil || b.conversionService == nil {
		return DefaultConversionService()
	}
	return b.conversionService
}

// BinderInitializer 表示控制器级 @InitBinder 的 Go 化初始化器。
type BinderInitializer interface {
	InitializeBinder(ctx *arkweb.Context, binder *DataBinder) error
}

// BinderInitializerFunc 将函数适配为 BinderInitializer。
type BinderInitializerFunc func(ctx *arkweb.Context, binder *DataBinder) error

// InitializeBinder 执行绑定器初始化函数。
func (f BinderInitializerFunc) InitializeBinder(ctx *arkweb.Context, binder *DataBinder) error {
	if f == nil {
		return ErrNilBinderInitializer
	}
	return f(ctx, binder)
}
