package static

import (
	"context"
	"net/http"

	"goark.dev/arkarta/servlet"
)

type cacheControlServlet struct {
	target       servlet.Servlet
	cacheControl string
}

func newCacheControlServlet(target servlet.Servlet, cacheControl string) servlet.Servlet {
	if cacheControl == "" {
		return target
	}
	return cacheControlServlet{
		target:       target,
		cacheControl: cacheControl,
	}
}

func (s cacheControlServlet) Init(ctx context.Context, config servlet.ServletConfig) error {
	return s.target.Init(ctx, config)
}

func (s cacheControlServlet) Serve(ctx context.Context, req *servlet.Request, res servlet.Response) error {
	if res == nil {
		return s.target.Serve(ctx, req, res)
	}
	return s.target.Serve(ctx, req, cacheControlResponse{
		Response:     res,
		cacheControl: s.cacheControl,
	})
}

func (s cacheControlServlet) Destroy(ctx context.Context) error {
	return s.target.Destroy(ctx)
}

type cacheControlResponse struct {
	servlet.Response
	cacheControl string
}

func (r cacheControlResponse) SetStatus(code int) {
	if isStaticCacheStatus(code) && r.Header().Get("Cache-Control") == "" {
		r.Header().Set("Cache-Control", r.cacheControl)
	}
	r.Response.SetStatus(code)
}

func isStaticCacheStatus(code int) bool {
	switch code {
	case http.StatusOK, http.StatusPartialContent, http.StatusNotModified:
		return true
	default:
		return false
	}
}
