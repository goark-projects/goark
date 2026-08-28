package mvc

import arkweb "goark.dev/arkarta/web"

// ModelAttribute 绑定 query/form 聚合参数并执行结构体验证。
func ModelAttribute[T any](ctx *arkweb.Context) (T, error) {
	var out T
	if ctx == nil {
		return out, arkweb.ErrNilContext
	}
	if err := ctx.BindForm(&out); err != nil {
		return out, err
	}
	result, err := ctx.Validate(&out)
	if err != nil {
		return out, err
	}
	return out, result.Error()
}
