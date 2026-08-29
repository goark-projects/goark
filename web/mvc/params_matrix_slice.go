package mvc

import (
	"time"

	arkweb "goark.dev/arkarta/web"
)

// MatrixVariableStrings 绑定字符串切片矩阵变量，支持逗号分隔值。
func MatrixVariableStrings(ctx *arkweb.Context, name string, options ...ParamOption) ([]string, error) {
	paramOptions := newParamOptions(ctx, options)
	values, ok, err := matrixValuesFor(ctx, name, paramOptions.matrixPathVariable)
	return resolveStringSliceParameter("矩阵变量", name, values, ok, err, paramOptions)
}

// MatrixVariableInts 绑定 int 切片矩阵变量。
func MatrixVariableInts(ctx *arkweb.Context, name string, options ...ParamOption) ([]int, error) {
	paramOptions := newParamOptions(ctx, options)
	values, ok, err := matrixValuesFor(ctx, name, paramOptions.matrixPathVariable)
	return resolveIntSliceParameter("矩阵变量", name, values, ok, err, paramOptions)
}

// MatrixVariableInt64s 绑定 int64 切片矩阵变量。
func MatrixVariableInt64s(ctx *arkweb.Context, name string, options ...ParamOption) ([]int64, error) {
	paramOptions := newParamOptions(ctx, options)
	values, ok, err := matrixValuesFor(ctx, name, paramOptions.matrixPathVariable)
	return resolveInt64SliceParameter("矩阵变量", name, values, ok, err, paramOptions)
}

// MatrixVariableBools 绑定 bool 切片矩阵变量。
func MatrixVariableBools(ctx *arkweb.Context, name string, options ...ParamOption) ([]bool, error) {
	paramOptions := newParamOptions(ctx, options)
	values, ok, err := matrixValuesFor(ctx, name, paramOptions.matrixPathVariable)
	return resolveBoolSliceParameter("矩阵变量", name, values, ok, err, paramOptions)
}

// MatrixVariableFloat64s 绑定 float64 切片矩阵变量。
func MatrixVariableFloat64s(ctx *arkweb.Context, name string, options ...ParamOption) ([]float64, error) {
	paramOptions := newParamOptions(ctx, options)
	values, ok, err := matrixValuesFor(ctx, name, paramOptions.matrixPathVariable)
	return resolveFloat64SliceParameter("矩阵变量", name, values, ok, err, paramOptions)
}

// MatrixVariableTimes 绑定 time.Time 切片矩阵变量。
func MatrixVariableTimes(ctx *arkweb.Context, name string, options ...ParamOption) ([]time.Time, error) {
	paramOptions := newParamOptions(ctx, options)
	values, ok, err := matrixValuesFor(ctx, name, paramOptions.matrixPathVariable)
	return resolveTimeSliceParameter("矩阵变量", name, values, ok, err, paramOptions)
}

func matrixValuesFor(ctx *arkweb.Context, name string, pathVariable string) ([]string, bool, error) {
	values, err := matrixValueListsForContext(ctx, pathVariable)
	if err != nil {
		return nil, false, err
	}
	list, ok := values[name]
	if !ok || len(list) == 0 {
		return nil, ok, err
	}
	return append([]string(nil), list...), true, nil
}
