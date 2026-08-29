package web

import (
	"strings"

	arkweb "goark.dev/arkarta/web"

	"goark.dev/arkarta/servlet"
	servletcontainer "goark.dev/arkarta/servlet/container"
)

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
