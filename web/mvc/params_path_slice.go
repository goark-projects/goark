package mvc

import (
	"time"

	arkweb "goark.dev/arkarta/web"
)

// PathStrings 绑定字符串切片路径变量，支持逗号分隔值。
func PathStrings(ctx *arkweb.Context, name string, options ...ParamOption) ([]string, error) {
	values, ok, err := pathValues(ctx, name)
	return resolveStringSliceParameter("路径变量", name, values, ok, err, newParamOptions(ctx, options))
}

// PathInts 绑定 int 切片路径变量。
func PathInts(ctx *arkweb.Context, name string, options ...ParamOption) ([]int, error) {
	values, ok, err := pathValues(ctx, name)
	return resolveIntSliceParameter("路径变量", name, values, ok, err, newParamOptions(ctx, options))
}

// PathInt64s 绑定 int64 切片路径变量。
func PathInt64s(ctx *arkweb.Context, name string, options ...ParamOption) ([]int64, error) {
	values, ok, err := pathValues(ctx, name)
	return resolveInt64SliceParameter("路径变量", name, values, ok, err, newParamOptions(ctx, options))
}

// PathBools 绑定 bool 切片路径变量。
func PathBools(ctx *arkweb.Context, name string, options ...ParamOption) ([]bool, error) {
	values, ok, err := pathValues(ctx, name)
	return resolveBoolSliceParameter("路径变量", name, values, ok, err, newParamOptions(ctx, options))
}

// PathFloat64s 绑定 float64 切片路径变量。
func PathFloat64s(ctx *arkweb.Context, name string, options ...ParamOption) ([]float64, error) {
	values, ok, err := pathValues(ctx, name)
	return resolveFloat64SliceParameter("路径变量", name, values, ok, err, newParamOptions(ctx, options))
}

// PathTimes 绑定 time.Time 切片路径变量。
func PathTimes(ctx *arkweb.Context, name string, options ...ParamOption) ([]time.Time, error) {
	values, ok, err := pathValues(ctx, name)
	return resolveTimeSliceParameter("路径变量", name, values, ok, err, newParamOptions(ctx, options))
}

func pathValues(ctx *arkweb.Context, name string) ([]string, bool, error) {
	if ctx == nil {
		return nil, false, arkweb.ErrNilContext
	}
	value, ok := ctx.Param(name)
	if !ok {
		return nil, false, nil
	}
	return []string{stripMatrixSegment(value)}, true, nil
}
