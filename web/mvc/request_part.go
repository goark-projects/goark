package mvc

import (
	servletmultipart "goark.dev/arkarta/servlet/multipart"
	arkweb "goark.dev/arkarta/web"
)

// RequestPart 绑定 multipart/form-data 中的指定文件段。
func RequestPart(ctx *arkweb.Context, name string, options ...ParamOption) (servletmultipart.Part, error) {
	part, _, err := requestPart(ctx, name, options...)
	return part, err
}

func requestPart(ctx *arkweb.Context, name string, options ...ParamOption) (servletmultipart.Part, bool, error) {
	var zero servletmultipart.Part
	if ctx == nil || ctx.Request() == nil {
		return zero, false, arkweb.ErrNilContext
	}
	paramOptions := newParamOptions(options)
	parser := servletmultipart.NewParser()
	part, ok, err := servletmultipart.RequestPart(ctx.Request(), name, parser)
	if err != nil {
		return zero, false, err
	}
	if ok {
		return part, true, nil
	}
	if paramOptions.required {
		return zero, false, missingParameterError("请求Part", name)
	}
	return zero, false, nil
}

// RequestParts 返回 multipart/form-data 中的全部文件段。
func RequestParts(ctx *arkweb.Context) ([]servletmultipart.Part, error) {
	if ctx == nil || ctx.Request() == nil {
		return nil, arkweb.ErrNilContext
	}
	parts, err := servletmultipart.Parts(ctx.Request(), servletmultipart.NewParser())
	if err != nil {
		return nil, err
	}
	return parts, nil
}
