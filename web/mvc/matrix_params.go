package mvc

import (
	"net/url"
	"strings"

	arkweb "goark.dev/arkarta/web"
)

// WithMatrixPathVariable 指定矩阵变量所属的路径变量段。
func WithMatrixPathVariable(pathVariable string) ParamOption {
	return func(options *paramOptions) {
		options.matrixPathVariable = strings.TrimSpace(pathVariable)
	}
}

// MatrixVariableString 绑定字符串矩阵变量。
func MatrixVariableString(ctx *arkweb.Context, name string, options ...ParamOption) (string, error) {
	paramOptions := newParamOptions(ctx, options)
	value, ok, err := matrixValue(ctx, name, paramOptions.matrixPathVariable)
	return resolveStringParameter("矩阵变量", name, value, ok, err, paramOptions)
}

// MatrixVariableInt 绑定 int 矩阵变量。
func MatrixVariableInt(ctx *arkweb.Context, name string, options ...ParamOption) (int, error) {
	paramOptions := newParamOptions(ctx, options)
	value, ok, err := matrixValue(ctx, name, paramOptions.matrixPathVariable)
	return resolveIntParameter("矩阵变量", name, value, ok, err, paramOptions)
}

// MatrixVariableInt64 绑定 int64 矩阵变量。
func MatrixVariableInt64(ctx *arkweb.Context, name string, options ...ParamOption) (int64, error) {
	paramOptions := newParamOptions(ctx, options)
	value, ok, err := matrixValue(ctx, name, paramOptions.matrixPathVariable)
	return resolveInt64Parameter("矩阵变量", name, value, ok, err, paramOptions)
}

// MatrixVariableBool 绑定 bool 矩阵变量。
func MatrixVariableBool(ctx *arkweb.Context, name string, options ...ParamOption) (bool, error) {
	paramOptions := newParamOptions(ctx, options)
	value, ok, err := matrixValue(ctx, name, paramOptions.matrixPathVariable)
	return resolveBoolParameter("矩阵变量", name, value, ok, err, paramOptions)
}

func matrixValue(ctx *arkweb.Context, name string, pathVariable string) (string, bool, error) {
	if ctx == nil || ctx.Request() == nil {
		return "", false, arkweb.ErrNilContext
	}
	values := matrixValues(ctx, pathVariable)
	value, ok := values[name]
	return value, ok, nil
}

func matrixValues(ctx *arkweb.Context, pathVariable string) map[string]string {
	if pathVariable != "" {
		segment, ok := ctx.Param(pathVariable)
		if !ok {
			return map[string]string{}
		}
		return matrixSegmentValues(segment)
	}
	out := make(map[string]string)
	for _, segment := range strings.Split(strings.Trim(ctx.Request().Path(), "/"), "/") {
		for name, value := range matrixSegmentValues(segment) {
			if _, exists := out[name]; !exists {
				out[name] = value
			}
		}
	}
	return out
}

func matrixSegmentValues(segment string) map[string]string {
	out := make(map[string]string)
	parts := strings.Split(segment, ";")
	if len(parts) < 2 {
		return out
	}
	for _, part := range parts[1:] {
		name, value, ok := strings.Cut(part, "=")
		if !ok {
			value = ""
		}
		name = pathUnescape(strings.TrimSpace(name))
		if name == "" {
			continue
		}
		if _, exists := out[name]; exists {
			continue
		}
		out[name] = pathUnescape(strings.TrimSpace(value))
	}
	return out
}

func stripMatrixSegment(value string) string {
	if base, _, ok := strings.Cut(value, ";"); ok {
		return base
	}
	return value
}

func pathUnescape(value string) string {
	out, err := url.PathUnescape(value)
	if err != nil {
		return value
	}
	return out
}
