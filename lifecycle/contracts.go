package lifecycle

import (
	"context"
	"strings"

	"goark.dev/goark/core/lang"
)

// Starter 表示可启动组件。
type Starter interface {
	Start(ctx context.Context) error
}

// Stopper 表示可停止组件。
type Stopper interface {
	Stop(ctx context.Context) error
}

// Closer 表示可释放资源的组件。
type Closer interface {
	Close() error
}

// Ordered 允许组件声明生命周期顺序，数值越小越早启动。
type Ordered = lang.Ordered

// PriorityOrdered 表示优先于普通 Ordered 启动的组件。
type PriorityOrdered = lang.PriorityOrdered

// Hook 描述一个生命周期组件。
type Hook struct {
	Name      string
	Order     int
	Priority  bool
	DependsOn []string
	Target    any
}

// Option 调整生命周期组件元数据。
type Option func(*Hook)

// WithOrder 设置生命周期顺序。
func WithOrder(order int) Option {
	return func(h *Hook) {
		h.Order = order
	}
}

// WithPriority 将生命周期组件标记为优先排序。
func WithPriority() Option {
	return func(h *Hook) {
		h.Priority = true
	}
}

// WithDependsOn 声明当前生命周期组件启动前必须先启动的组件名称。
func WithDependsOn(names ...string) Option {
	copied := splitHookDependencyNames(names)
	return func(h *Hook) {
		h.DependsOn = append(h.DependsOn, copied...)
	}
}

func splitHookDependencyNames(names []string) []string {
	out := make([]string, 0, len(names))
	for _, raw := range names {
		parts := strings.Split(raw, ",")
		for _, part := range parts {
			name := strings.TrimSpace(part)
			if name == "" {
				continue
			}
			out = append(out, name)
		}
	}
	return out
}
