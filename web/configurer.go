package web

import (
	"context"

	"goark.dev/goark/container"
)

// Configurer 定义 Web 模块对 Router 和 Deployment 的贡献点。
type Configurer interface {
	ConfigureWeb(ctx context.Context, registry *Registry) error
}

// ConfigurerFunc 将函数适配为 Web 配置器。
type ConfigurerFunc func(ctx context.Context, registry *Registry) error

// ConfigureWeb 执行函数型配置器。
func (f ConfigurerFunc) ConfigureWeb(ctx context.Context, registry *Registry) error {
	if f == nil {
		return ErrNilConfigurer
	}
	return f(ctx, registry)
}

// RegisterConfigurer 注册 Web 配置器 Bean。
func RegisterConfigurer(registry *container.Registry, name string, configurer Configurer, options ...container.Option) error {
	return container.RegisterInstance[Configurer](registry, name, configurer, options...)
}

// ApplyConfigurers 按容器顺序执行所有 Web 配置器。
func ApplyConfigurers(ctx context.Context, resolver container.Resolver, registry *Registry) error {
	if registry == nil {
		return ErrNilRegistry
	}
	configurers, err := container.GetAllByType[Configurer](ctx, resolver)
	if err != nil {
		return err
	}
	for _, configurer := range configurers {
		if configurer == nil {
			continue
		}
		if err := configurer.ConfigureWeb(ctx, registry); err != nil {
			return err
		}
	}
	return nil
}
