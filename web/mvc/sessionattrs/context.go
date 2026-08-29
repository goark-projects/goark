package sessionattrs

import (
	"strings"

	"goark.dev/arkarta/servlet"
)

const (
	// AttributePending 保存当前请求待写入 Session 的 MVC 模型属性。
	AttributePending = "goark.web.mvc.sessionattrs.pending"
	// AttributeComplete 保存当前请求需要清理的 MVC SessionAttributes 名称。
	AttributeComplete = "goark.web.mvc.sessionattrs.complete"
)

// Save 记录请求结束后需要写入 Session 的模型属性。
func Save(req *servlet.Request, values map[string]any) {
	if req == nil || len(values) == 0 {
		return
	}
	req.SetAttribute(AttributePending, cleanAttributes(values))
}

// Complete 记录请求结束后需要从 Session 清理的模型属性名。
func Complete(req *servlet.Request, names []string) {
	if req == nil || len(names) == 0 {
		return
	}
	req.SetAttribute(AttributeComplete, cleanNames(names))
}

func pending(req *servlet.Request) map[string]any {
	if req == nil {
		return nil
	}
	value, ok := req.Attribute(AttributePending)
	if !ok {
		return nil
	}
	attributes, ok := value.(map[string]any)
	if !ok || len(attributes) == 0 {
		return nil
	}
	return cleanAttributes(attributes)
}

func completed(req *servlet.Request) []string {
	if req == nil {
		return nil
	}
	value, ok := req.Attribute(AttributeComplete)
	if !ok {
		return nil
	}
	names, ok := value.([]string)
	if !ok || len(names) == 0 {
		return nil
	}
	return cleanNames(names)
}

func cleanAttributes(values map[string]any) map[string]any {
	if len(values) == 0 {
		return nil
	}
	out := make(map[string]any, len(values))
	for name, value := range values {
		name = strings.TrimSpace(name)
		if name != "" {
			out[name] = value
		}
	}
	return out
}

func cleanNames(names []string) []string {
	if len(names) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(names))
	out := make([]string, 0, len(names))
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		if _, exists := seen[name]; exists {
			continue
		}
		seen[name] = struct{}{}
		out = append(out, name)
	}
	return out
}
