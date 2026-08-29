package mvc

import (
	"time"

	arkweb "goark.dev/arkarta/web"
)

// MatrixVariableStrings 绑定字符串切片矩阵变量，支持逗号分隔值。
func MatrixVariableStrings(ctx *arkweb.Context, name string, options ...ParamOption) ([]string, error) {
	values, ok, err := matrixValuesFor(ctx, name)
	return resolveStringSliceParameter("矩阵变量", name, values, ok, err, newParamOptions(options))
}

// MatrixVariableInts 绑定 int 切片矩阵变量。
func MatrixVariableInts(ctx *arkweb.Context, name string, options ...ParamOption) ([]int, error) {
	values, ok, err := matrixValuesFor(ctx, name)
	return resolveIntSliceParameter("矩阵变量", name, values, ok, err, newParamOptions(options))
}

// MatrixVariableInt64s 绑定 int64 切片矩阵变量。
func MatrixVariableInt64s(ctx *arkweb.Context, name string, options ...ParamOption) ([]int64, error) {
	values, ok, err := matrixValuesFor(ctx, name)
	return resolveInt64SliceParameter("矩阵变量", name, values, ok, err, newParamOptions(options))
}

// MatrixVariableBools 绑定 bool 切片矩阵变量。
func MatrixVariableBools(ctx *arkweb.Context, name string, options ...ParamOption) ([]bool, error) {
	values, ok, err := matrixValuesFor(ctx, name)
	return resolveBoolSliceParameter("矩阵变量", name, values, ok, err, newParamOptions(options))
}

// MatrixVariableFloat64s 绑定 float64 切片矩阵变量。
func MatrixVariableFloat64s(ctx *arkweb.Context, name string, options ...ParamOption) ([]float64, error) {
	values, ok, err := matrixValuesFor(ctx, name)
	return resolveFloat64SliceParameter("矩阵变量", name, values, ok, err, newParamOptions(options))
}

// MatrixVariableTimes 绑定 time.Time 切片矩阵变量。
func MatrixVariableTimes(ctx *arkweb.Context, name string, options ...ParamOption) ([]time.Time, error) {
	values, ok, err := matrixValuesFor(ctx, name)
	return resolveTimeSliceParameter("矩阵变量", name, values, ok, err, newParamOptions(options))
}

func matrixValuesFor(ctx *arkweb.Context, name string) ([]string, bool, error) {
	value, ok, err := matrixValue(ctx, name)
	if err != nil || !ok {
		return nil, ok, err
	}
	return []string{value}, true, nil
}
