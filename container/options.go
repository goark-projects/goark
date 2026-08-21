package container

// ContainerOption 调整容器运行时行为。
type ContainerOption func(*containerOptions)

type containerOptions struct {
	allowCircularReferences bool
}

func newContainerOptions(options []ContainerOption) containerOptions {
	var out containerOptions
	for _, option := range options {
		if option != nil {
			option(&out)
		}
	}
	return out
}

// WithAllowCircularReferences 设置是否允许单例字段注入循环依赖。
func WithAllowCircularReferences(allow bool) ContainerOption {
	return func(options *containerOptions) {
		options.allowCircularReferences = allow
	}
}
