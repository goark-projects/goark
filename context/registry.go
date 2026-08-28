package context

import (
	"goark.dev/goark/container"
	arkerrors "goark.dev/goark/errors"
)

// RegisterDefinition 注册 Bean 定义，必须在 Refresh 前调用。
func (a *ApplicationContext) RegisterDefinition(definition container.Definition) error {
	if a == nil {
		return arkerrors.New(arkerrors.CodeInvalidArgument, "application context is nil")
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.closed || a.closing {
		return arkerrors.New(arkerrors.CodeClosed, "application context is closed")
	}
	if a.refreshed || a.refreshing {
		return arkerrors.New(arkerrors.CodeConflict, "application context has already been refreshed")
	}
	return a.registry.Register(definition)
}
