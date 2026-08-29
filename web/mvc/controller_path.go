package mvc

import "strings"

// WithPathPrefixes 设置控制器级路径前缀，对齐 Spring 类型级 @RequestMapping path。
func (c Controller) WithPathPrefixes(prefixes ...string) Controller {
	c.pathPrefixes = cleanControllerPathPrefixes(prefixes)
	return c
}

// PathPrefixes 返回控制器级路径前缀快照。
func (c Controller) PathPrefixes() []string {
	out := make([]string, 0, len(c.pathPrefixes))
	for _, prefix := range c.pathPrefixes {
		if prefix == "" {
			out = append(out, "/")
			continue
		}
		out = append(out, prefix)
	}
	return out
}

func (c Controller) registrationPathPrefixes() []string {
	if len(c.pathPrefixes) == 0 {
		return []string{""}
	}
	return append([]string(nil), c.pathPrefixes...)
}

func cleanControllerPathPrefixes(prefixes []string) []string {
	out := make([]string, 0, len(prefixes))
	seen := make(map[string]struct{}, len(prefixes))
	for _, value := range prefixes {
		for _, item := range strings.Split(value, ",") {
			prefix := normalizeControllerPathPrefix(item)
			if _, exists := seen[prefix]; exists {
				continue
			}
			seen[prefix] = struct{}{}
			out = append(out, prefix)
		}
	}
	return out
}

func normalizeControllerPathPrefix(prefix string) string {
	prefix = strings.TrimSpace(strings.ReplaceAll(prefix, "\\", "/"))
	if prefix == "" || prefix == "/" {
		return ""
	}
	if !strings.HasPrefix(prefix, "/") {
		prefix = "/" + prefix
	}
	prefix = strings.TrimRight(prefix, "/")
	if prefix == "" {
		return ""
	}
	return prefix
}

func joinControllerRoutePattern(prefix string, pattern string) string {
	pattern = strings.TrimSpace(pattern)
	if pattern == "" {
		pattern = "/"
	}
	if !strings.HasPrefix(pattern, "/") {
		pattern = "/" + pattern
	}
	if prefix == "" {
		return pattern
	}
	if pattern == "/" {
		return prefix
	}
	return prefix + "/" + strings.TrimLeft(pattern, "/")
}
