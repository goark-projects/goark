package web

import (
	"context"

	arkweb "goark.dev/arkarta/web"
	"goark.dev/goark/container"
)

// ErrorMapper 将处理错误映射为稳定的 Web 响应。
type ErrorMapper = arkweb.ErrorMapper

// ErrorMapperFunc 将普通函数适配为 ErrorMapper。
type ErrorMapperFunc = arkweb.ErrorMapperFunc

// ErrorMapperChain 按注册顺序组合多个错误映射器。
type ErrorMapperChain struct {
	mappers  []arkweb.ErrorMapper
	fallback arkweb.ErrorMapper
}

// NewErrorMapperChain 创建错误映射器链；未命中时回退到 Arkarta 默认映射器。
func NewErrorMapperChain(mappers ...arkweb.ErrorMapper) ErrorMapperChain {
	return newErrorMapperChain(nil, mappers)
}

func newErrorMapperChain(fallback arkweb.ErrorMapper, mappers []arkweb.ErrorMapper) ErrorMapperChain {
	chain := ErrorMapperChain{fallback: fallback}
	for _, mapper := range mappers {
		if isNilErrorMapper(mapper) {
			continue
		}
		chain.mappers = append(chain.mappers, mapper)
	}
	return chain
}

// MapError 按注册顺序返回第一个非空映射结果。
func (c ErrorMapperChain) MapError(ctx *arkweb.Context, err error) arkweb.Result {
	for _, mapper := range c.mappers {
		if isNilErrorMapper(mapper) {
			continue
		}
		if result := mapper.MapError(ctx, err); result != nil {
			return result
		}
	}
	if !isNilErrorMapper(c.fallback) {
		return c.fallback.MapError(ctx, err)
	}
	return arkweb.DefaultErrorMapper{}.MapError(ctx, err)
}

// ErrorMappers 返回错误映射器快照。
func (c ErrorMapperChain) ErrorMappers() []arkweb.ErrorMapper {
	return append([]arkweb.ErrorMapper(nil), c.mappers...)
}

// RegisterErrorMapper 注册 Web 错误映射器贡献点。
func RegisterErrorMapper(registry *container.Registry, name string, mapper arkweb.ErrorMapper, options ...container.Option) error {
	if isNilErrorMapper(mapper) {
		return ErrNilErrorMapper
	}
	return RegisterConfigurer(registry, name, ConfigurerFunc(func(ctx context.Context, webRegistry *Registry) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if webRegistry == nil {
			return ErrNilRegistry
		}
		webRegistry.UseErrorMapper(mapper)
		return nil
	}), options...)
}

// RegisterFallbackErrorMapper 注册仅在普通错误映射器均未命中时执行的兜底映射器。
func RegisterFallbackErrorMapper(registry *container.Registry, name string, mapper arkweb.ErrorMapper, options ...container.Option) error {
	if isNilErrorMapper(mapper) {
		return ErrNilErrorMapper
	}
	return RegisterConfigurer(registry, name, ConfigurerFunc(func(ctx context.Context, webRegistry *Registry) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if webRegistry == nil {
			return ErrNilRegistry
		}
		webRegistry.UseFallbackErrorMapper(mapper)
		return nil
	}), options...)
}

func isNilErrorMapper(mapper arkweb.ErrorMapper) bool {
	return isNilWebValue(mapper)
}
