package web

import (
	arkweb "goark.dev/arkarta/web"
	"goark.dev/goark/web/uri"
)

// CreatedFromCurrentRequest 基于当前请求 URI 追加路径模板并创建 201 Created 响应。
func CreatedFromCurrentRequest[T any](ctx *arkweb.Context, path string, variables map[string]string, body T) (ResponseEntity[T], error) {
	location, err := currentRequestLocation(ctx, path, variables)
	if err != nil {
		return ResponseEntity[T]{}, err
	}
	return Created(location, body), nil
}

// CreatedNoBodyFromCurrentRequest 基于当前请求 URI 追加路径模板并创建无响应体 201 Created 响应。
func CreatedNoBodyFromCurrentRequest(ctx *arkweb.Context, path string, variables map[string]string) (ResponseEntity[struct{}], error) {
	location, err := currentRequestLocation(ctx, path, variables)
	if err != nil {
		return ResponseEntity[struct{}]{}, err
	}
	return CreatedNoBody(location), nil
}

func currentRequestLocation(ctx *arkweb.Context, path string, variables map[string]string) (string, error) {
	if ctx == nil || ctx.Request() == nil {
		return "", arkweb.ErrNilContext
	}
	if variables == nil {
		variables = map[string]string{}
	}
	return uri.FromCurrentRequestURI(ctx).Path(path).BuildAndExpand(variables)
}
