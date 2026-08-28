package mvc

import (
	servletmultipart "goark.dev/arkarta/servlet/multipart"
	arkweb "goark.dev/arkarta/web"
)

// RequestPart 绑定 multipart/form-data 中的指定文件段。
func RequestPart(ctx *arkweb.Context, name string, options ...ParamOption) (servletmultipart.Part, error) {
	var zero servletmultipart.Part
	if ctx == nil || ctx.Request() == nil {
		return zero, arkweb.ErrNilContext
	}
	paramOptions := newParamOptions(options)
	parser := servletmultipart.NewParser()
	part, ok, err := servletmultipart.RequestPart(ctx.Request(), name, parser)
	if err != nil {
		return zero, err
	}
	if ok {
		return part, nil
	}
	if paramOptions.required {
		return zero, missingParameterError("请求Part", name)
	}
	return zero, nil
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
