package container

import "strings"

// DependencyKind 表示依赖边的运行时语义。
type DependencyKind string

const (
	// DependencyKindFactory 表示工厂方法或构造参数依赖，创建实例前必须解析，不能通过 early singleton 解决循环。
	DependencyKindFactory DependencyKind = "factory"
	// DependencyKindInjection 表示字段或 setter 注入依赖，可在允许循环引用时通过 early singleton 解决单例循环。
	DependencyKindInjection DependencyKind = "injection"
	// DependencyKindDependsOn 表示手工 depends-on 初始化顺序依赖，不能通过 early singleton 解决循环。
	DependencyKindDependsOn DependencyKind = "depends-on"
)

// DependencySource 表示依赖边来源。
type DependencySource string

const (
	// DependencySourceInferred 表示由生成器按注入点自动推导出的依赖。
	DependencySourceInferred DependencySource = "inferred"
	// DependencySourceManual 表示由用户通过 goark 注解或显式注册选项声明的依赖。
	DependencySourceManual DependencySource = "manual"
)

// DependencyDescriptor 描述一个 Bean 到另一个 Bean 的依赖边。
type DependencyDescriptor struct {
	Name     string
	Kind     DependencyKind
	Source   DependencySource
	Optional bool
}

func dependencyDescriptor(name string, kind DependencyKind, source DependencySource, optional bool) DependencyDescriptor {
	return DependencyDescriptor{
		Name:     strings.TrimSpace(name),
		Kind:     kind.normalized(),
		Source:   source.normalized(),
		Optional: optional,
	}
}

func (k DependencyKind) normalized() DependencyKind {
	switch k {
	case DependencyKindFactory, DependencyKindInjection, DependencyKindDependsOn:
		return k
	default:
		return DependencyKindFactory
	}
}

func (s DependencySource) normalized() DependencySource {
	switch s {
	case DependencySourceInferred, DependencySourceManual:
		return s
	default:
		return DependencySourceInferred
	}
}

func splitDependencyNames(names []string) []string {
	out := make([]string, 0, len(names))
	for _, raw := range names {
		parts := strings.Split(raw, ",")
		for _, part := range parts {
			out = append(out, strings.TrimSpace(part))
		}
	}
	return out
}

func normalizeDefinitionDependencies(descriptors []DependencyDescriptor, dependsOn []string, dependencies []string) ([]DependencyDescriptor, []string, []string) {
	normalized := normalizeDependencyDescriptors(descriptors)
	for _, name := range splitDependencyNames(dependsOn) {
		normalized = appendDependencyDescriptor(normalized, dependencyDescriptor(name, DependencyKindDependsOn, DependencySourceManual, false))
	}
	for _, name := range splitDependencyNames(dependencies) {
		if containsDependencyDescriptorName(normalized, name) {
			continue
		}
		normalized = appendDependencyDescriptor(normalized, dependencyDescriptor(name, DependencyKindFactory, DependencySourceInferred, false))
	}
	dependsOnNames := dependencyNamesByKind(normalized, DependencyKindDependsOn)
	allNames := dependencyNames(normalized)
	return normalized, dependsOnNames, allNames
}

func normalizeDependencyDescriptors(descriptors []DependencyDescriptor) []DependencyDescriptor {
	normalized := make([]DependencyDescriptor, 0, len(descriptors))
	for _, descriptor := range descriptors {
		descriptor.Name = strings.TrimSpace(descriptor.Name)
		descriptor.Kind = descriptor.Kind.normalized()
		descriptor.Source = descriptor.Source.normalized()
		normalized = appendDependencyDescriptor(normalized, descriptor)
	}
	return normalized
}

func appendDependencyDescriptor(descriptors []DependencyDescriptor, descriptor DependencyDescriptor) []DependencyDescriptor {
	for i, existing := range descriptors {
		if existing.Name == descriptor.Name && existing.Kind == descriptor.Kind {
			if !descriptor.Optional {
				descriptors[i].Optional = false
			}
			if existing.Source != DependencySourceManual && descriptor.Source == DependencySourceManual {
				descriptors[i].Source = DependencySourceManual
			}
			return descriptors
		}
	}
	return append(descriptors, descriptor)
}

func containsDependencyDescriptorName(descriptors []DependencyDescriptor, name string) bool {
	name = strings.TrimSpace(name)
	for _, descriptor := range descriptors {
		if descriptor.Name == name {
			return true
		}
	}
	return false
}

func dependencyNamesByKind(descriptors []DependencyDescriptor, kind DependencyKind) []string {
	names := make([]string, 0, len(descriptors))
	for _, descriptor := range descriptors {
		if descriptor.Kind == kind && !containsDependencyName(names, descriptor.Name) {
			names = append(names, descriptor.Name)
		}
	}
	return names
}

func dependencyNames(descriptors []DependencyDescriptor) []string {
	names := make([]string, 0, len(descriptors))
	for _, descriptor := range descriptors {
		if !containsDependencyName(names, descriptor.Name) {
			names = append(names, descriptor.Name)
		}
	}
	return names
}
