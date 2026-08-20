package context

import (
	stdcontext "context"
	"sort"

	"github.com/goark-projects/goark/container"
	coreenv "github.com/goark-projects/goark/core/env"
	"github.com/goark-projects/goark/core/util"
	arkerrors "github.com/goark-projects/goark/errors"
	"github.com/goark-projects/goark/event"
	"github.com/goark-projects/goark/internal/reflectx"
	"github.com/goark-projects/goark/lifecycle"
)

type refreshPlan struct {
	registry       *container.Registry
	env            coreenv.ConfigurableEnvironment
	configurations []Configuration
	events         *event.Bus
	skip           bool
}

// Refresh 构建容器并初始化所有非延迟单例。
func (a *ApplicationContext) Refresh(ctx stdcontext.Context) error {
	if a == nil {
		return arkerrors.New(arkerrors.CodeInvalidArgument, "application context is nil")
	}
	if ctx == nil {
		return arkerrors.New(arkerrors.CodeInvalidArgument, "context is nil")
	}

	plan, err := a.beginRefresh()
	if err != nil {
		return err
	}
	if plan.skip {
		return nil
	}

	if err := applyConfigurations(ctx, plan.env, plan.registry, plan.configurations); err != nil {
		a.finishRefresh(nil, nil, nil, err)
		return err
	}
	runtimeContainer, manager, err := buildRuntime(ctx, plan.registry, plan.events)
	a.finishRefresh(plan.registry, runtimeContainer, manager, err)
	if err != nil {
		return err
	}
	return plan.events.Publish(ctx, RefreshedEvent{Source: a})
}

func (a *ApplicationContext) beginRefresh() (refreshPlan, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.closed || a.closing {
		return refreshPlan{}, arkerrors.New(arkerrors.CodeClosed, "application context is closed")
	}
	if a.refreshed {
		return refreshPlan{skip: true}, nil
	}
	if a.refreshing {
		return refreshPlan{}, arkerrors.New(arkerrors.CodeConflict, "application context is refreshing")
	}
	registry, err := cloneRegistry(a.registry)
	if err != nil {
		return refreshPlan{}, err
	}
	a.refreshing = true
	configurations := make([]Configuration, 0, len(a.configurations))
	for _, configuration := range a.configurations {
		configurations = append(configurations, configuration)
	}
	return refreshPlan{
		registry:       registry,
		env:            a.env,
		configurations: configurations,
		events:         a.events,
	}, nil
}

func (a *ApplicationContext) finishRefresh(registry *container.Registry, runtimeContainer *container.Container, manager *lifecycle.Manager, err error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if err != nil {
		a.refreshing = false
		return
	}
	a.registry = registry
	a.container = runtimeContainer
	a.lifecycle = manager
	a.refreshed = true
	a.refreshing = false
}

func cloneRegistry(source *container.Registry) (*container.Registry, error) {
	if source == nil {
		return nil, arkerrors.New(arkerrors.CodeInvalidArgument, "bean registry is nil")
	}
	cloned := container.NewRegistry()
	for _, definition := range source.Definitions() {
		if err := cloned.Register(definition); err != nil {
			return nil, err
		}
	}
	return cloned, nil
}

func buildRuntime(ctx stdcontext.Context, registry *container.Registry, events *event.Bus) (*container.Container, *lifecycle.Manager, error) {
	runtimeContainer, err := container.New(registry)
	if err != nil {
		return nil, nil, err
	}
	if err := runtimeContainer.InitializeSingletons(ctx); err != nil {
		return nil, nil, err
	}
	manager := lifecycle.NewManager()
	if err := registerRuntimeHooks(manager, events, runtimeContainer.SingletonInstances()); err != nil {
		return nil, nil, err
	}
	return runtimeContainer, manager, nil
}

func registerRuntimeHooks(manager *lifecycle.Manager, events *event.Bus, instances map[string]any) error {
	names := make([]string, 0, len(instances))
	for name := range instances {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		instance := instances[name]
		if isLifecycleTarget(instance) {
			if err := manager.Register(name, instance); err != nil {
				return err
			}
		}
		if handler, ok := instance.(event.Handler); ok {
			options := []event.Option{event.WithName(name)}
			if ordered, ok := instance.(lifecycle.Ordered); ok {
				options = append(options, event.WithOrder(ordered.Order()))
			}
			if util.IsPriorityOrdered(instance) {
				options = append(options, event.WithPriority())
			}
			if err := events.Subscribe(handler, options...); err != nil {
				return err
			}
		}
	}
	return nil
}

func isLifecycleTarget(value any) bool {
	if reflectx.IsNil(value) {
		return false
	}
	if _, ok := value.(lifecycle.Starter); ok {
		return true
	}
	if _, ok := value.(lifecycle.Stopper); ok {
		return true
	}
	if _, ok := value.(lifecycle.Closer); ok {
		return true
	}
	return false
}
