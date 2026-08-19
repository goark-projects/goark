package lifecycle

import "context"

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
type Ordered interface {
	Order() int
}

// Hook 描述一个生命周期组件。
type Hook struct {
	Name   string
	Order  int
	Target any
}

// Option 调整生命周期组件元数据。
type Option func(*Hook)

// WithOrder 设置生命周期顺序。
func WithOrder(order int) Option {
	return func(h *Hook) {
		h.Order = order
	}
}
