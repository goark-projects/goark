package mvc

import (
	arkweb "goark.dev/arkarta/web"
	mvcflash "goark.dev/goark/web/mvc/flash"
)

const (
	// AttributeModel 保存当前 MVC 请求的视图模型。
	AttributeModel = "goark.web.mvc.model"
)

// CurrentModel 返回当前 MVC 请求模型；没有模型时创建空模型。
func CurrentModel(ctx *arkweb.Context) Model {
	model, ok := currentModel(ctx)
	if ok {
		return model
	}
	input := mvcflash.Input(ctx)
	model = NewModel().AddAllAttributes((&input).Values())
	setCurrentModel(ctx, model)
	return model
}

func currentModel(ctx *arkweb.Context) (Model, bool) {
	if ctx == nil || ctx.Request() == nil {
		return NewModel(), false
	}
	value, ok := ctx.Request().Attribute(AttributeModel)
	if !ok {
		return NewModel(), false
	}
	switch typed := value.(type) {
	case Model:
		return typed, true
	case *Model:
		if typed != nil {
			return *typed, true
		}
	}
	return NewModel(), false
}

func setCurrentModel(ctx *arkweb.Context, model Model) {
	if ctx == nil || ctx.Request() == nil {
		return
	}
	ctx.Request().SetAttribute(AttributeModel, model)
}

func mergeCurrentModel(ctx *arkweb.Context, model Model) Model {
	base, ok := currentModel(ctx)
	if !ok {
		setCurrentModel(ctx, model)
		return model
	}
	merged := base.AddAllAttributes(model.Values())
	setCurrentModel(ctx, merged)
	return merged
}

func modelForView(ctx *arkweb.Context) any {
	model, ok := currentModel(ctx)
	if !ok {
		return nil
	}
	return model.Values()
}
