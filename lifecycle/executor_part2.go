package lifecycle

import (
	"context"
	stderrors "errors"
	"sort"
	"strings"

	arkerrors "goark.dev/goark/errors"
)

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
				arkerrors.Wrapf(
					arkerrors.CodeLifecycle,
					err,
					"failed to start lifecycle hook %q",
					hook.Name,
				),
				rollbackErr,
			)
		}
		started = append(started, hook)
	}
	return started, nil
}

func lifecycleDependencyCycleDescription(
	hooks []Hook,
	emitted []bool,
	indicesByName map[string]int,
) string {
	parts := make([]string, 0)
	for index := range hooks {
		if emitted[index] {
			continue
		}
		for _, dependency := range hooks[index].DependsOn {
			dependencyIndex, exists := indicesByName[dependency]
			if !exists || emitted[dependencyIndex] {
				continue
			}
			parts = append(parts, hooks[index].Name+" -> "+dependency)
		}
	}
	if len(parts) == 0 {
		for index := range hooks {
			if !emitted[index] {
				parts = append(parts, hooks[index].Name)
			}
		}
	}
	sort.Strings(parts)
	return strings.Join(parts, "; ")
}

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

func stopHooks(ctx context.Context, hooks []Hook) error {
	var joined error
	for i := len(hooks) - 1; i >= 0; i-- {
		hook := hooks[i]
		if stopper, ok := hook.Target.(Stopper); ok {
			if err := stopper.Stop(ctx); err != nil {
				joined = stderrors.Join(
					joined,
					arkerrors.Wrapf(
						arkerrors.CodeLifecycle,
						err,
						"failed to stop lifecycle hook %q",
						hook.Name,
					),
				)
			}
		}
	}
	return joined
}

func (m *Manager) beginStart() ([]Hook, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	switch m.state {
	case stateRunning:
		return nil, nil
	case stateStopped:
		hooks, err := sortedHooks(m.hooks)
		if err != nil {
			return nil, err
		}
		m.state = stateStarting
		return hooks, nil
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

func (m *Manager) finishStop() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.started = nil
	m.state = stateStopped
}

func (m *Manager) finishClose() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.started = nil
	m.state = stateClosed
}
