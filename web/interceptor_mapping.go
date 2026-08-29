package web

import (
	"path"
	"strings"

	arkweb "goark.dev/arkarta/web"
)

// InterceptorMapping 描述拦截器的包含和排除路径模式。
type InterceptorMapping struct {
	includes []string
	excludes []string
}

// InterceptorMappingOption 定制拦截器路径映射。
type InterceptorMappingOption func(*interceptorMappingConfig) error

type interceptorMappingConfig struct {
	includes []string
	excludes []string
}

// NewInterceptorMapping 创建路径映射；未设置包含模式时默认匹配全部路径。
func NewInterceptorMapping(options ...InterceptorMappingOption) (InterceptorMapping, error) {
	config := interceptorMappingConfig{}
	for _, option := range options {
		if option == nil {
			continue
		}
		if err := option(&config); err != nil {
			return InterceptorMapping{}, err
		}
	}
	return InterceptorMapping{
		includes: append([]string(nil), config.includes...),
		excludes: append([]string(nil), config.excludes...),
	}, nil
}

// WithInterceptorPathPatterns 设置需要拦截的路径模式，支持字面量、* 和 /** 后缀。
func WithInterceptorPathPatterns(patterns ...string) InterceptorMappingOption {
	return func(config *interceptorMappingConfig) error {
		cleaned, err := cleanInterceptorPatterns(patterns)
		if err != nil {
			return err
		}
		config.includes = append(config.includes, cleaned...)
		return nil
	}
}

// WithInterceptorExcludePathPatterns 设置需要跳过的路径模式，支持字面量、* 和 /** 后缀。
func WithInterceptorExcludePathPatterns(patterns ...string) InterceptorMappingOption {
	return func(config *interceptorMappingConfig) error {
		cleaned, err := cleanInterceptorPatterns(patterns)
		if err != nil {
			return err
		}
		config.excludes = append(config.excludes, cleaned...)
		return nil
	}
}

// Includes 返回包含路径模式副本。
func (m InterceptorMapping) Includes() []string {
	return append([]string(nil), m.includes...)
}

// Excludes 返回排除路径模式副本。
func (m InterceptorMapping) Excludes() []string {
	return append([]string(nil), m.excludes...)
}

// Matches 判断请求路径是否命中当前映射。
func (m InterceptorMapping) Matches(requestPath string) bool {
	requestPath = cleanInterceptorPath(requestPath)
	included := len(m.includes) == 0
	for _, pattern := range m.includes {
		if matchInterceptorPathPattern(pattern, requestPath) {
			included = true
			break
		}
	}
	if !included {
		return false
	}
	for _, pattern := range m.excludes {
		if matchInterceptorPathPattern(pattern, requestPath) {
			return false
		}
	}
	return true
}

type mappedInterceptor struct {
	target  arkweb.Interceptor
	mapping InterceptorMapping
}

func (i mappedInterceptor) Intercept(ctx *arkweb.Context, next arkweb.Handler) (arkweb.Result, error) {
	requestPath := "/"
	if ctx != nil && ctx.Request() != nil {
		requestPath = ctx.Request().Path()
	}
	if !i.mapping.Matches(requestPath) {
		return next.Handle(ctx)
	}
	return i.target.Intercept(ctx, next)
}

func cleanInterceptorPatterns(patterns []string) ([]string, error) {
	if len(patterns) == 0 {
		return nil, ErrInvalidInterceptorMapping
	}
	out := make([]string, 0, len(patterns))
	for _, pattern := range patterns {
		pattern = cleanInterceptorPath(pattern)
		if pattern == "" || !validInterceptorPattern(pattern) {
			return nil, ErrInvalidInterceptorMapping
		}
		out = append(out, pattern)
	}
	return out, nil
}

func cleanInterceptorPath(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if !strings.HasPrefix(value, "/") {
		value = "/" + value
	}
	return path.Clean(value)
}

func validInterceptorPattern(pattern string) bool {
	if strings.ContainsAny(pattern, "\x00\r\n") {
		return false
	}
	if strings.Contains(pattern, "**") && !strings.HasSuffix(pattern, "/**") {
		return false
	}
	if strings.HasSuffix(pattern, "/**") {
		prefix := strings.TrimSuffix(pattern, "/**")
		return prefix == "" || !strings.ContainsAny(prefix, "*?[")
	}
	_, err := path.Match(pattern, pattern)
	return err == nil
}

func matchInterceptorPathPattern(pattern, requestPath string) bool {
	if pattern == "/**" {
		return true
	}
	if strings.HasSuffix(pattern, "/**") {
		prefix := strings.TrimSuffix(pattern, "/**")
		return requestPath == prefix || strings.HasPrefix(requestPath, prefix+"/")
	}
	matched, err := path.Match(pattern, requestPath)
	return err == nil && matched
}
