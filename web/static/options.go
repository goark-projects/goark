// Package static 提供 Goark Web 静态资源注册入口。
package static

import (
	"errors"
	"strconv"
	"strings"
	"time"

	servletresource "goark.dev/arkarta/servlet/resource"
)

// ErrInvalidCacheControl 表示静态资源缓存控制头非法。
var ErrInvalidCacheControl = errors.New("goark/web/static: invalid cache-control")

type config struct {
	servletName     string
	contentType     servletresource.ContentTypeFunc
	welcomeFiles    []string
	welcomeFilesSet bool
	cacheControl    string
	contentVersion  bool
}

// Option 定制静态资源配置器。
type Option func(*config) error

// WithServletName 设置底层 default servlet 名称。
func WithServletName(name string) Option {
	return func(cfg *config) error {
		cfg.servletName = strings.TrimSpace(name)
		return nil
	}
}

// WithContentTypeFunc 设置文件扩展名到媒体类型的解析函数。
func WithContentTypeFunc(fn servletresource.ContentTypeFunc) Option {
	return func(cfg *config) error {
		if fn != nil {
			cfg.contentType = fn
		}
		return nil
	}
}

// WithWelcomeFiles 设置目录请求的 welcome 文件；空列表表示禁用 welcome 解析。
func WithWelcomeFiles(files ...string) Option {
	return func(cfg *config) error {
		cfg.welcomeFiles = append([]string(nil), files...)
		cfg.welcomeFilesSet = true
		return nil
	}
}

// WithCacheControl 设置静态资源命中响应的 Cache-Control 头。
func WithCacheControl(value string) Option {
	return func(cfg *config) error {
		value = strings.TrimSpace(value)
		if strings.ContainsAny(value, "\r\n") {
			return ErrInvalidCacheControl
		}
		cfg.cacheControl = value
		return nil
	}
}

// WithCacheMaxAge 设置 public max-age 缓存控制。
func WithCacheMaxAge(maxAge time.Duration) Option {
	return func(cfg *config) error {
		if maxAge < 0 {
			return ErrInvalidCacheControl
		}
		seconds := int64(maxAge / time.Second)
		cfg.cacheControl = "public, max-age=" + strconv.FormatInt(seconds, 10)
		return nil
	}
}

// WithContentVersioning 启用静态资源内容哈希版本路径解析。
func WithContentVersioning() Option {
	return func(cfg *config) error {
		cfg.contentVersion = true
		return nil
	}
}
