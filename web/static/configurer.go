package static

import (
	"context"
	"io/fs"

	"goark.dev/arkarta/servlet"
	servletresource "goark.dev/arkarta/servlet/resource"
	"goark.dev/goark/container"
	goweb "goark.dev/goark/web"
)

// Configurer 将 fs.FS 静态资源发布为 Arkarta default servlet。
type Configurer struct {
	pattern     string
	servletName string
	servlet     servlet.Servlet
}

// New 创建静态资源配置器。
func New(pattern string, root fs.FS, options ...Option) (Configurer, error) {
	cfg := config{}
	for _, option := range options {
		if option == nil {
			continue
		}
		if err := option(&cfg); err != nil {
			return Configurer{}, err
		}
	}
	providerOptions := make([]servletresource.FSProviderOption, 0, 1)
	if cfg.contentType != nil {
		providerOptions = append(providerOptions, servletresource.WithContentTypeFunc(cfg.contentType))
	}
	provider, err := servletresource.NewFSProvider(root, providerOptions...)
	if err != nil {
		return Configurer{}, err
	}
	servletOptions := make([]servletresource.DefaultServletOption, 0, 1)
	if cfg.welcomeFilesSet {
		servletOptions = append(servletOptions, servletresource.WithWelcomeFiles(cfg.welcomeFiles...))
	}
	handler, err := servletresource.NewDefaultServlet(provider, servletOptions...)
	if err != nil {
		return Configurer{}, err
	}
	return Configurer{
		pattern:     pattern,
		servletName: cfg.servletName,
		servlet:     handler,
	}, nil
}

// Register 注册静态资源配置器 Bean。
func Register(registry *container.Registry, name string, pattern string, root fs.FS, options ...Option) error {
	configurer, err := New(pattern, root, options...)
	if err != nil {
		return err
	}
	return goweb.RegisterConfigurer(registry, name, configurer)
}

// ConfigureWeb 将静态资源映射追加到 Web 部署描述。
func (c Configurer) ConfigureWeb(ctx context.Context, registry *goweb.Registry) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if registry == nil {
		return goweb.ErrNilRegistry
	}
	return registry.AddServlet(c.pattern, c.servletName, c.servlet)
}
