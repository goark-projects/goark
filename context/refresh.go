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

type refreshPlan struct {
	registry                *container.Registry
	env                     coreenv.ConfigurableEnvironment
	configurations          []Configuration
	events                  *event.Bus
	allowCircularReferences bool
	skip                    bool
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
	runtimeContainer, manager, err := buildRuntime(ctx, plan.registry, plan.events, plan.allowCircularReferences)
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
		registry:                registry,
		env:                     a.env,
		configurations:          configurations,
		events:                  a.events,
		allowCircularReferences: a.allowCircularReferences,
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

func buildRuntime(ctx stdcontext.Context, registry *container.Registry, events *event.Bus, allowCircularReferences bool) (*container.Container, *lifecycle.Manager, error) {
	runtimeContainer, err := container.New(registry, container.WithAllowCircularReferences(allowCircularReferences))
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

func registerRuntimeHooks(manager *lifecycle.Manager, events *event.Bus, instances map[string]any, names []string, definitions []container.Definition) error {
	names = completeRuntimeHookNames(instances, names)
	lifecycleNames := runtimeLifecycleNames(instances, names)
	dependencies := runtimeHookDependencies(definitions, lifecycleNames)
	for _, name := range names {
		instance := instances[name]
		if isLifecycleTarget(instance) {
			if err := manager.Register(name, instance, lifecycle.WithDependsOn(dependencies[name]...)); err != nil {
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

func runtimeLifecycleNames(instances map[string]any, names []string) map[string]struct{} {
	lifecycleNames := make(map[string]struct{}, len(instances))
	for _, name := range names {
		instance, exists := instances[name]
		if exists && isLifecycleTarget(instance) {
			lifecycleNames[name] = struct{}{}
		}
	}
	return lifecycleNames
}

func runtimeHookDependencies(definitions []container.Definition, lifecycleNames map[string]struct{}) map[string][]string {
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

func removeCyclicRuntimeHookDependencies(dependencies map[string][]string) {
	components := runtimeHookDependencyComponents(dependencies)
	for _, component := range components {
		if len(component) == 1 && !containsRuntimeHookDependency(dependencies[component[0]], component[0]) {
			continue
		}
		members := make(map[string]struct{}, len(component))
		for _, name := range component {
			members[name] = struct{}{}
		}
		for _, name := range component {
			filtered := dependencies[name][:0]
			for _, dependency := range dependencies[name] {
				if _, cyclic := members[dependency]; cyclic {
					continue
				}
				filtered = append(filtered, dependency)
			}
			dependencies[name] = filtered
		}
	}
}

func runtimeHookDependencyComponents(dependencies map[string][]string) [][]string {
	names := make([]string, 0, len(dependencies))
	for name := range dependencies {
		names = append(names, name)
	}
	sort.Strings(names)

	index := 0
	stack := make([]string, 0, len(names))
	onStack := make(map[string]bool, len(names))
	indices := make(map[string]int, len(names))
	lowlink := make(map[string]int, len(names))
	for _, name := range names {
		indices[name] = -1
	}

	components := make([][]string, 0)
	var strongConnect func(string)
	strongConnect = func(name string) {
		indices[name] = index
		lowlink[name] = index
		index++
		stack = append(stack, name)
		onStack[name] = true

		for _, dependency := range dependencies[name] {
			if _, exists := dependencies[dependency]; !exists {
				continue
			}
			if indices[dependency] == -1 {
				strongConnect(dependency)
				if lowlink[dependency] < lowlink[name] {
					lowlink[name] = lowlink[dependency]
				}
				continue
			}
			if onStack[dependency] && indices[dependency] < lowlink[name] {
				lowlink[name] = indices[dependency]
			}
		}

		if lowlink[name] != indices[name] {
			return
		}
		component := make([]string, 0)
		for {
			last := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			onStack[last] = false
			component = append(component, last)
			if last == name {
				break
			}
		}
		sort.Strings(component)
		components = append(components, component)
	}

	for _, name := range names {
		if indices[name] == -1 {
			strongConnect(name)
		}
	}
	sort.Slice(components, func(i, j int) bool {
		return components[i][0] < components[j][0]
	})
	return components
}

func containsRuntimeHookDependency(names []string, target string) bool {
	for _, name := range names {
		if name == target {
			return true
		}
	}
	return false
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
