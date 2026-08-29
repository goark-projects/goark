package uri

import (
	"fmt"
	"net/url"
	"strings"
)

func joinPath(base string, addition string) string {
	if addition == "" {
		return normalizePath(base)
	}
	base = normalizePath(base)
	addition = normalizePath(addition)
	if base == "" || base == "/" {
		return addition
	}
	return strings.TrimRight(base, "/") + "/" + strings.TrimLeft(addition, "/")
}

func normalizePath(value string) string {
	if value == "" {
		return ""
	}
	if !strings.HasPrefix(value, "/") {
		return "/" + value
	}
	return value
}

func expandPath(path string, variables map[string]string) (string, string, error) {
	if path == "" {
		return "", "", nil
	}
	if variables == nil || !strings.Contains(path, "{") {
		return path, "", nil
	}
	var decoded strings.Builder
	var raw strings.Builder
	for {
		start := strings.IndexByte(path, '{')
		if start < 0 {
			writeLiteralPath(&decoded, &raw, path)
			return decoded.String(), raw.String(), nil
		}
		writeLiteralPath(&decoded, &raw, path[:start])
		end := strings.IndexByte(path[start+1:], '}')
		if end < 0 {
			return "", "", ErrInvalidURI
		}
		name := strings.TrimSpace(path[start+1 : start+1+end])
		if name == "" {
			return "", "", ErrInvalidURI
		}
		value, ok := variables[name]
		if !ok {
			return "", "", fmt.Errorf("%w: %s", ErrMissingPathVariable, name)
		}
		decoded.WriteString(value)
		raw.WriteString(url.PathEscape(value))
		path = path[start+2+end:]
	}
}

func writeLiteralPath(decoded *strings.Builder, raw *strings.Builder, value string) {
	decoded.WriteString(value)
	raw.WriteString(escapePathLiteral(value))
}

func escapePathLiteral(value string) string {
	if value == "" {
		return ""
	}
	parts := strings.Split(value, "/")
	for i, part := range parts {
		parts[i] = url.PathEscape(part)
	}
	return strings.Join(parts, "/")
}
