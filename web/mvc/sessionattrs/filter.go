package sessionattrs

import (
	"context"

	"goark.dev/arkarta/servlet"
	"goark.dev/arkarta/servlet/session"
)

// Filter 在 Servlet Session 边界保存并清理 MVC SessionAttributes。
type Filter struct {
	accessor *session.Accessor
}

// NewFilter 创建 MVC SessionAttributes 过滤器。
func NewFilter(accessor *session.Accessor) (*Filter, error) {
	if accessor == nil {
		return nil, ErrNilAccessor
	}
	return &Filter{accessor: accessor}, nil
}

// NewSessionFilter 使用 Session Manager 创建 MVC SessionAttributes 过滤器。
func NewSessionFilter(manager session.Manager) (*Filter, error) {
	accessor, err := session.NewAccessor(manager)
	if err != nil {
		return nil, err
	}
	return NewFilter(accessor)
}

// Filter 执行请求级 Session 绑定和响应级模型属性持久化。
func (f *Filter) Filter(ctx context.Context, req *servlet.Request, res servlet.Response, chain servlet.Chain) error {
	if chain == nil {
		return servlet.ErrNilHandler
	}
	if f == nil || f.accessor == nil {
		return ErrNilAccessor
	}
	if _, _, err := f.accessor.Get(ctx, req, nil, false); err != nil {
		return err
	}
	if err := chain.Next(ctx, req, res); err != nil {
		return err
	}
	if names := completed(req); len(names) > 0 {
		return f.removeCompleted(ctx, req, names)
	}
	return f.savePending(ctx, req, res)
}

func (f *Filter) savePending(ctx context.Context, req *servlet.Request, res servlet.Response) error {
	values := pending(req)
	if len(values) == 0 {
		return nil
	}
	current, _, err := f.accessor.Get(ctx, req, res, true)
	if err != nil {
		return err
	}
	for name, value := range values {
		if err := current.SetAttribute(name, value); err != nil {
			return err
		}
	}
	return nil
}

func (f *Filter) removeCompleted(ctx context.Context, req *servlet.Request, names []string) error {
	current, ok, err := f.accessor.Get(ctx, req, nil, false)
	if err != nil || !ok {
		return err
	}
	for _, name := range names {
		if err := current.RemoveAttribute(name); err != nil {
			return err
		}
	}
	return nil
}

var _ servlet.Filter = (*Filter)(nil)
