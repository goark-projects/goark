package uri

import (
	"net/url"

	arkweb "goark.dev/arkarta/web"
)

// FromCurrentRequest 基于当前请求的 scheme、host、path 和 query 创建构建器。
func FromCurrentRequest(ctx *arkweb.Context) Builder {
	if ctx == nil || ctx.Request() == nil {
		return New()
	}
	req := ctx.Request()
	query, _ := url.ParseQuery(req.QueryString())
	return Builder{
		scheme: req.Scheme(),
		host:   req.Host(),
		path:   joinPath(req.ContextPath(), req.Path()),
		query:  query,
	}
}

// FromCurrentRequestURI 基于当前请求的 scheme、host 和 path 创建构建器，不携带 query。
func FromCurrentRequestURI(ctx *arkweb.Context) Builder {
	return FromCurrentRequest(ctx).ClearQuery()
}
