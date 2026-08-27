package container

import (
	"reflect"
	"sort"
	"strings"

	arkerrors "goark.dev/goark/errors"
)

func (c *Container) selectByType(typ reflect.Type, options resolveOptions) (string, error) {
	if options.qualifier != "" {
		return c.selectQualifier(typ, options.qualifier)
	}
	names := c.matchingNames(typ)
	switch len(names) {
	case 0:
		return "", arkerrors.Newf(arkerrors.CodeNotFound, "bean type %s not found", typ)
	case 1:
		return names[0], nil
	default:
		return c.selectPreferred(typ, names)
	}
}

func (c *Container) selectQualifier(typ reflect.Type, qualifier string) (string, error) {
	definition, ok := c.definitions[qualifier]
	if !ok {
		return "", arkerrors.Newf(arkerrors.CodeNotFound, "bean %q not found", qualifier)
	}
	if !typeAssignable(definition.Type, typ) {
		return "", arkerrors.Newf(arkerrors.CodeTypeMismatch, "bean %q type %s is not assignable to %s", qualifier, definition.Type, typ)
	}
	return qualifier, nil
}

func (c *Container) selectPreferred(typ reflect.Type, names []string) (string, error) {
	primary := make([]string, 0, len(names))
	for _, name := range names {
		if c.definitions[name].Primary {
			primary = append(primary, name)
		}
	}
	if len(primary) == 1 {
		return primary[0], nil
	}
	if len(primary) > 1 {
		return "", arkerrors.Newf(arkerrors.CodeConflict, "bean type %s has multiple primary candidates: %s", typ, strings.Join(primary, ", "))
	}
	if name, ok, err := c.selectPriority(typ, names); ok || err != nil {
		return name, err
	}
	return "", arkerrors.Newf(arkerrors.CodeConflict, "bean type %s has multiple candidates: %s", typ, strings.Join(names, ", "))
}

func (c *Container) selectPriority(typ reflect.Type, names []string) (string, bool, error) {
	bestNames := make([]string, 0, len(names))
	var bestValue int
	hasPriority := false
	for _, name := range names {
		value, ok := c.definitions[name].Priority.Value()
		if !ok {
			continue
		}
		if !hasPriority || value < bestValue {
			bestValue = value
			bestNames = bestNames[:0]
			bestNames = append(bestNames, name)
			hasPriority = true
			continue
		}
		if value == bestValue {
			bestNames = append(bestNames, name)
		}
	}
	if !hasPriority {
		return "", false, nil
	}
	if len(bestNames) == 1 {
		return bestNames[0], true, nil
	}
	return "", true, arkerrors.Newf(arkerrors.CodeConflict, "bean type %s has multiple priority candidates with priority %d: %s", typ, bestValue, strings.Join(bestNames, ", "))
}

func (c *Container) matchingNames(typ reflect.Type) []string {
	if names, ok := c.indexedNames(typ); ok {
		return names
	}
	names := c.scanMatchingNames(typ)
	return c.cacheTypeIndex(typ, names)
}

func (c *Container) matchingNamesInStartupOrder(typ reflect.Type) []string {
	names := c.matchingNames(typ)
	if len(names) < 2 || len(c.singletonOrder) == 0 {
		return names
	}
	seen := make(map[string]struct{}, len(names))
	for _, name := range names {
		seen[name] = struct{}{}
	}
	ordered := make([]string, 0, len(names))
	for _, name := range c.singletonOrder {
		if _, ok := seen[name]; !ok {
			continue
		}
		ordered = append(ordered, name)
		delete(seen, name)
	}
	if len(seen) == 0 {
		return ordered
	}
	remaining := make([]string, 0, len(seen))
	for name := range seen {
		remaining = append(remaining, name)
	}
	sort.Strings(remaining)
	return append(ordered, remaining...)
}

func (c *Container) rebuildTypeIndex(definitions []Definition) {
	indexTypes := definitionTypes(definitions)
	index := make(map[reflect.Type][]string, len(indexTypes))
	for _, typ := range indexTypes {
		index[typ] = matchingDefinitionNames(definitions, typ)
	}
	c.typeIndex = index
}

func definitionTypes(definitions []Definition) []reflect.Type {
	seen := make(map[reflect.Type]struct{}, len(definitions))
	types := make([]reflect.Type, 0, len(definitions))
	for _, definition := range definitions {
		if definition.Type == nil {
			continue
		}
		if _, ok := seen[definition.Type]; ok {
			continue
		}
		seen[definition.Type] = struct{}{}
		types = append(types, definition.Type)
	}
	sort.Slice(types, func(i, j int) bool {
		return types[i].String() < types[j].String()
	})
	return types
}

func matchingDefinitionNames(definitions []Definition, typ reflect.Type) []string {
	names := make([]string, 0)
	for _, definition := range definitions {
		if typeAssignable(definition.Type, typ) {
			names = append(names, definition.Name)
		}
	}
	sort.Strings(names)
	return names
}

func (c *Container) indexedNames(typ reflect.Type) ([]string, bool) {
	c.typeIndexMu.RLock()
	defer c.typeIndexMu.RUnlock()
	names, ok := c.typeIndex[typ]
	if !ok {
		return nil, false
	}
	return append([]string(nil), names...), true
}

func (c *Container) cacheTypeIndex(typ reflect.Type, names []string) []string {
	c.typeIndexMu.Lock()
	defer c.typeIndexMu.Unlock()
	if existing, ok := c.typeIndex[typ]; ok {
		return append([]string(nil), existing...)
	}
	copied := append([]string(nil), names...)
	c.typeIndex[typ] = copied
	return append([]string(nil), copied...)
}

func (c *Container) scanMatchingNames(typ reflect.Type) []string {
	names := make([]string, 0)
	for name, definition := range c.definitions {
		if typeAssignable(definition.Type, typ) {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names
}
