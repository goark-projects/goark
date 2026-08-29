package mvc

import (
	"time"

	arkweb "goark.dev/arkarta/web"
)

// PathVariableString 绑定字符串路径变量，对齐 Spring @PathVariable 命名。
func PathVariableString(ctx *arkweb.Context, name string, options ...ParamOption) (string, error) {
	return PathString(ctx, name, options...)
}

// PathVariableInt 绑定 int 路径变量。
func PathVariableInt(ctx *arkweb.Context, name string, options ...ParamOption) (int, error) {
	return PathInt(ctx, name, options...)
}

// PathVariableInt64 绑定 int64 路径变量。
func PathVariableInt64(ctx *arkweb.Context, name string, options ...ParamOption) (int64, error) {
	return PathInt64(ctx, name, options...)
}

// PathVariableBool 绑定 bool 路径变量。
func PathVariableBool(ctx *arkweb.Context, name string, options ...ParamOption) (bool, error) {
	return PathBool(ctx, name, options...)
}

// PathVariableFloat64 绑定 float64 路径变量。
func PathVariableFloat64(ctx *arkweb.Context, name string, options ...ParamOption) (float64, error) {
	return PathFloat64(ctx, name, options...)
}

// PathVariableTime 绑定 time.Time 路径变量。
func PathVariableTime(ctx *arkweb.Context, name string, options ...ParamOption) (time.Time, error) {
	return PathTime(ctx, name, options...)
}

// PathVariableAs 将路径变量转换为目标类型。
func PathVariableAs[T any](ctx *arkweb.Context, name string, options ...ParamOption) (T, error) {
	return PathValueAs[T](ctx, name, options...)
}

// PathVariableStrings 绑定字符串切片路径变量。
func PathVariableStrings(ctx *arkweb.Context, name string, options ...ParamOption) ([]string, error) {
	return PathStrings(ctx, name, options...)
}

// PathVariableInts 绑定 int 切片路径变量。
func PathVariableInts(ctx *arkweb.Context, name string, options ...ParamOption) ([]int, error) {
	return PathInts(ctx, name, options...)
}

// PathVariableInt64s 绑定 int64 切片路径变量。
func PathVariableInt64s(ctx *arkweb.Context, name string, options ...ParamOption) ([]int64, error) {
	return PathInt64s(ctx, name, options...)
}

// PathVariableBools 绑定 bool 切片路径变量。
func PathVariableBools(ctx *arkweb.Context, name string, options ...ParamOption) ([]bool, error) {
	return PathBools(ctx, name, options...)
}

// PathVariableFloat64s 绑定 float64 切片路径变量。
func PathVariableFloat64s(ctx *arkweb.Context, name string, options ...ParamOption) ([]float64, error) {
	return PathFloat64s(ctx, name, options...)
}

// PathVariableTimes 绑定 time.Time 切片路径变量。
func PathVariableTimes(ctx *arkweb.Context, name string, options ...ParamOption) ([]time.Time, error) {
	return PathTimes(ctx, name, options...)
}
