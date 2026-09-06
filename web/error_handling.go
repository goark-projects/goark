package web

import (
	"context"
	"errors"

	"goark.dev/arkarta/servlet"
	arkweb "goark.dev/arkarta/web"
	"goark.dev/goark/container"
)

var (
	// ErrNilConfigurer 表示 Web 配置器为空。
	ErrNilConfigurer = errors.New("goark/web: configurer is nil")
	// ErrNilRegistry 表示 Web 注册表为空。
	ErrNilRegistry = errors.New("goark/web: registry is nil")
	// ErrInvalidRoute 表示路由描述非法。
	ErrInvalidRoute = errors.New("goark/web: invalid route")
	// ErrNilErrorMapper 表示错误映射器为空。
	ErrNilErrorMapper = errors.New("goark/web: error mapper is nil")
	// ErrNilResponseAdvice 表示响应增强器为空。
	ErrNilResponseAdvice = errors.New("goark/web: response advice is nil")
	// ErrNilRequestBodyAdvice 表示请求体读取增强器为空。
	ErrNilRequestBodyAdvice = errors.New("goark/web: request body advice is nil")
	// ErrNilMessageConverter 表示消息转换器为空。
	ErrNilMessageConverter = errors.New("goark/web: message converter is nil")
	// ErrNilValidator 表示校验器为空。
	ErrNilValidator = errors.New("goark/web: validator is nil")
	// ErrNilInterceptor 表示 Web 拦截器为空。
	ErrNilInterceptor = errors.New("goark/web: interceptor is nil")
	// ErrInvalidInterceptorMapping 表示拦截器路径映射非法。
	ErrInvalidInterceptorMapping = errors.New("goark/web: invalid interceptor mapping")
	// ErrNilFilter 表示 Servlet 过滤器为空。
	ErrNilFilter = errors.New("goark/web: filter is nil")
	// ErrNilServlet 表示 Servlet 为空。
	ErrNilServlet = errors.New("goark/web: servlet is nil")
	// ErrNilDownloadReader 表示下载响应体为空。
	ErrNilDownloadReader = errors.New("goark/web: download reader is nil")
	// ErrInvalidRedirectLocation 表示重定向 Location 非法。
	ErrInvalidRedirectLocation = errors.New("goark/web: invalid redirect location")
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

func newErrorMapperChain(
	fallback arkweb.ErrorMapper,
	mappers []arkweb.ErrorMapper,
) ErrorMapperChain {
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
func RegisterErrorMapper(
	registry *container.Registry,
	name string,
	mapper arkweb.ErrorMapper,
	options ...container.Option,
) error {
	if isNilErrorMapper(mapper) {
		return ErrNilErrorMapper
	}
	return RegisterConfigurer(
		registry,
		name,
		ConfigurerFunc(func(ctx context.Context, webRegistry *Registry) error {
			if err := ctx.Err(); err != nil {
				return err
			}
			if webRegistry == nil {
				return ErrNilRegistry
			}
			webRegistry.UseErrorMapper(mapper)
			return nil
		}),
		options...)
}

// RegisterFallbackErrorMapper 注册仅在普通错误映射器均未命中时执行的兜底映射器。
func RegisterFallbackErrorMapper(
	registry *container.Registry,
	name string,
	mapper arkweb.ErrorMapper,
	options ...container.Option,
) error {
	if isNilErrorMapper(mapper) {
		return ErrNilErrorMapper
	}
	return RegisterConfigurer(
		registry,
		name,
		ConfigurerFunc(func(ctx context.Context, webRegistry *Registry) error {
			if err := ctx.Err(); err != nil {
				return err
			}
			if webRegistry == nil {
				return ErrNilRegistry
			}
			webRegistry.UseFallbackErrorMapper(mapper)
			return nil
		}),
		options...)
}

func isNilErrorMapper(mapper arkweb.ErrorMapper) bool {
	return isNilWebValue(mapper)
}

// StatusError 表示可由 Goark Web 映射为 HTTP 状态响应的错误。
type StatusError = servlet.StatusError

// ResponseStatusException 是 Spring ResponseStatusException 的 Go 等价类型。
type ResponseStatusException = servlet.HTTPError

// NewStatusError 创建带 HTTP 状态码和安全公开消息的错误。
func NewStatusError(statusCode int, publicMessage string, cause error) error {
	return servlet.NewHTTPError(statusCode, publicMessage, cause)
}

// NewResponseStatusException 创建可直接从处理器返回的 HTTP 状态异常。
func NewResponseStatusException(
	statusCode int,
	reason string,
	cause error,
) *ResponseStatusException {
	return servlet.NewHTTPError(statusCode, reason, cause)
}
