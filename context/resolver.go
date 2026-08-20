package context

import (
	stdcontext "context"
	"reflect"

	"github.com/goark-projects/goark/container"
	arkerrors "github.com/goark-projects/goark/errors"
	"github.com/goark-projects/goark/event"
	"github.com/goark-projects/goark/lifecycle"
)

// Get 按名称解析 Bean。
func (a *ApplicationContext) Get(ctx stdcontext.Context, name string) (any, error) {
	runtimeContainer, err := a.runtimeContainer()
	if err != nil {
		return nil, err
	}
	return runtimeContainer.Get(ctx, name)
}

// GetByType 按类型解析 Bean。
func (a *ApplicationContext) GetByType(ctx stdcontext.Context, typ reflect.Type, options ...container.ResolveOption) (any, error) {
	runtimeContainer, err := a.runtimeContainer()
	if err != nil {
		return nil, err
	}
	return runtimeContainer.GetByType(ctx, typ, options...)
}

// Container 返回底层容器。
func (a *ApplicationContext) Container() (*container.Container, bool) {
	if a == nil {
		return nil, false
	}
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.container, a.container != nil
}

func (a *ApplicationContext) runtimeContainer() (*container.Container, error) {
	if a == nil {
		return nil, arkerrors.New(arkerrors.CodeInvalidArgument, "application context is nil")
	}
	a.mu.RLock()
	defer a.mu.RUnlock()
	if a.closed || a.closing {
		return nil, arkerrors.New(arkerrors.CodeClosed, "application context is closed")
	}
	if !a.refreshed || a.container == nil {
		return nil, arkerrors.New(arkerrors.CodeConflict, "application context has not been refreshed")
	}
	return a.container, nil
}

func (a *ApplicationContext) runtime() (*lifecycle.Manager, *event.Bus, error) {
	runtimeContainer, err := a.runtimeContainer()
	if err != nil {
		return nil, nil, err
	}
	_ = runtimeContainer

	a.mu.RLock()
	defer a.mu.RUnlock()
	if a.lifecycle == nil || a.events == nil {
		return nil, nil, arkerrors.New(arkerrors.CodeConflict, "application context runtime is incomplete")
	}
	return a.lifecycle, a.events, nil
}
