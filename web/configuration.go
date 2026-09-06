// Package web 提供基于 Arkarta Servlet 的 Goark Web 运行时契约。
package web

import (
	"context"
	"strings"

	"goark.dev/arkarta/servlet"
	servletcontainer "goark.dev/arkarta/servlet/container"
	arkweb "goark.dev/arkarta/web"
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

const (
	defaultAppName        = "goark"
	defaultContextPath    = "/"
	defaultMappingPattern = "/"
)

// DeploymentSpec 描述 Arkarta Servlet 部署构建参数。
type DeploymentSpec struct {
	AppName           string
	ContextPath       string
	MappingPattern    string
	RouterOptions     []arkweb.Option
	WebAppOptions     []servlet.WebAppOption
	DeploymentOptions []servletcontainer.DeploymentOption
}

// BuildDeployment 将 Web 注册表构造成 Arkarta Servlet 部署。
func BuildDeployment(registry *Registry, spec DeploymentSpec) (*servletcontainer.Deployment, error) {
	if registry == nil {
		return nil, ErrNilRegistry
	}
	appName := strings.TrimSpace(spec.AppName)
	if appName == "" {
		appName = defaultAppName
	}
	contextPath := strings.TrimSpace(spec.ContextPath)
	if contextPath == "" {
		contextPath = defaultContextPath
	}
	mappingPattern := strings.TrimSpace(spec.MappingPattern)
	if mappingPattern == "" {
		mappingPattern = defaultMappingPattern
	}

	webAppOptions := make([]servlet.WebAppOption, 0, len(spec.WebAppOptions)+1)
	webAppOptions = append(webAppOptions, servlet.WithContextPath(contextPath))
	webAppOptions = append(webAppOptions, spec.WebAppOptions...)
	app, err := servlet.NewWebApp(appName, webAppOptions...)
	if err != nil {
		return nil, err
	}
	router, err := registry.Router(spec.RouterOptions...)
	if err != nil {
		return nil, err
	}

	globalFilters := registry.Filters()
	webFilters := make([]servlet.Filter, 0, len(globalFilters)+1)
	webFilters = append(webFilters, webAppRequestFilter(app))
	webFilters = append(webFilters, globalFilters...)
	deploymentOptions := []servletcontainer.DeploymentOption{
		servletcontainer.WithMapping(mappingPattern, router, webFilters...),
	}
	for _, mapping := range registry.servletMappings() {
		deploymentOptions = append(deploymentOptions, servletcontainer.WithServlet(
			mapping.pattern,
			mapping.name,
			mapping.handler,
			servletMappingFilters(globalFilters, mapping.filters)...,
		))
	}
	for _, profile := range registry.Profiles() {
		deploymentOptions = append(deploymentOptions, servletcontainer.WithProfile(profile))
	}
	deploymentOptions = append(deploymentOptions, registry.DeploymentOptions()...)
	deploymentOptions = append(deploymentOptions, spec.DeploymentOptions...)
	return servletcontainer.NewDeployment(app, deploymentOptions...)
}

const (
	// AttributeWebApp 保存当前请求所属的 Arkarta WebApp。
	AttributeWebApp = "goark.web.web_app"
)

// CurrentWebApp 返回当前 Goark Web 请求绑定的 Arkarta WebApp。
func CurrentWebApp(ctx *arkweb.Context) (*servlet.WebApp, bool) {
	if ctx == nil || ctx.Request() == nil {
		return nil, false
	}
	value, ok := ctx.Request().Attribute(AttributeWebApp)
	if !ok {
		return nil, false
	}
	app, ok := value.(*servlet.WebApp)
	if !ok || app == nil {
		return nil, false
	}
	return app, true
}

func webAppRequestFilter(app *servlet.WebApp) servlet.Filter {
	return servlet.FilterFunc(func(ctx context.Context, req *servlet.Request, res servlet.Response, chain servlet.Chain) error {
		if chain == nil {
			return servlet.ErrNilHandler
		}
		if app == nil || req == nil {
			return chain.Next(ctx, req, res)
		}
		previous, existed := req.Attribute(AttributeWebApp)
		req.SetAttribute(AttributeWebApp, app)
		defer restoreWebAppAttribute(req, previous, existed)
		return chain.Next(ctx, req, res)
	})
}

func restoreWebAppAttribute(req *servlet.Request, previous any, existed bool) {
	if req == nil {
		return
	}
	if existed {
		req.SetAttribute(AttributeWebApp, previous)
		return
	}
	req.SetAttribute(AttributeWebApp, nil)
}
