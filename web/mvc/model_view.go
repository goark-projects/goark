package mvc

import (
	"net/http"
	"path"
	"strings"

	arkweb "goark.dev/arkarta/web"
	"goark.dev/goark/web/mvc/view"
)

const defaultViewName = "index"

// ModelAndView 表示视图名、模型和状态码的组合结果。
type ModelAndView struct {
	viewName string
	model    Model
	status   int
	resolver view.Resolver
}

// ModelAndViewOption 定制 ModelAndView。
type ModelAndViewOption func(*ModelAndView)

// NewModelAndView 创建 MVC 模型视图结果。
func NewModelAndView(viewName string, model any, options ...ModelAndViewOption) ModelAndView {
	out := ModelAndView{
		viewName: strings.TrimSpace(viewName),
		model:    normalizeModel(model),
	}
	for _, option := range options {
		if option != nil {
			option(&out)
		}
	}
	return out
}

// WithViewStatus 设置视图响应状态码。
func WithViewStatus(statusCode int) ModelAndViewOption {
	return func(out *ModelAndView) {
		out.status = normalizeResponseStatus(statusCode, 0)
	}
}

// WithViewResolver 设置模型视图使用的视图解析器。
func WithViewResolver(resolver view.Resolver) ModelAndViewOption {
	return func(out *ModelAndView) {
		out.resolver = resolver
	}
}

// ViewName 返回逻辑视图名。
func (v ModelAndView) ViewName() string {
	return v.viewName
}

// Model 返回视图模型副本。
func (v ModelAndView) Model() map[string]any {
	return v.model.Values()
}

// Write 渲染模型视图。
func (v ModelAndView) Write(ctx *arkweb.Context) error {
	viewName := v.viewName
	if viewName == "" {
		viewName = DefaultViewName(ctx)
	}
	if result, ok := forwardResultFromViewName(viewName); ok {
		return result.Write(ctx)
	}
	if result, ok := redirectResultFromViewName(ctx, v.status, viewName); ok {
		return result.Write(ctx)
	}
	return view.Using(
		v.resolver,
		viewName,
		v.model.Values(),
		view.WithStatus(resolveResponseStatus(ctx, v.status, http.StatusOK)),
	).Write(ctx)
}

// DefaultViewName 按请求路径推导默认逻辑视图名。
func DefaultViewName(ctx *arkweb.Context) string {
	if ctx == nil || ctx.Request() == nil {
		return defaultViewName
	}
	return defaultViewNameFromPath(ctx.Request().Path())
}

func implicitModelResult(ctx *arkweb.Context, statusCode int, model Model) arkweb.Result {
	return NewModelAndView(DefaultViewName(ctx), model, WithViewStatus(resolveResponseStatus(ctx, statusCode, http.StatusOK)))
}

func mergeModelAndView(ctx *arkweb.Context, value ModelAndView) ModelAndView {
	value.model = mergeCurrentModel(ctx, value.model)
	return value
}

func normalizeModel(model any) Model {
	switch value := model.(type) {
	case nil:
		return NewModel()
	case Model:
		return value
	case *Model:
		if value == nil {
			return NewModel()
		}
		return *value
	case map[string]any:
		return NewModel().AddAllAttributes(value)
	default:
		return NewModel().AddAttribute("value", value)
	}
}

func defaultViewNameFromPath(rawPath string) string {
	rawPath = strings.TrimSpace(strings.ReplaceAll(rawPath, "\\", "/"))
	if rawPath == "" || rawPath == "/" {
		return defaultViewName
	}
	segments := strings.Split(rawPath, "/")
	out := make([]string, 0, len(segments))
	for _, segment := range segments {
		segment = stripPathSemicolon(segment)
		if segment == "" || segment == "." || segment == ".." {
			continue
		}
		out = append(out, segment)
	}
	if len(out) == 0 {
		return defaultViewName
	}
	last := out[len(out)-1]
	if ext := path.Ext(last); ext != "" {
		last = strings.TrimSuffix(last, ext)
	}
	out[len(out)-1] = last
	name := path.Clean(strings.Join(out, "/"))
	if name == "." || strings.HasPrefix(name, "../") || name == ".." {
		return defaultViewName
	}
	return name
}

func stripPathSemicolon(segment string) string {
	if index := strings.IndexByte(segment, ';'); index >= 0 {
		return segment[:index]
	}
	return segment
}
