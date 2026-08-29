package mvc

import (
	"strings"

	"goark.dev/arkarta/servlet"
	"goark.dev/arkarta/servlet/session"
	arkweb "goark.dev/arkarta/web"
	"goark.dev/goark/web/mvc/sessionattrs"
)

const (
	// AttributeSessionStatus 保存当前 MVC 请求的 SessionStatus。
	AttributeSessionStatus = "goark.web.mvc.session.status"
)

// SessionStatus 表示控制器级 SessionAttributes 的完成状态。
type SessionStatus struct {
	complete bool
}

// SetComplete 标记当前控制器 SessionAttributes 已完成，请求结束后清理对应 Session 属性。
func (s *SessionStatus) SetComplete() {
	if s != nil {
		s.complete = true
	}
}

// IsComplete 返回当前控制器 SessionAttributes 是否已完成。
func (s *SessionStatus) IsComplete() bool {
	return s != nil && s.complete
}

// CurrentSessionStatus 返回当前请求的 SessionStatus，不存在时按需创建。
func CurrentSessionStatus(ctx *arkweb.Context) *SessionStatus {
	if ctx == nil || ctx.Request() == nil {
		return &SessionStatus{}
	}
	return sessionStatusForRequest(ctx.Request())
}

// SetSessionComplete 标记当前控制器 SessionAttributes 已完成。
func SetSessionComplete(ctx *arkweb.Context) {
	CurrentSessionStatus(ctx).SetComplete()
}

// WithSessionAttributes 设置控制器级 Session 模型属性名，对齐 Spring @SessionAttributes。
func (c Controller) WithSessionAttributes(names ...string) Controller {
	c.sessionAttrs = mergeSessionAttributeNames(c.sessionAttrs, names)
	return c
}

func wrapSessionAttributes(handler arkweb.Handler, names []string) arkweb.Handler {
	names = normalizeSessionAttributeNames(names)
	if handler == nil || len(names) == 0 {
		return handler
	}
	return arkweb.HandlerFunc(func(ctx *arkweb.Context) (arkweb.Result, error) {
		loadSessionAttributes(ctx, names)
		status := CurrentSessionStatus(ctx)
		result, err := handler.Handle(ctx)
		if err != nil {
			return result, err
		}
		if status.IsComplete() {
			markSessionAttributesComplete(ctx, names)
			return result, nil
		}
		saveSessionAttributes(ctx, names)
		return result, nil
	})
}

func loadSessionAttributes(ctx *arkweb.Context, names []string) {
	if ctx == nil || ctx.Request() == nil {
		return
	}
	current, ok := session.Current(ctx.Request())
	if !ok {
		return
	}
	model := CurrentModel(ctx)
	for _, name := range names {
		if value, exists := current.Attribute(name); exists {
			model = model.AddAttribute(name, value)
		}
	}
	setCurrentModel(ctx, model)
}

func saveSessionAttributes(ctx *arkweb.Context, names []string) {
	if ctx == nil || ctx.Request() == nil {
		return
	}
	model, ok := currentModel(ctx)
	if !ok {
		return
	}
	values := make(map[string]any, len(names))
	for _, name := range names {
		if value, exists := model.Attribute(name); exists {
			values[name] = value
		}
	}
	sessionattrs.Save(ctx.Request(), values)
}

func markSessionAttributesComplete(ctx *arkweb.Context, names []string) {
	if ctx == nil || ctx.Request() == nil {
		return
	}
	sessionattrs.Complete(ctx.Request(), names)
}

func sessionStatusForRequest(req *servlet.Request) *SessionStatus {
	if req == nil {
		return &SessionStatus{}
	}
	if value, ok := req.Attribute(AttributeSessionStatus); ok {
		if status, ok := value.(*SessionStatus); ok && status != nil {
			return status
		}
	}
	status := &SessionStatus{}
	req.SetAttribute(AttributeSessionStatus, status)
	return status
}

func mergeSessionAttributeNames(existing []string, names []string) []string {
	out := append([]string(nil), existing...)
	return appendSessionAttributeNames(out, names)
}

func normalizeSessionAttributeNames(names []string) []string {
	return appendSessionAttributeNames(nil, names)
}

func appendSessionAttributeNames(out []string, names []string) []string {
	seen := make(map[string]struct{}, len(out)+len(names))
	cleaned := out[:0]
	for _, name := range out {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		if _, exists := seen[name]; exists {
			continue
		}
		seen[name] = struct{}{}
		cleaned = append(cleaned, name)
	}
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		if _, exists := seen[name]; exists {
			continue
		}
		seen[name] = struct{}{}
		cleaned = append(cleaned, name)
	}
	return cleaned
}
