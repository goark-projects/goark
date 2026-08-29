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

// MatrixVariableMap 绑定全部矩阵变量，每个名称取第一个值。
func MatrixVariableMap(ctx *arkweb.Context, options ...ParamOption) (map[string]string, error) {
	paramOptions := newParamOptions(ctx, options)
	values, err := matrixValueListsForContext(ctx, paramOptions.matrixPathVariable)
	if err != nil {
		return nil, err
	}
	return firstStringValueMap(values), nil
}

// MatrixVariableValuesMap 绑定全部矩阵变量，保留每个名称的全部值。
func MatrixVariableValuesMap(ctx *arkweb.Context, options ...ParamOption) (map[string][]string, error) {
	paramOptions := newParamOptions(ctx, options)
	values, err := matrixValueListsForContext(ctx, paramOptions.matrixPathVariable)
	if err != nil {
		return nil, err
	}
	return cloneStringValuesMap(values), nil
}

func matrixValue(ctx *arkweb.Context, name string, pathVariable string) (string, bool, error) {
	if ctx == nil || ctx.Request() == nil {
		return "", false, arkweb.ErrNilContext
	}
	values := firstStringValueMap(matrixValueLists(ctx, pathVariable))
	value, ok := values[name]
	return value, ok, nil
}

func matrixValueListsForContext(ctx *arkweb.Context, pathVariable string) (map[string][]string, error) {
	if ctx == nil || ctx.Request() == nil {
		return nil, arkweb.ErrNilContext
	}
	return matrixValueLists(ctx, pathVariable), nil
}

func matrixValueLists(ctx *arkweb.Context, pathVariable string) map[string][]string {
	if pathVariable != "" {
		segment, ok := ctx.Param(pathVariable)
		if !ok {
			return map[string][]string{}
		}
		return matrixSegmentValueLists(segment)
	}
	out := make(map[string][]string)
	for _, segment := range strings.Split(strings.Trim(ctx.Request().Path(), "/"), "/") {
		for name, values := range matrixSegmentValueLists(segment) {
			out[name] = append(out[name], values...)
		}
	}
	return out
}

func matrixSegmentValues(segment string) map[string]string {
	return firstStringValueMap(matrixSegmentValueLists(segment))
}

func matrixSegmentValueLists(segment string) map[string][]string {
	out := make(map[string][]string)
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
		out[name] = append(out[name], pathUnescape(strings.TrimSpace(value)))
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
