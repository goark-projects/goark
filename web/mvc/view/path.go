package view

import (
	"path"
	"strings"
)

func cleanTemplateName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" || strings.ContainsAny(name, "\x00\\") {
		return "", ErrInvalidViewName
	}
	if strings.HasPrefix(name, "/") {
		return "", ErrInvalidViewName
	}
	for _, segment := range strings.Split(name, "/") {
		if segment == "" || segment == "." || segment == ".." {
			return "", ErrInvalidViewName
		}
	}
	cleaned := path.Clean(name)
	if cleaned == "." || strings.HasPrefix(cleaned, "../") || cleaned == ".." {
		return "", ErrInvalidViewName
	}
	return cleaned, nil
}

func cleanTemplatePrefix(prefix string) (string, error) {
	prefix = strings.TrimSpace(prefix)
	if prefix == "" {
		return "", nil
	}
	if strings.HasPrefix(prefix, "/") || strings.ContainsAny(prefix, "\x00\\") {
		return "", ErrInvalidViewName
	}
	cleaned := path.Clean(prefix)
	if cleaned == "." {
		return "", nil
	}
	for _, segment := range strings.Split(cleaned, "/") {
		if segment == ".." {
			return "", ErrInvalidViewName
		}
	}
	return strings.TrimSuffix(cleaned, "/"), nil
}
