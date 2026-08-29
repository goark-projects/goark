package mvc

import (
	"net/url"
	"strings"

	arkweb "goark.dev/arkarta/web"
)

// MatrixVariableString 绑定字符串矩阵变量。
func MatrixVariableString(ctx *arkweb.Context, name string, options ...ParamOption) (string, error) {
	value, ok, err := matrixValue(ctx, name)
	return resolveStringParameter("矩阵变量", name, value, ok, err, newParamOptions(ctx, options))
}

// MatrixVariableInt 绑定 int 矩阵变量。
func MatrixVariableInt(ctx *arkweb.Context, name string, options ...ParamOption) (int, error) {
	value, ok, err := matrixValue(ctx, name)
	return resolveIntParameter("矩阵变量", name, value, ok, err, newParamOptions(ctx, options))
}

// MatrixVariableInt64 绑定 int64 矩阵变量。
func MatrixVariableInt64(ctx *arkweb.Context, name string, options ...ParamOption) (int64, error) {
	value, ok, err := matrixValue(ctx, name)
	return resolveInt64Parameter("矩阵变量", name, value, ok, err, newParamOptions(ctx, options))
}

// MatrixVariableBool 绑定 bool 矩阵变量。
func MatrixVariableBool(ctx *arkweb.Context, name string, options ...ParamOption) (bool, error) {
	value, ok, err := matrixValue(ctx, name)
	return resolveBoolParameter("矩阵变量", name, value, ok, err, newParamOptions(ctx, options))
}

func matrixValue(ctx *arkweb.Context, name string) (string, bool, error) {
	if ctx == nil || ctx.Request() == nil {
		return "", false, arkweb.ErrNilContext
	}
	values := matrixValues(ctx.Request().Path())
	value, ok := values[name]
	return value, ok, nil
}

func matrixValues(path string) map[string]string {
	out := make(map[string]string)
	for _, segment := range strings.Split(strings.Trim(path, "/"), "/") {
		parts := strings.Split(segment, ";")
		if len(parts) < 2 {
			continue
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
