package event

import (
	"context"
	"reflect"
	"sort"
	"sync"
	"sync/atomic"

	"goark.dev/goark/core/util"
	arkerrors "goark.dev/goark/errors"
	"goark.dev/goark/internal/reflectx"
)

// Handler 处理同步事件。
type Handler interface {
	HandleEvent(ctx context.Context, evt any) error
}

// HandlerFunc 将函数适配为事件处理器。
type HandlerFunc func(ctx context.Context, evt any) error

func (f HandlerFunc) HandleEvent(ctx context.Context, evt any) error {
	return f(ctx, evt)
}

// Subscription 描述一个事件订阅。
type Subscription struct {
	Name      string
	EventType reflect.Type
	Order     int
	Priority  bool
	Handler   Handler
}

// Option 调整事件订阅元数据。
type Option func(*Subscription)

// WithName 设置订阅名称。
func WithName(name string) Option {
	return func(s *Subscription) {
		s.Name = name
	}
}

// WithOrder 设置订阅顺序。
func WithOrder(order int) Option {
	return func(s *Subscription) {
		s.Order = order
	}
}

// WithPriority 将订阅标记为优先排序。
func WithPriority() Option {
	return func(s *Subscription) {
		s.Priority = true
	}
}

// WithEventType 限制订阅处理的事件类型。
func WithEventType(t reflect.Type) Option {
	return func(s *Subscription) {
		s.EventType = t
	}
}

// Bus 是同步事件总线，按订阅顺序串行分发事件。
type Bus struct {
	mu       sync.RWMutex
	seq      atomic.Uint64
	handlers []registeredHandler
}

type registeredHandler struct {
	Subscription
	seq uint64
}

// NewBus 创建事件总线。
func NewBus() *Bus {
	return &Bus{}
}

// Subscribe 注册事件处理器。
func (b *Bus) Subscribe(handler Handler, options ...Option) error {
	if b == nil {
		return arkerrors.New(arkerrors.CodeInvalidArgument, "event bus is nil")
	}
	if reflectx.IsNil(handler) {
		return arkerrors.New(arkerrors.CodeInvalidArgument, "event handler is nil")
	}

	subscription := Subscription{
		Order:    util.OrderOf(handler),
		Priority: util.IsPriorityOrdered(handler),
		Handler:  handler,
	}
	for _, option := range options {
		if option != nil {
			option(&subscription)
		}
	}
	if subscription.Name == "" {
		subscription.Name = reflect.TypeOf(handler).String()
	}

	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers = append(b.handlers, registeredHandler{
		Subscription: subscription,
		seq:          b.seq.Add(1),
	})
	return nil
}

// Subscribe 注册指定类型事件的处理函数。
func Subscribe[T any](b *Bus, handler func(context.Context, T) error, options ...Option) error {
	if handler == nil {
		return arkerrors.New(arkerrors.CodeInvalidArgument, "typed event handler is nil")
	}
	options = append([]Option{WithEventType(reflectx.TypeOf[T]())}, options...)
	return b.Subscribe(HandlerFunc(func(ctx context.Context, evt any) error {
		typed, ok := evt.(T)
		if !ok {
			return arkerrors.Newf(arkerrors.CodeTypeMismatch, "event %T cannot be used as %s", evt, reflectx.TypeOf[T]())
		}
		return handler(ctx, typed)
	}), options...)
}

// Publish 同步发布事件，任一处理器失败即停止后续分发。
func (b *Bus) Publish(ctx context.Context, evt any) error {
	if b == nil {
		return arkerrors.New(arkerrors.CodeInvalidArgument, "event bus is nil")
	}
	if ctx == nil {
		return arkerrors.New(arkerrors.CodeInvalidArgument, "context is nil")
	}
	if evt == nil {
		return arkerrors.New(arkerrors.CodeInvalidArgument, "event is nil")
	}
	if err := ctx.Err(); err != nil {
		return arkerrors.Wrap(arkerrors.CodeLifecycle, err, "event publish canceled")
	}

	handlers := b.matchingHandlers(reflect.TypeOf(evt))
	for _, registered := range handlers {
		if err := ctx.Err(); err != nil {
			return arkerrors.Wrap(arkerrors.CodeLifecycle, err, "event publish canceled")
		}
		if err := registered.Handler.HandleEvent(ctx, evt); err != nil {
			return arkerrors.Wrapf(arkerrors.CodeLifecycle, err, "event handler %q failed", registered.Name)
		}
	}
	return nil
}

func (b *Bus) matchingHandlers(eventType reflect.Type) []registeredHandler {
	b.mu.RLock()
	copied := append([]registeredHandler(nil), b.handlers...)
	b.mu.RUnlock()

	matched := copied[:0]
	for _, registered := range copied {
		if registered.EventType == nil || eventMatches(eventType, registered.EventType) {
			matched = append(matched, registered)
		}
	}
	sort.SliceStable(matched, func(i, j int) bool {
		if matched[i].Priority != matched[j].Priority {
			return matched[i].Priority
		}
		if matched[i].Order == matched[j].Order {
			return matched[i].seq < matched[j].seq
		}
		return matched[i].Order < matched[j].Order
	})
	return matched
}

func eventMatches(actual reflect.Type, expected reflect.Type) bool {
	if actual == expected {
		return true
	}
	if actual.AssignableTo(expected) {
		return true
	}
	if expected.Kind() == reflect.Interface && actual.Implements(expected) {
		return true
	}
	return false
}
