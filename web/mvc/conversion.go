package mvc

import (
	"context"
	"fmt"

	arkweb "goark.dev/arkarta/web"
	"goark.dev/goark/container"
	"goark.dev/goark/core/convert"
	"goark.dev/goark/core/lang"
	goweb "goark.dev/goark/web"
)

const (
	// AttributeConversionService 是请求属性中保存 MVC 转换服务的键。
	AttributeConversionService = "goark.dev/goark/web/mvc.conversionService"
)

var defaultMVCConversionService = convert.DefaultService()

// Converter 描述 MVC 参数绑定可使用的类型转换器。
type Converter = convert.Converter

// ConverterFunc 将类型安全函数适配为 MVC 转换器。
type ConverterFunc[S any, T any] = convert.ConverterFunc[S, T]

// DefaultConversionService 创建 MVC 默认转换服务。
func DefaultConversionService() *convert.Service {
	return defaultMVCConversionService
}

// BindConversionService 将 MVC 转换服务绑定到当前请求上下文。
func BindConversionService(ctx *arkweb.Context, service *convert.Service) error {
	if ctx == nil || ctx.Request() == nil {
		return arkweb.ErrNilContext
	}
	if service == nil {
		service = DefaultConversionService()
	}
	ctx.Request().SetAttribute(AttributeConversionService, service)
	return nil
}

// ConversionServiceFromContext 返回当前请求绑定的转换服务；未绑定时返回默认服务。
func ConversionServiceFromContext(ctx *arkweb.Context) *convert.Service {
	if ctx == nil || ctx.Request() == nil {
		return DefaultConversionService()
	}
	value, ok := ctx.Request().Attribute(AttributeConversionService)
	if !ok {
		return DefaultConversionService()
	}
	service, ok := value.(*convert.Service)
	if !ok || service == nil {
		return DefaultConversionService()
	}
	return service
}

// ConversionInterceptor 在请求进入 MVC 处理前绑定转换服务。
func ConversionInterceptor(service *convert.Service) arkweb.Interceptor {
	if service == nil {
		service = DefaultConversionService()
	}
	return arkweb.InterceptorFunc(func(ctx *arkweb.Context, next arkweb.Handler) (arkweb.Result, error) {
		if ctx != nil && ctx.Request() != nil {
			ctx.Request().SetAttribute(AttributeConversionService, service)
		}
		return next.Handle(ctx)
	})
}

// RegisterConversionService 注册 MVC 转换服务贡献点。
func RegisterConversionService(registry *container.Registry, name string, service *convert.Service, options ...container.Option) error {
	return goweb.RegisterConfigurer(registry, name, goweb.ConfigurerFunc(func(ctx context.Context, webRegistry *goweb.Registry) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if webRegistry == nil {
			return goweb.ErrNilRegistry
		}
		webRegistry.Use(ConversionInterceptor(service))
		return nil
	}), options...)
}

// WithConversionService 设置单次参数绑定使用的转换服务。
func WithConversionService(service *convert.Service) ParamOption {
	return func(options *paramOptions) {
		if service != nil {
			options.conversionService = service
		}
	}
}

func convertParamValue[T any](options paramOptions) func(string) (T, error) {
	return func(value string) (T, error) {
		return convert.Convert[T](options.conversionService, value)
	}
}

func paramTargetType[T any]() string {
	return fmt.Sprint(lang.TypeOf[T]())
}
