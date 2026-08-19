package container

import (
	"reflect"
	"sort"
	"strings"

	arkerrors "github.com/goark-projects/goark/errors"
)

func (c *Container) selectByType(typ reflect.Type) (string, error) {
	names := c.matchingNames(typ)
	switch len(names) {
	case 0:
		return "", arkerrors.Newf(arkerrors.CodeNotFound, "bean type %s not found", typ)
	case 1:
		return names[0], nil
	default:
		return c.selectPrimary(typ, names)
	}
}

func (c *Container) selectPrimary(typ reflect.Type, names []string) (string, error) {
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
	return "", arkerrors.Newf(arkerrors.CodeConflict, "bean type %s has multiple candidates: %s", typ, strings.Join(names, ", "))
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
