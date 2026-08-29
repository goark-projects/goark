package web

import (
	"context"

	"goark.dev/arkarta/servlet"
)

// FilterMapping 描述 Filter 的包含和排除路径模式。
type FilterMapping = InterceptorMapping

// NewFilterMapping 创建 Filter 路径映射；未设置包含模式时默认匹配全部路径。
func NewFilterMapping(options ...InterceptorMappingOption) (FilterMapping, error) {
	return NewInterceptorMapping(options...)
}

// WithFilterPathPatterns 设置需要过滤的路径模式，支持字面量、* 和 /** 后缀。
func WithFilterPathPatterns(patterns ...string) InterceptorMappingOption {
	return WithInterceptorPathPatterns(patterns...)
}

// WithFilterExcludePathPatterns 设置需要跳过的路径模式，支持字面量、* 和 /** 后缀。
func WithFilterExcludePathPatterns(patterns ...string) InterceptorMappingOption {
	return WithInterceptorExcludePathPatterns(patterns...)
}

type filterRegistration struct {
	filter  servlet.Filter
	mapping FilterMapping
}

func (r filterRegistration) Filter() servlet.Filter {
	if len(r.mapping.includes) == 0 && len(r.mapping.excludes) == 0 {
		return r.filter
	}
	return mappedFilter{
		target:  r.filter,
		mapping: r.mapping,
	}
}

type mappedFilter struct {
	target  servlet.Filter
	mapping FilterMapping
}

func (f mappedFilter) Filter(ctx context.Context, req *servlet.Request, res servlet.Response, chain servlet.Chain) error {
	requestPath := "/"
	if req != nil {
		requestPath = req.Path()
	}
	if !f.mapping.Matches(requestPath) {
		return chain.Next(ctx, req, res)
	}
	return f.target.Filter(ctx, req, res, chain)
}
