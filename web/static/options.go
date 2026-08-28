// Package static 提供 Goark Web 静态资源注册入口。
package static

import (
	"strings"

	servletresource "goark.dev/arkarta/servlet/resource"
)

type config struct {
	servletName     string
	contentType     servletresource.ContentTypeFunc
	welcomeFiles    []string
	welcomeFilesSet bool
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
