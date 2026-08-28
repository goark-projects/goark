package container

import (
	"fmt"
	"sort"
	"strings"

	arkerrors "goark.dev/goark/errors"
)

type dependencyEdge struct {
	From       string
	To         string
	Descriptor DependencyDescriptor
}

func (c *Container) buildDependencyGraph() (map[string][]dependencyEdge, error) {
	graph := make(map[string][]dependencyEdge, len(c.definitions))
	for name := range c.definitions {
		graph[name] = nil
	}
	for name, definition := range c.definitions {
		normalized := definition.normalized()
		for _, dependency := range normalized.DependencyDescriptors {
			if _, exists := c.definitions[dependency.Name]; !exists {
				if dependency.Optional {
					continue
				}
				return nil, arkerrors.Newf(arkerrors.CodeNotFound, "bean %q depends on missing bean %q", name, dependency.Name)
			}
			graph[name] = append(graph[name], dependencyEdge{
				From:       name,
				To:         dependency.Name,
				Descriptor: dependency,
			})
		}
		sortDependencyEdges(graph[name])
	}
	return graph, nil
}

func sortDependencyEdges(edges []dependencyEdge) {
	sort.Slice(edges, func(i, j int) bool {
		if edges[i].To != edges[j].To {
			return edges[i].To < edges[j].To
		}
		if edges[i].Descriptor.Kind != edges[j].Descriptor.Kind {
			return edges[i].Descriptor.Kind < edges[j].Descriptor.Kind
		}
		return edges[i].Descriptor.Source < edges[j].Descriptor.Source
	})
}

func (c *Container) validateDependencyGraph(graph map[string][]dependencyEdge) error {
	components := dependencyComponents(graph)
	for _, component := range components {
		edges := internalDependencyEdges(graph, component)
		if len(component) == 1 && !hasSelfDependency(edges, component[0]) {
			continue
		}
		if c.allowCircularReferences && c.canResolveCircularComponent(component, edges) {
			continue
		}
		return arkerrors.Newf(arkerrors.CodeCircularDependency, "circular dependency detected: %s", dependencyCycleDescription(component, edges))
	}
	return nil
}

func (c *Container) canResolveCircularComponent(component []string, edges []dependencyEdge) bool {
	for _, name := range component {
		definition := c.definitions[name].normalized()
		if definition.Scope != ScopeSingleton || definition.DependencyInjector == nil {
			return false
		}
	}
	for _, edge := range edges {
		if edge.Descriptor.Kind != DependencyKindInjection {
			return false
		}
	}
	return len(edges) > 0
}

func hasSelfDependency(edges []dependencyEdge, name string) bool {
	for _, edge := range edges {
		if edge.From == name && edge.To == name {
			return true
		}
	}
	return false
}

func internalDependencyEdges(graph map[string][]dependencyEdge, component []string) []dependencyEdge {
	members := make(map[string]struct{}, len(component))
	for _, name := range component {
		members[name] = struct{}{}
	}
	edges := make([]dependencyEdge, 0)
	for _, from := range component {
		for _, edge := range graph[from] {
			if _, ok := members[edge.To]; ok {
				edges = append(edges, edge)
			}
		}
	}
	sortDependencyEdges(edges)
	return edges
}

func dependencyCycleDescription(component []string, edges []dependencyEdge) string {
	if len(edges) == 0 {
		names := append([]string(nil), component...)
		sort.Strings(names)
		return strings.Join(names, " -> ")
	}
	parts := make([]string, 0, len(edges))
	for _, edge := range edges {
		parts = append(parts, fmt.Sprintf("%s -> %s (%s,%s)", edge.From, edge.To, edge.Descriptor.Kind, edge.Descriptor.Source))
	}
	return strings.Join(parts, "; ")
}

func dependencyComponents(graph map[string][]dependencyEdge) [][]string {
	names := make([]string, 0, len(graph))
	for name := range graph {
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

		for _, edge := range graph[name] {
			next := edge.To
			if indices[next] == -1 {
				strongConnect(next)
				if lowlink[next] < lowlink[name] {
					lowlink[name] = lowlink[next]
				}
				continue
			}
			if onStack[next] && indices[next] < lowlink[name] {
				lowlink[name] = indices[next]
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

func (c *Container) computeStartupOrder(graph map[string][]dependencyEdge) []string {
	components := dependencyComponents(graph)
	componentIndex := make(map[string]int, len(c.definitions))
	for index, component := range components {
		for _, name := range component {
			componentIndex[name] = index
		}
	}

	dependents := make(map[int][]int, len(components))
	inDegree := make([]int, len(components))
	seenEdges := make(map[[2]int]struct{})
	for from, edges := range graph {
		fromComponent := componentIndex[from]
		for _, edge := range edges {
			toComponent := componentIndex[edge.To]
			if fromComponent == toComponent {
				continue
			}
			key := [2]int{toComponent, fromComponent}
			if _, exists := seenEdges[key]; exists {
				continue
			}
			seenEdges[key] = struct{}{}
			dependents[toComponent] = append(dependents[toComponent], fromComponent)
			inDegree[fromComponent]++
		}
	}
	for component := range dependents {
		sort.Ints(dependents[component])
	}

	ready := make([]int, 0, len(components))
	for index, degree := range inDegree {
		if degree == 0 {
			ready = append(ready, index)
		}
	}
	c.sortComponents(ready, components)

	order := make([]string, 0, len(c.definitions))
	for len(ready) > 0 {
		component := ready[0]
		ready = ready[1:]
		for _, name := range components[component] {
			definition := c.definitions[name].normalized()
			if definition.Scope == ScopeSingleton {
				order = append(order, name)
			}
		}
		for _, dependent := range dependents[component] {
			inDegree[dependent]--
			if inDegree[dependent] == 0 {
				ready = append(ready, dependent)
			}
		}
		c.sortComponents(ready, components)
	}
	return order
}

func (c *Container) sortComponents(indices []int, components [][]string) {
	sort.Slice(indices, func(i, j int) bool {
		left := c.componentOrder(components[indices[i]])
		right := c.componentOrder(components[indices[j]])
		if left != right {
			return left < right
		}
		return components[indices[i]][0] < components[indices[j]][0]
	})
}

func (c *Container) componentOrder(component []string) int {
	if len(component) == 0 {
		return 0
	}
	order := c.definitions[component[0]].normalized().Order
	for _, name := range component[1:] {
		if current := c.definitions[name].normalized().Order; current < order {
			order = current
		}
	}
	return order
}
