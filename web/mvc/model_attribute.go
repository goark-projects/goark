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
	if err := bindModelAttribute(ctx, &out); err != nil {
		return out, err
	}
	return out, validateBound(ctx, &out, groups)
}

// ModelAttributeResult 绑定 query/form 聚合参数，并返回可由调用方处理的绑定和验证结果。
func ModelAttributeResult[T any](ctx *arkweb.Context) (T, BindingResult, error) {
	return ModelAttributeResultGroups[T](ctx)
}

// ModelAttributeResultGroups 绑定 query/form 聚合参数，并按显式分组返回绑定和验证结果。
func ModelAttributeResultGroups[T any](ctx *arkweb.Context, groups ...string) (T, BindingResult, error) {
	var out T
	if ctx == nil {
		return out, BindingResult{}, arkweb.ErrNilContext
	}
	if err := bindModelAttribute(ctx, &out); err != nil {
		return out, newBindingErrorResult(err), nil
	}
	result, err := validateBindingResult(ctx, &out, groups)
	return out, result, err
}
