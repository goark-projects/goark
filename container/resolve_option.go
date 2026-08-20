package container

import (
	"strings"

	arkerrors "github.com/goark-projects/goark/errors"
)

// ResolveOption 调整按类型解析 Bean 时的候选选择规则。
type ResolveOption func(*resolveOptions)

type resolveOptions struct {
	qualifier string
	qualified bool
}

// WithQualifier 指定按类型解析时优先使用的 Bean 名称。
func WithQualifier(name string) ResolveOption {
	return func(options *resolveOptions) {
		options.qualifier = strings.TrimSpace(name)
		options.qualified = true
	}
}

func newResolveOptions(options []ResolveOption) (resolveOptions, error) {
	out := resolveOptions{}
	for _, option := range options {
		if option != nil {
			option(&out)
		}
	}
	if out.qualified && out.qualifier == "" {
		return resolveOptions{}, arkerrors.New(arkerrors.CodeInvalidArgument, "bean qualifier is empty")
	}
	return out, nil
}
