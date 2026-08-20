package lifecycle

import (
	"sync"

	"github.com/goark-projects/goark/core/util"
	arkerrors "github.com/goark-projects/goark/errors"
	"github.com/goark-projects/goark/internal/reflectx"
)

type managerState uint8

const (
	stateStopped managerState = iota
	stateStarting
	stateRunning
	stateStopping
	stateClosed
)

// Manager 负责按顺序启动、按反序停止并最终关闭组件。
type Manager struct {
	mu      sync.Mutex
	hooks   []Hook
	started []Hook
	state   managerState
}

// NewManager 创建生命周期管理器。
func NewManager() *Manager {
	return &Manager{}
}

// Register 注册一个生命周期组件。
func (m *Manager) Register(name string, target any, options ...Option) error {
	if m == nil {
		return arkerrors.New(arkerrors.CodeInvalidArgument, "lifecycle manager is nil")
	}
	hook, err := newHook(name, target, options...)
	if err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	if m.state == stateClosed {
		return arkerrors.New(arkerrors.CodeClosed, "lifecycle manager is closed")
	}
	if m.state != stateStopped {
		return arkerrors.New(arkerrors.CodeLifecycle, "lifecycle manager is not stopped")
	}
	for _, existing := range m.hooks {
		if existing.Name == name {
			return arkerrors.Newf(arkerrors.CodeAlreadyExists, "lifecycle hook %q already exists", name)
		}
	}
	m.hooks = append(m.hooks, hook)
	return nil
}

// Running 返回生命周期管理器当前是否处于运行态。
func (m *Manager) Running() bool {
	if m == nil {
		return false
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.state == stateRunning
}

func newHook(name string, target any, options ...Option) (Hook, error) {
	if name == "" {
		return Hook{}, arkerrors.New(arkerrors.CodeInvalidArgument, "lifecycle hook name is empty")
	}
	if reflectx.IsNil(target) {
		return Hook{}, arkerrors.Newf(arkerrors.CodeInvalidArgument, "lifecycle hook %q target is nil", name)
	}
	if _, ok := target.(Starter); !ok {
		if _, ok := target.(Stopper); !ok {
			if _, ok := target.(Closer); !ok {
				return Hook{}, arkerrors.Newf(arkerrors.CodeInvalidArgument, "lifecycle hook %q does not implement lifecycle contracts", name)
			}
		}
	}

	hook := Hook{
		Name:     name,
		Order:    util.OrderOf(target),
		Priority: util.IsPriorityOrdered(target),
		Target:   target,
	}
	for _, option := range options {
		if option != nil {
			option(&hook)
		}
	}
	return hook, nil
}
