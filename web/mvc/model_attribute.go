package mvc

import arkweb "goark.dev/arkarta/web"

// ModelAttribute 绑定 query/form 聚合参数并执行结构体验证。
func ModelAttribute[T any](ctx *arkweb.Context) (T, error) {
	return ModelAttributeGroups[T](ctx)
}

// ModelAttributeGroups 绑定 query/form 聚合参数并按显式分组执行结构体验证。
func ModelAttributeGroups[T any](ctx *arkweb.Context, groups ...string) (T, error) {
	var out T
	if ctx == nil {
		return out, arkweb.ErrNilContext
	}
	if err := ctx.BindForm(&out); err != nil {
		return out, err
	}
	return out, validateBound(ctx, &out, groups)
}
