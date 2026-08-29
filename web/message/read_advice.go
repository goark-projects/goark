package message

import arkweb "goark.dev/arkarta/web"

// ReadAdviceContext 描述一次请求体读取扩展点的上下文。
type ReadAdviceContext struct {
	MediaType string
	Target    any
	Converter ReadConverter
}

// ReadAdvice 在请求体转换前后增强读取流程。
type ReadAdvice interface {
	BeforeRead(ctx *arkweb.Context, input ReadAdviceContext) error
	AfterRead(ctx *arkweb.Context, input ReadAdviceContext) error
}

// ReadAdviceFunc 将函数组适配为 ReadAdvice。
type ReadAdviceFunc struct {
	Before func(ctx *arkweb.Context, input ReadAdviceContext) error
	After  func(ctx *arkweb.Context, input ReadAdviceContext) error
}

// BeforeRead 执行请求体读取前回调。
func (f ReadAdviceFunc) BeforeRead(ctx *arkweb.Context, input ReadAdviceContext) error {
	if f.Before == nil {
		return nil
	}
	return f.Before(ctx, input)
}

// AfterRead 执行请求体读取后回调。
func (f ReadAdviceFunc) AfterRead(ctx *arkweb.Context, input ReadAdviceContext) error {
	if f.After == nil {
		return nil
	}
	return f.After(ctx, input)
}

func cleanReadAdvice(advice []ReadAdvice) []ReadAdvice {
	if len(advice) == 0 {
		return nil
	}
	out := make([]ReadAdvice, 0, len(advice))
	for _, item := range advice {
		if item != nil {
			out = append(out, item)
		}
	}
	return out
}
