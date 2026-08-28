package web

import (
	"context"

	"goark.dev/arkarta/servlet"
	"goark.dev/goark/container"
)

// Filter 表示 Arkarta Servlet 过滤器。
type Filter = servlet.Filter

// FilterFunc 将普通函数适配为 Filter。
type FilterFunc = servlet.FilterFunc

// RegisterFilter 注册 Servlet 过滤器贡献点。
func RegisterFilter(registry *container.Registry, name string, filter servlet.Filter, options ...container.Option) error {
	if isNilFilter(filter) {
		return ErrNilFilter
	}
	return RegisterConfigurer(registry, name, ConfigurerFunc(func(ctx context.Context, webRegistry *Registry) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if webRegistry == nil {
			return ErrNilRegistry
		}
		webRegistry.AddFilter(filter)
		return nil
	}), options...)
}

func isNilFilter(filter servlet.Filter) bool {
	return isNilWebValue(filter)
}
