// Package view 提供 Goark MVC 视图解析与模板渲染能力。
package view

import (
	"net/http"

	"goark.dev/arkarta/servlet"
	arkweb "goark.dev/arkarta/web"
)

const (
	// AttributeResolver 保存当前请求可用的 MVC 视图解析器。
	AttributeResolver = "goark.web.mvc.view.resolver"
	defaultStatus     = http.StatusOK
)

// View 表示一个已经解析完成的可渲染视图。
type View interface {
	ContentType() string
	Render(ctx *arkweb.Context, model any) error
}

// Resolver 将逻辑视图名解析为具体视图。
type Resolver interface {
	ResolveView(ctx *arkweb.Context, name string) (View, bool, error)
}

// Result 表示 MVC 逻辑视图名响应。
type Result struct {
	name        string
	model       any
	status      int
	contentType string
	resolver    Resolver
}

// ResultOption 定制视图响应。
type ResultOption func(*Result)

// Render 使用请求上下文中的视图解析器渲染逻辑视图名。
func Render(name string, model any, options ...ResultOption) arkweb.Result {
	return newResult(nil, name, model, options...)
}

// Using 使用指定视图解析器渲染逻辑视图名。
func Using(resolver Resolver, name string, model any, options ...ResultOption) arkweb.Result {
	return newResult(resolver, name, model, options...)
}

// WithResolver 设置该视图结果使用的解析器。
func WithResolver(resolver Resolver) ResultOption {
	return func(result *Result) {
		result.resolver = resolver
	}
}

// WithStatus 设置视图响应状态码。
func WithStatus(status int) ResultOption {
	return func(result *Result) {
		if status >= 100 && status <= 999 {
			result.status = status
		}
	}
}

// WithContentType 覆盖视图响应媒体类型。
func WithContentType(contentType string) ResultOption {
	return func(result *Result) {
		result.contentType = cleanHeaderValue(contentType)
	}
}

func newResult(resolver Resolver, name string, model any, options ...ResultOption) *Result {
	result := &Result{
		name:     name,
		model:    model,
		status:   defaultStatus,
		resolver: resolver,
	}
	for _, option := range options {
		if option != nil {
			option(result)
		}
	}
	return result
}

// Write 解析并渲染视图。
func (r *Result) Write(ctx *arkweb.Context) error {
	if ctx == nil || ctx.Response() == nil {
		return arkweb.ErrNilContext
	}
	resolver, err := r.resolveResolver(ctx)
	if err != nil {
		return servlet.NewHTTPError(http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError), err)
	}
	resolved, ok, err := resolver.ResolveView(ctx, r.name)
	if err != nil {
		return err
	}
	if !ok {
		return servlet.NewHTTPError(http.StatusNotFound, http.StatusText(http.StatusNotFound), ErrViewNotFound)
	}
	if resolved == nil {
		return servlet.NewHTTPError(http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError), ErrNilView)
	}
	contentType := r.contentType
	if contentType == "" {
		contentType = cleanHeaderValue(resolved.ContentType())
	}
	if contentType != "" {
		ctx.Response().Header().Set("Content-Type", contentType)
	}
	ctx.Response().SetStatus(r.status)
	return resolved.Render(ctx, r.model)
}

func (r *Result) resolveResolver(ctx *arkweb.Context) (Resolver, error) {
	if r != nil && r.resolver != nil {
		return r.resolver, nil
	}
	resolver, ok := ResolverFromContext(ctx)
	if !ok {
		return nil, ErrNilResolver
	}
	return resolver, nil
}
