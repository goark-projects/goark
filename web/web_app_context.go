package web

import (
	"context"

	"goark.dev/arkarta/servlet"
	arkweb "goark.dev/arkarta/web"
)

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
