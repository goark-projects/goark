package container

import (
	"reflect"
	"sort"
	"strings"

	arkerrors "github.com/goark-projects/goark/errors"
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
	seen := make(map[string]struct{})
	if names, ok := c.typeIndex[typ]; ok {
		for _, name := range names {
			seen[name] = struct{}{}
		}
	}
	for name, definition := range c.definitions {
		if typeAssignable(definition.Type, typ) {
			seen[name] = struct{}{}
		}
	}
	names := make([]string, 0, len(seen))
	for name := range seen {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
