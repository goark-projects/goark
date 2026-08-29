package flash

import (
	"goark.dev/arkarta/servlet"
	arkweb "goark.dev/arkarta/web"
)

const (
	// AttributeInputMap 保存当前请求可读取的一次性 FlashMap。
	AttributeInputMap = "goark.web.mvc.flash.input"
	// AttributeOutputMap 保存当前请求准备写入下一次请求的 FlashMap。
	AttributeOutputMap = "goark.web.mvc.flash.output"
)

// Input 返回当前请求输入 FlashMap 的副本。
func Input(ctx *arkweb.Context) Map {
	if ctx == nil || ctx.Request() == nil {
		return Map{}
	}
	return inputMap(ctx.Request())
}

// Output 返回当前请求输出 FlashMap；不存在时按需创建。
func Output(ctx *arkweb.Context) *Map {
	if ctx == nil || ctx.Request() == nil {
		return nil
	}
	return outputMap(ctx.Request())
}

func inputMap(req *servlet.Request) Map {
	if req == nil {
		return Map{}
	}
	value, ok := req.Attribute(AttributeInputMap)
	if !ok {
		return Map{}
	}
	switch typed := value.(type) {
	case Map:
		return (&typed).clone()
	case *Map:
		return typed.clone()
	default:
		return Map{}
	}
}

func setInputMap(req *servlet.Request, value Map) {
	if req == nil {
		return
	}
	if (&value).Len() == 0 {
		req.SetAttribute(AttributeInputMap, nil)
		return
	}
	copied := value.clone()
	req.SetAttribute(AttributeInputMap, copied)
}

func outputMap(req *servlet.Request) *Map {
	value, ok := req.Attribute(AttributeOutputMap)
	if ok {
		if typed, ok := value.(*Map); ok && typed != nil {
			return typed
		}
	}
	out := NewMap()
	req.SetAttribute(AttributeOutputMap, out)
	return out
}

func existingOutputMap(req *servlet.Request) (*Map, bool) {
	if req == nil {
		return nil, false
	}
	value, ok := req.Attribute(AttributeOutputMap)
	if !ok {
		return nil, false
	}
	out, ok := value.(*Map)
	return out, ok && out != nil
}
