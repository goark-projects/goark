package lifecycle

import (
	"context"
	stderrors "errors"
	"sort"

	arkerrors "github.com/goark-projects/goark/errors"
)

// Start 启动所有组件；若中途失败，会回滚已启动组件。
func (m *Manager) Start(ctx context.Context) error {
	if m == nil {
		return arkerrors.New(arkerrors.CodeInvalidArgument, "lifecycle manager is nil")
	}
	if ctx == nil {
		return arkerrors.New(arkerrors.CodeInvalidArgument, "context is nil")
	}
	if err := ctx.Err(); err != nil {
		return arkerrors.Wrap(arkerrors.CodeLifecycle, err, "lifecycle start canceled")
	}

	hooks, err := m.beginStart()
	if err != nil {
		return err
	}
	if hooks == nil {
		return nil
	}

	started, err := startHooks(ctx, hooks)
	m.finishStart(started, err)
	return err
}

// Stop 停止所有已启动组件，不释放 Close 资源。
func (m *Manager) Stop(ctx context.Context) error {
	if m == nil {
		return arkerrors.New(arkerrors.CodeInvalidArgument, "lifecycle manager is nil")
	}
	if ctx == nil {
		return arkerrors.New(arkerrors.CodeInvalidArgument, "context is nil")
	}

	started, err := m.beginStop()
	if err != nil {
		return err
	}
	if started == nil {
		return nil
	}

	err = stopHooks(ctx, started)
	m.finishStop()
	return err
}

// Close 停止运行中组件，并按反序释放所有已注册资源。
func (m *Manager) Close(ctx context.Context) error {
	if m == nil {
		return arkerrors.New(arkerrors.CodeInvalidArgument, "lifecycle manager is nil")
	}
	if ctx == nil {
		return arkerrors.New(arkerrors.CodeInvalidArgument, "context is nil")
	}

	started, hooks, err := m.beginClose()
	if err != nil {
		return err
	}
	if hooks == nil {
		return nil
	}

	stopErr := stopHooks(ctx, started)
	closeErr := closeHooks(hooks)
	m.finishClose()
	return stderrors.Join(stopErr, closeErr)
}

func (m *Manager) beginStart() ([]Hook, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	switch m.state {
	case stateRunning:
		return nil, nil
	case stateStopped:
		m.state = stateStarting
		return sortedHooks(m.hooks), nil
	case stateClosed:
		return nil, arkerrors.New(arkerrors.CodeClosed, "lifecycle manager is closed")
	default:
		return nil, arkerrors.New(arkerrors.CodeConflict, "lifecycle manager is busy")
	}
}

func (m *Manager) finishStart(started []Hook, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err != nil {
		m.started = nil
		m.state = stateStopped
		return
	}
	m.started = append([]Hook(nil), started...)
	m.state = stateRunning
}

func (m *Manager) beginStop() ([]Hook, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	switch m.state {
	case stateStopped:
		return nil, nil
	case stateRunning:
		m.state = stateStopping
		return append([]Hook(nil), m.started...), nil
	case stateClosed:
		return nil, nil
	default:
		return nil, arkerrors.New(arkerrors.CodeConflict, "lifecycle manager is busy")
	}
}

func (m *Manager) finishStop() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.started = nil
	m.state = stateStopped
}

func (m *Manager) beginClose() ([]Hook, []Hook, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	switch m.state {
	case stateClosed:
		return nil, nil, nil
	case stateRunning:
		m.state = stateStopping
		return append([]Hook(nil), m.started...), sortedHooks(m.hooks), nil
	case stateStopped:
		m.state = stateStopping
		return nil, sortedHooks(m.hooks), nil
	default:
		return nil, nil, arkerrors.New(arkerrors.CodeConflict, "lifecycle manager is busy")
	}
}

func (m *Manager) finishClose() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.started = nil
	m.state = stateClosed
}

func startHooks(ctx context.Context, hooks []Hook) ([]Hook, error) {
	started := make([]Hook, 0, len(hooks))
	for _, hook := range hooks {
		if err := ctx.Err(); err != nil {
			rollbackErr := stopHooks(ctx, started)
			return nil, stderrors.Join(
				arkerrors.Wrap(arkerrors.CodeLifecycle, err, "lifecycle start canceled"),
				rollbackErr,
			)
		}
		starter, ok := hook.Target.(Starter)
		if !ok {
			started = append(started, hook)
			continue
		}
		if err := starter.Start(ctx); err != nil {
			rollbackErr := stopHooks(ctx, started)
			return nil, stderrors.Join(
				arkerrors.Wrapf(arkerrors.CodeLifecycle, err, "failed to start lifecycle hook %q", hook.Name),
				rollbackErr,
			)
		}
		started = append(started, hook)
	}
	return started, nil
}

func stopHooks(ctx context.Context, hooks []Hook) error {
	var joined error
	for i := len(hooks) - 1; i >= 0; i-- {
		hook := hooks[i]
		if stopper, ok := hook.Target.(Stopper); ok {
			if err := stopper.Stop(ctx); err != nil {
				joined = stderrors.Join(joined, arkerrors.Wrapf(arkerrors.CodeLifecycle, err, "failed to stop lifecycle hook %q", hook.Name))
			}
		}
	}
	return joined
}

func closeHooks(hooks []Hook) error {
	var joined error
	for i := len(hooks) - 1; i >= 0; i-- {
		hook := hooks[i]
		if closer, ok := hook.Target.(Closer); ok {
			if err := closer.Close(); err != nil {
				joined = stderrors.Join(joined, arkerrors.Wrapf(arkerrors.CodeLifecycle, err, "failed to close lifecycle hook %q", hook.Name))
			}
		}
	}
	return joined
}

func sortedHooks(hooks []Hook) []Hook {
	copied := append([]Hook(nil), hooks...)
	sort.SliceStable(copied, func(i, j int) bool {
		if copied[i].Order == copied[j].Order {
			return copied[i].Name < copied[j].Name
		}
		return copied[i].Order < copied[j].Order
	})
	return copied
}
