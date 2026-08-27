package mvc

import (
	"context"

	goweb "goark.dev/goark/web"
)

// Configurer 将 MVC 控制器注册到 Web 注册表。
type Configurer struct {
	controllers []Controller
}

// NewConfigurer 创建 MVC Web 配置器。
func NewConfigurer(controllers ...Controller) Configurer {
	return Configurer{controllers: append([]Controller(nil), controllers...)}
}

// ConfigureWeb 注册控制器路由。
func (c Configurer) ConfigureWeb(ctx context.Context, registry *goweb.Registry) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	for _, controller := range c.controllers {
		if err := controller.Register(registry); err != nil {
			return err
		}
	}
	return nil
}
