package context

import (
	stdcontext "context"
	stderrors "errors"

	arkerrors "goark.dev/goark/errors"
	"goark.dev/goark/event"
	"goark.dev/goark/lifecycle"
)

// Start 启动应用上下文生命周期；未 Refresh 时会先 Refresh。
func (a *ApplicationContext) Start(ctx stdcontext.Context) error {
	if err := a.Refresh(ctx); err != nil {
		return err
	}

	manager, events, err := a.runtime()
	if err != nil {
		return err
	}
	wasRunning := manager.Running()
	if err := manager.Start(ctx); err != nil {
		return err
	}
	if wasRunning {
		return nil
	}
	return events.Publish(ctx, StartedEvent{Source: a})
}

// Stop 停止应用上下文生命周期。
func (a *ApplicationContext) Stop(ctx stdcontext.Context) error {
	manager, events, err := a.runtime()
	if err != nil {
		return err
	}
	wasRunning := manager.Running()
	if err := manager.Stop(ctx); err != nil {
		return err
	}
	if !wasRunning {
		return nil
	}
	return events.Publish(ctx, StoppedEvent{Source: a})
}

// Close 停止生命周期、释放资源并关闭上下文。
func (a *ApplicationContext) Close(ctx stdcontext.Context) error {
	if a == nil {
		return arkerrors.New(arkerrors.CodeInvalidArgument, "application context is nil")
	}
	if ctx == nil {
		return arkerrors.New(arkerrors.CodeInvalidArgument, "context is nil")
	}

	manager, events, refreshed, closed, err := a.beginClose()
	if err != nil {
		return err
	}
	if closed {
		return nil
	}
	if !refreshed {
		a.markClosed()
		return events.Publish(ctx, ClosedEvent{Source: a})
	}

	wasRunning := manager.Running()
	var joined error
	if wasRunning {
		stopErr := manager.Stop(ctx)
		joined = stderrors.Join(joined, stopErr)
		if stopErr == nil {
			joined = stderrors.Join(joined, events.Publish(ctx, StoppedEvent{Source: a}))
		}
	}
	a.markClosed()
	joined = stderrors.Join(joined, events.Publish(ctx, ClosedEvent{Source: a}))
	joined = stderrors.Join(joined, manager.Close(ctx))
	return joined
}

func (a *ApplicationContext) beginClose() (*lifecycle.Manager, *event.Bus, bool, bool, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.closed || a.closing {
		return nil, nil, false, true, nil
	}
	a.closing = true
	return a.lifecycle, a.events, a.refreshed, false, nil
}

func (a *ApplicationContext) markClosed() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.closed = true
	a.closing = false
}
