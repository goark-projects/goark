package filter

import (
	"context"
	"net/http"
	"strings"

	"goark.dev/arkarta/servlet"
)

const onceAttributePrefix = "goark.web.filter.once."

type onceFilter struct {
	attributeName string
	delegate      servlet.Filter
}

// Once 保证同一个请求分发链中同名过滤器最多执行一次。
func Once(name string, delegate servlet.Filter) (servlet.Filter, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, ErrInvalidFilterName
	}
	if delegate == nil {
		return nil, ErrNilFilter
	}
	return onceFilter{
		attributeName: onceAttributePrefix + name,
		delegate:      delegate,
	}, nil
}

func (f onceFilter) Filter(ctx context.Context, req *servlet.Request, res servlet.Response, chain servlet.Chain) error {
	if req == nil {
		return servlet.NewHTTPError(http.StatusBadRequest, http.StatusText(http.StatusBadRequest), nil)
	}
	if chain == nil {
		return ErrNilChain
	}
	if value, ok := req.Attribute(f.attributeName); ok && value == true {
		return chain.Next(ctx, req, res)
	}
	req.SetAttribute(f.attributeName, true)
	defer req.SetAttribute(f.attributeName, nil)
	return f.delegate.Filter(ctx, req, res, chain)
}
