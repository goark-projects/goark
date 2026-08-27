package lifecycle

import (
	"context"
	stderrors "errors"
	"sort"
	"strings"

	arkerrors "goark.dev/goark/errors"
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

func (m *Manager) beginStop() ([]Hook, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	switch m.state {
	case stateStopped:
		return nil, nil
	case stateRunning:
		m.state = stateStopping
		return append([]Hook{}, m.started...), nil
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
		hooks, err := sortedHooks(m.hooks)
		if err != nil {
			return nil, nil, err
		}
		m.state = stateStopping
		return append([]Hook(nil), m.started...), hooks, nil
	case stateStopped:
		hooks, err := sortedHooks(m.hooks)
		if err != nil {
			return nil, nil, err
		}
		m.state = stateStopping
		return nil, hooks, nil
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

func sortedHooks(hooks []Hook) ([]Hook, error) {
	copied := append([]Hook{}, hooks...)
	sort.SliceStable(copied, func(i, j int) bool {
		return lessHook(copied[i], copied[j])
	})
	return dependencySortedHooks(copied)
}

func dependencySortedHooks(hooks []Hook) ([]Hook, error) {
	if len(hooks) == 0 {
		return hooks, nil
	}
	indicesByName := make(map[string]int, len(hooks))
	for index, hook := range hooks {
		indicesByName[hook.Name] = index
	}

	dependents := make(map[int][]int, len(hooks))
	inDegree := make([]int, len(hooks))
	seenEdges := make(map[[2]int]struct{})
	for index, hook := range hooks {
		for _, dependency := range hook.DependsOn {
			dependencyIndex, exists := indicesByName[dependency]
			if !exists {
				continue
			}
			key := [2]int{dependencyIndex, index}
			if _, exists := seenEdges[key]; exists {
				continue
			}
			seenEdges[key] = struct{}{}
			dependents[dependencyIndex] = append(dependents[dependencyIndex], index)
			inDegree[index]++
		}
	}
	for index := range dependents {
		sortHookIndices(dependents[index], hooks)
	}

	ready := make([]int, 0, len(hooks))
	for index, degree := range inDegree {
		if degree == 0 {
			ready = append(ready, index)
		}
	}
	sortHookIndices(ready, hooks)

	sorted := make([]Hook, 0, len(hooks))
	emitted := make([]bool, len(hooks))
	for len(sorted) < len(hooks) {
		if len(ready) == 0 {
			return nil, arkerrors.Newf(arkerrors.CodeCircularDependency, "circular lifecycle dependency detected: %s", lifecycleDependencyCycleDescription(hooks, emitted, indicesByName))
		}
		index := ready[0]
		ready = ready[1:]
		if emitted[index] {
			continue
		}
		emitted[index] = true
		sorted = append(sorted, hooks[index])
		for _, dependent := range dependents[index] {
			if emitted[dependent] {
				continue
			}
			inDegree[dependent]--
			if inDegree[dependent] <= 0 {
				ready = append(ready, dependent)
			}
		}
		sortHookIndices(ready, hooks)
	}
	return sorted, nil
}

func lifecycleDependencyCycleDescription(hooks []Hook, emitted []bool, indicesByName map[string]int) string {
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

func sortHookIndices(indices []int, hooks []Hook) {
	sort.SliceStable(indices, func(i, j int) bool {
		return lessHook(hooks[indices[i]], hooks[indices[j]])
	})
}

func lessHook(left Hook, right Hook) bool {
	if left.Priority != right.Priority {
		return left.Priority
	}
	if left.Order == right.Order {
		return left.Name < right.Name
	}
	return left.Order < right.Order
}
