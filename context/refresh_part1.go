package context

import (
	"sort"

	"goark.dev/goark/container"
	arkerrors "goark.dev/goark/errors"
	"goark.dev/goark/lifecycle"
)

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
		return refreshPlan{}, arkerrors.New(
			arkerrors.CodeConflict,
			"application context is refreshing",
		)
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

func removeCyclicRuntimeHookDependencies(dependencies map[string][]string) {
	components := runtimeHookDependencyComponents(dependencies)
	for _, component := range components {
		if len(component) == 1 &&
			!containsRuntimeHookDependency(dependencies[component[0]], component[0]) {
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

func (a *ApplicationContext) finishRefresh(
	registry *container.Registry,
	runtimeContainer *container.Container,
	manager *lifecycle.Manager,
	err error,
) {
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

func runtimeLifecycleNames(
	instances map[string]any,
	names []string,
	managed map[string]bool,
) map[string]struct{} {
	lifecycleNames := make(map[string]struct{}, len(instances))
	for _, name := range names {
		instance, exists := instances[name]
		if exists && managed[name] && isLifecycleTarget(instance) {
			lifecycleNames[name] = struct{}{}
		}
	}
	return lifecycleNames
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

func runtimeLifecycleManagement(definitions []container.Definition) map[string]bool {
	managed := make(map[string]bool, len(definitions))
	for _, definition := range definitions {
		managed[definition.Name] = definition.LifecycleManaged
	}
	return managed
}
