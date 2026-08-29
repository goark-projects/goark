package mvc

import (
	"net/url"
	"reflect"
	"strings"

	"goark.dev/goark/web/uri"
)

func redirectLocationWithModel(location string, model Model) (string, error) {
	location = strings.TrimSpace(location)
	if location == "" {
		return location, nil
	}
	variables, err := redirectPathVariables(location, model)
	if err != nil {
		return "", err
	}
	builder, err := uri.From(location)
	if err != nil {
		return "", err
	}
	used := make(map[string]struct{}, len(variables))
	for name := range variables {
		used[name] = struct{}{}
	}
	for name, value := range model.Values() {
		if _, ok := used[name]; ok {
			continue
		}
		if values := redirectAttributeValues(value); len(values) > 0 {
			builder = builder.QueryParam(name, values...)
		}
	}
	return builder.BuildAndExpand(variables)
}

func redirectPathVariables(location string, model Model) (map[string]string, error) {
	names, err := redirectTemplateNames(location)
	if err != nil {
		return nil, err
	}
	if len(names) == 0 {
		return nil, nil
	}
	attributes := model.Values()
	variables := make(map[string]string, len(names))
	for _, name := range names {
		values := redirectAttributeValues(attributes[name])
		if len(values) > 0 {
			variables[name] = values[0]
		}
	}
	return variables, nil
}

func redirectTemplateNames(location string) ([]string, error) {
	parsed, err := url.Parse(location)
	if err != nil {
		return nil, uri.ErrInvalidURI
	}
	path := parsed.Path
	if !strings.Contains(path, "{") {
		return nil, nil
	}
	names := make([]string, 0, 4)
	seen := make(map[string]struct{}, 4)
	for {
		start := strings.IndexByte(path, '{')
		if start < 0 {
			return names, nil
		}
		end := strings.IndexByte(path[start+1:], '}')
		if end < 0 {
			return nil, uri.ErrInvalidURI
		}
		name := strings.TrimSpace(path[start+1 : start+1+end])
		if name == "" {
			return nil, uri.ErrInvalidURI
		}
		if _, ok := seen[name]; !ok {
			seen[name] = struct{}{}
			names = append(names, name)
		}
		path = path[start+2+end:]
	}
}

func redirectAttributeValues(value any) []string {
	if value == nil {
		return nil
	}
	current := reflect.ValueOf(value)
	if isNilRedirectValue(current) {
		return nil
	}
	if isRedirectMultiValue(current) {
		values := make([]string, 0, current.Len())
		for i := 0; i < current.Len(); i++ {
			item := current.Index(i)
			if isNilRedirectValue(item) {
				continue
			}
			values = append(values, attributeString(item.Interface()))
		}
		return values
	}
	return []string{attributeString(value)}
}

func isRedirectMultiValue(value reflect.Value) bool {
	if !value.IsValid() {
		return false
	}
	kind := value.Kind()
	return (kind == reflect.Slice || kind == reflect.Array) && value.Type().Elem().Kind() != reflect.Uint8
}

func isNilRedirectValue(value reflect.Value) bool {
	if !value.IsValid() {
		return true
	}
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return value.IsNil()
	default:
		return false
	}
}
