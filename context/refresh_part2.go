package context

import (
	stdcontext "context"
	"sort"

	"goark.dev/goark/container"
	coreenv "goark.dev/goark/core/env"
	"goark.dev/goark/core/util"
	arkerrors "goark.dev/goark/errors"
	"goark.dev/goark/event"
	"goark.dev/goark/internal/reflectx"
	"goark.dev/goark/lifecycle"
)

func registerRuntimeHooks(
	manager *lifecycle.Manager,
	events *event.Bus,
	instances map[string]any,
	names []string,
	definitions []container.Definition,
) error {
	names = completeRuntimeHookNames(instances, names)
	managed := runtimeLifecycleManagement(definitions)
	lifecycleNames := runtimeLifecycleNames(instances, names, managed)
	dependencies := runtimeHookDependencies(definitions, lifecycleNames)
	for _, name := range names {
		instance := instances[name]
		if managed[name] && isLifecycleTarget(instance) {
			if err := manager.Register(name, instance, lifecycle.WithDependsOn(
				dependencies[name]...)); err != nil {
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
	runtimeContainer, manager, err := buildRuntime(
		ctx,
		plan.registry,
		plan.events,
		plan.allowCircularReferences,
	)
	a.finishRefresh(plan.registry, runtimeContainer, manager, err)
	if err != nil {
		return err
	}
	return plan.events.Publish(ctx, RefreshedEvent{Source: a})
}

func buildRuntime(
	ctx stdcontext.Context,
	registry *container.Registry,
	events *event.Bus,
	allowCircularReferences bool,
) (*container.Container, *lifecycle.Manager, error) {
	runtimeContainer, err := container.New(
		registry,
		container.WithAllowCircularReferences(allowCircularReferences),
	)
	if err != nil {
		return nil, nil, err
	}
	if err := runtimeContainer.InitializeSingletons(ctx); err != nil {
		return nil, nil, err
	}
	manager := lifecycle.NewManager()
	if err := registerRuntimeHooks(
		manager,
		events,
		runtimeContainer.SingletonInstances(),
		runtimeContainer.SingletonNamesInStartupOrder(),
		registry.Definitions(),
	); err != nil {
		return nil, nil, err
	}
	return runtimeContainer, manager, nil
}

func runtimeHookDependencies(
	definitions []container.Definition,
	lifecycleNames map[string]struct{},
) map[string][]string {
	dependencies := make(map[string][]string, len(definitions))
	for _, definition := range definitions {
		if _, ok := lifecycleNames[definition.Name]; !ok {
			continue
		}
		names := make([]string, 0, len(definition.DependencyDescriptors))
		for _, descriptor := range definition.DependencyDescriptors {
			if _, ok := lifecycleNames[descriptor.Name]; !ok {
				continue
			}
			if descriptor.Name == "" || containsRuntimeHookDependency(names, descriptor.Name) {
				continue
			}
			names = append(names, descriptor.Name)
		}
		dependencies[definition.Name] = names
	}
	removeCyclicRuntimeHookDependencies(dependencies)
	return dependencies
}

func completeRuntimeHookNames(instances map[string]any, names []string) []string {
	completed := make([]string, 0, len(instances))
	seen := make(map[string]struct{}, len(instances))
	for _, name := range names {
		if _, exists := instances[name]; !exists {
			continue
		}
		completed = append(completed, name)
		seen[name] = struct{}{}
	}
	remaining := make([]string, 0)
	for name := range instances {
		if _, exists := seen[name]; exists {
			continue
		}
		remaining = append(remaining, name)
	}
	sort.Strings(remaining)
	return append(completed, remaining...)
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

type refreshPlan struct {
	registry                *container.Registry
	env                     coreenv.ConfigurableEnvironment
	configurations          []Configuration
	events                  *event.Bus
	allowCircularReferences bool
	skip                    bool
}

func containsRuntimeHookDependency(names []string, target string) bool {
	for _, name := range names {
		if name == target {
			return true
		}
	}
	return false
}
