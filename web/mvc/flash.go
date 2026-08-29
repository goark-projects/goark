package mvc

import (
	arkweb "goark.dev/arkarta/web"
	mvcflash "goark.dev/goark/web/mvc/flash"
)

// FlashMap 是 Spring FlashMap 的 Go 化一次性属性集合。
type FlashMap = mvcflash.Map

// InputFlashMap 返回当前请求输入 FlashMap 的副本。
func InputFlashMap(ctx *arkweb.Context) mvcflash.Map {
	return mvcflash.Input(ctx)
}

// OutputFlashMap 返回当前请求输出 FlashMap；不存在时按需创建。
func OutputFlashMap(ctx *arkweb.Context) *mvcflash.Map {
	return mvcflash.Output(ctx)
}

// FlashAttribute 读取一次性 Flash 属性，并转换为目标类型。
func FlashAttribute[T any](ctx *arkweb.Context, name string, options ...ParamOption) (T, error) {
	value, ok, err := rawFlashAttributeValue(ctx, name)
	paramOptions := newParamOptions(ctx, options)
	return resolveAttributeValue[T]("Flash属性", name, value, ok, err, paramOptions)
}

func saveRedirectFlash(ctx *arkweb.Context, location string, model Model) error {
	if model.Len() == 0 {
		return nil
	}
	output := mvcflash.Output(ctx)
	if output == nil {
		return arkweb.ErrNilContext
	}
	output.AddAllAttributes(model.Values())
	output.SetTargetLocation(location)
	return nil
}

func rawFlashAttributeValue(ctx *arkweb.Context, name string) (any, bool, error) {
	if ctx == nil {
		return nil, false, arkweb.ErrNilContext
	}
	input := mvcflash.Input(ctx)
	value, ok := (&input).Attribute(name)
	return value, ok, nil
}

func flashAttributeValue(ctx *arkweb.Context, name string) (string, bool, error) {
	value, ok, err := rawFlashAttributeValue(ctx, name)
	if err != nil || !ok {
		return "", ok, err
	}
	return attributeString(value), true, nil
}
