package mvc

import arkweb "goark.dev/arkarta/web"

// RequestBody 将 JSON 请求体绑定到目标类型，不执行结构体验证。
func RequestBody[T any](ctx *arkweb.Context) (T, error) {
	var out T
	if ctx == nil {
		return out, arkweb.ErrNilContext
	}
	if err := ctx.BindJSON(&out); err != nil {
		return out, err
	}
	return out, nil
}

// ValidatedRequestBody 将 JSON 请求体绑定到目标类型，并按可选分组执行结构体验证。
func ValidatedRequestBody[T any](ctx *arkweb.Context, groups ...string) (T, error) {
	var out T
	if err := bindAndValidateJSON(ctx, &out, groups); err != nil {
		return out, err
	}
	return out, nil
}
