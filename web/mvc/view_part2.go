package mvc

import (
	"reflect"
	"strings"

	mvcflash "goark.dev/goark/web/mvc/flash"

	arkweb "goark.dev/arkarta/web"
	goweb "goark.dev/goark/web"

	"goark.dev/arkarta/servlet/session"
	"goark.dev/goark/web/mvc/sessionattrs"
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

func flashModelFrom(model any) Model {
	switch value := model.(type) {
	case RedirectAttributes:
		return value.FlashModel()
	case *RedirectAttributes:
		if value == nil {
			return NewModel()
		}
		return value.FlashModel()
	case *mvcflash.Map:
		if value == nil {
			return NewModel()
		}
		return NewModel().AddAllAttributes(value.Values())
	case mvcflash.Map:
		return NewModel().AddAllAttributes((&value).Values())
	default:
		return NewModel()
	}
}

func redirectResultAndLocationFromViewNameWithModel(
	ctx *arkweb.Context,
	statusCode int,
	viewName string,
	model Model,
) (arkweb.Result, string, bool, error) {
	location, ok := redirectLocationFromViewName(viewName)
	if !ok {
		return nil, "", false, nil
	}
	location, err := redirectLocationWithModel(location, model)
	if err != nil {
		return nil, "", true, err
	}
	if status, ok := redirectStatus(ctx, statusCode); ok {
		return goweb.Redirect(location, goweb.WithRedirectStatus(status)), location, true, nil
	}
	return goweb.Redirect(location), location, true, nil
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

func saveRedirectFlash(ctx *arkweb.Context, location string, model Model) error {
	if model.Len() == 0 {
		return nil
	}
	output := mvcflash.Output(ctx)
	if output == nil {
		return arkweb.ErrNilContext
	}
	output.AddAllAttributes(model.Values())
	output.SetTargetLocation(location)
	return nil
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

func rawFlashAttributeValue(ctx *arkweb.Context, name string) (any, bool, error) {
	if ctx == nil {
		return nil, false, arkweb.ErrNilContext
	}
	input := mvcflash.Input(ctx)
	value, ok := (&input).Attribute(name)
	return value, ok, nil
}

func isRedirectMultiValue(value reflect.Value) bool {
	if !value.IsValid() {
		return false
	}
	kind := value.Kind()
	return (kind == reflect.Slice || kind == reflect.Array) && value.Type().Elem().Kind() !=
		reflect.Uint8
}

func forwardLocationFromViewName(viewName string) (string, bool) {
	viewName = strings.TrimSpace(viewName)
	if !strings.HasPrefix(viewName, ForwardViewNamePrefix) {
		return "", false
	}
	return strings.TrimSpace(strings.TrimPrefix(viewName, ForwardViewNamePrefix)), true
}

func redirectLocationFromViewName(viewName string) (string, bool) {
	viewName = strings.TrimSpace(viewName)
	if !strings.HasPrefix(viewName, RedirectViewNamePrefix) {
		return "", false
	}
	return strings.TrimSpace(strings.TrimPrefix(viewName, RedirectViewNamePrefix)), true
}

// FlashAttribute 读取一次性 Flash 属性，并转换为目标类型。
func FlashAttribute[T any](ctx *arkweb.Context, name string, options ...ParamOption) (T, error) {
	value, ok, err := rawFlashAttributeValue(ctx, name)
	paramOptions := newParamOptions(ctx, options)
	return resolveAttributeValue[T]("Flash属性", name, value, ok, err, paramOptions)
}

// Redirect 创建 redirect: ModelAndView，并使用属性展开路径变量和查询参数。
func Redirect(location string, attributes RedirectAttributes,
	options ...ModelAndViewOption) ModelAndView {
	return NewModelAndView(prefixedViewControllerName(RedirectViewNamePrefix, location),
		attributes, options...)
}

// SetComplete 标记当前控制器 SessionAttributes 已完成，请求结束后清理对应 Session 属性。
func (s *SessionStatus) SetComplete() {
	if s != nil {
		s.complete = true
	}
}

// RedirectViewController 创建只执行重定向的 GET 路由。
func RedirectViewController(pattern string, location string,
	options ...ViewControllerOption) Route {
	return viewControllerRoute(pattern, prefixedViewControllerName(RedirectViewNamePrefix,
		location), options...)
}

func viewControllerRoute(pattern string, viewName string, options ...ViewControllerOption) Route {
	config := newViewControllerOptions(options)
	return GET(pattern, arkweb.HandlerFunc(func(*arkweb.Context) (arkweb.Result, error) {
		return NewModelAndView(viewName, nil, WithViewStatus(config.status)), nil
	}))
}

// RedirectAttributes 表示重定向 URI 属性和一次性 Flash 属性。
type RedirectAttributes struct {
	model      Model
	flashModel Model
}

// AddAttributeValue 添加重定向属性，并按值类型推导属性名。
func (a RedirectAttributes) AddAttributeValue(value any) RedirectAttributes {
	a.model = a.Model().AddAttributeValue(value)
	return a
}

// AddFlashAttribute 添加一次性 Flash 属性。
func (a RedirectAttributes) AddFlashAttribute(name string, value any) RedirectAttributes {
	a.flashModel = a.FlashModel().AddAttribute(name, value)
	return a
}

// AddAllFlashAttributes 添加多个 Flash 属性。
func (a RedirectAttributes) AddAllFlashAttributes(attributes map[string]any) RedirectAttributes {
	a.flashModel = a.FlashModel().AddAllAttributes(attributes)
	return a
}

// ForwardViewController 创建只执行服务端转发的 GET 路由。
func ForwardViewController(pattern string, target string, options ...ViewControllerOption) Route {
	return viewControllerRoute(pattern, prefixedViewControllerName(ForwardViewNamePrefix, target),
		options...)
}

const (
	// ForwardViewNamePrefix 表示 Spring 风格服务端转发视图名前缀。
	ForwardViewNamePrefix = "forward:"
)

// Model 返回可用于 ModelAndView 的模型副本。
func (a RedirectAttributes) Model() Model {
	return NewModel().AddAllAttributes(a.model.Values())
}

// FlashModel 返回 Flash 属性模型副本。
func (a RedirectAttributes) FlashModel() Model {
	return NewModel().AddAllAttributes(a.flashModel.Values())
}

const (
	// RedirectViewNamePrefix 表示 Spring 风格重定向视图名前缀。
	RedirectViewNamePrefix = "redirect:"
)

// SessionStatus 表示控制器级 SessionAttributes 的完成状态。
type SessionStatus struct {
	complete bool
}

// SetSessionComplete 标记当前控制器 SessionAttributes 已完成。
func SetSessionComplete(ctx *arkweb.Context) {
	CurrentSessionStatus(ctx).SetComplete()
}

// ViewController 创建只渲染固定逻辑视图名的 GET 路由。
func ViewController(pattern string, viewName string, options ...ViewControllerOption) Route {
	return viewControllerRoute(pattern, strings.TrimSpace(viewName), options...)
}

func normalizeSessionAttributeNames(names []string) []string {
	return appendSessionAttributeNames(nil, names)
}

// FlashMap 是 Spring FlashMap 的 Go 化一次性属性集合。
type FlashMap = mvcflash.Map

// ViewControllerOption 定制简单视图控制器。
type ViewControllerOption func(*viewControllerOptions)
