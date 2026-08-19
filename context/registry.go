package context

import (
	"github.com/goark-projects/goark/container"
	arkerrors "github.com/goark-projects/goark/errors"
)

// RegisterDefinition 注册 Bean 定义，必须在 Refresh 前调用。
func (a *ApplicationContext) RegisterDefinition(definition container.Definition) error {
	if a == nil {
		return arkerrors.New(arkerrors.CodeInvalidArgument, "application context is nil")
	}
	a.mu.RLock()
	if a.closed || a.closing {
		a.mu.RUnlock()
		return arkerrors.New(arkerrors.CodeClosed, "application context is closed")
	}
	if a.refreshed || a.refreshing {
		a.mu.RUnlock()
		return arkerrors.New(arkerrors.CodeConflict, "application context has already been refreshed")
	}
	registry := a.registry
	a.mu.RUnlock()
	return registry.Register(definition)
}
