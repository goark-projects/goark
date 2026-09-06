package mvc

import (
	"net/http"
	"net/url"
	"reflect"
	"strings"

	mvcflash "goark.dev/goark/web/mvc/flash"

	arkweb "goark.dev/arkarta/web"
	goweb "goark.dev/goark/web"

	"goark.dev/arkarta/servlet"
	"goark.dev/goark/web/mvc/sessionattrs"
	"goark.dev/goark/web/uri"
)

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

func (r forwardResult) Write(ctx *arkweb.Context) error {
	if ctx == nil || ctx.Request() == nil || ctx.Response() == nil {
		return arkweb.ErrNilContext
	}
	target := cleanForwardLocation(r.target)
	if target == "" {
		return servlet.NewHTTPError(http.StatusInternalServerError, http.StatusText(
			http.StatusInternalServerError), ErrInvalidForwardLocation)
	}
	app, ok := goweb.CurrentWebApp(ctx)
	if !ok {
		return servlet.NewHTTPError(http.StatusInternalServerError, http.StatusText(
			http.StatusInternalServerError), ErrForwardDispatcherUnavailable)
	}
	dispatcher, err := app.RequestDispatcher(target)
	if err != nil {
		return err
	}
	return dispatcher.Forward(ctx.Context(), ctx.Request(), ctx.Response())
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

func redirectStatus(ctx *arkweb.Context, statusCode int) (int, bool) {
	if isRedirectStatus(statusCode) {
		return statusCode, true
	}
	if statusCode != 0 {
		return 0, false
	}
	status, ok := responseStatusFromContext(ctx)
	if !ok || !isRedirectStatus(status) {
		return 0, false
	}
	return status, true
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

func isRedirectStatus(statusCode int) bool {
	switch statusCode {
	case http.StatusMovedPermanently,
		http.StatusFound,
		http.StatusSeeOther,
		http.StatusTemporaryRedirect,
		http.StatusPermanentRedirect:
		return true
	default:
		return false
	}
}

func newViewControllerOptions(options []ViewControllerOption) viewControllerOptions {
	var config viewControllerOptions
	for _, option := range options {
		if option != nil {
			option(&config)
		}
	}
	return config
}

func flashAttributeValue(ctx *arkweb.Context, name string) (string, bool, error) {
	value, ok, err := rawFlashAttributeValue(ctx, name)
	if err != nil || !ok {
		return "", ok, err
	}
	return attributeString(value), true, nil
}

func forwardResultFromViewName(viewName string) (arkweb.Result, bool) {
	target, ok := forwardLocationFromViewName(viewName)
	if !ok {
		return nil, false
	}
	return forwardResult{target: target}, true
}

func cleanForwardLocation(location string) string {
	location = strings.TrimSpace(location)
	if location == "" || strings.ContainsAny(location, "\r\n") || !strings.HasPrefix(location, "/") {
		return ""
	}
	return location
}

// CurrentSessionStatus 返回当前请求的 SessionStatus，不存在时按需创建。
func CurrentSessionStatus(ctx *arkweb.Context) *SessionStatus {
	if ctx == nil || ctx.Request() == nil {
		return &SessionStatus{}
	}
	return sessionStatusForRequest(ctx.Request())
}

func redirectResultFromViewName(ctx *arkweb.Context, statusCode int, viewName string) (
	arkweb.Result, bool) {
	result, _, ok, _ := redirectResultAndLocationFromViewNameWithModel(ctx, statusCode, viewName,
		NewModel())
	return result, ok
}

func markSessionAttributesComplete(ctx *arkweb.Context, names []string) {
	if ctx == nil || ctx.Request() == nil {
		return
	}
	sessionattrs.Complete(ctx.Request(), names)
}

// WithViewControllerStatus 设置简单视图控制器响应状态码。
func WithViewControllerStatus(statusCode int) ViewControllerOption {
	return func(options *viewControllerOptions) {
		options.status = normalizeResponseStatus(statusCode, 0)
	}
}

func prefixedViewControllerName(prefix string, value string) string {
	value = strings.TrimSpace(value)
	if strings.HasPrefix(value, prefix) {
		return value
	}
	return prefix + value
}

// AddAttribute 添加一个重定向属性。
func (a RedirectAttributes) AddAttribute(name string, value any) RedirectAttributes {
	a.model = a.Model().AddAttribute(name, value)
	return a
}

// AddAllAttributes 添加多个重定向属性。
func (a RedirectAttributes) AddAllAttributes(attributes map[string]any) RedirectAttributes {
	a.model = a.Model().AddAllAttributes(attributes)
	return a
}

// AddFlashAttributeValue 添加 Flash 属性，并按值类型推导属性名。
func (a RedirectAttributes) AddFlashAttributeValue(value any) RedirectAttributes {
	a.flashModel = a.FlashModel().AddAttributeValue(value)
	return a
}

// WithSessionAttributes 设置控制器级 Session 模型属性名，对齐 Spring @SessionAttributes。
func (c Controller) WithSessionAttributes(names ...string) Controller {
	c.sessionAttrs = mergeSessionAttributeNames(c.sessionAttrs, names)
	return c
}

// InputFlashMap 返回当前请求输入 FlashMap 的副本。
func InputFlashMap(ctx *arkweb.Context) mvcflash.Map {
	return mvcflash.Input(ctx)
}

// OutputFlashMap 返回当前请求输出 FlashMap；不存在时按需创建。
func OutputFlashMap(ctx *arkweb.Context) *mvcflash.Map {
	return mvcflash.Output(ctx)
}

// NewRedirectAttributes 创建空重定向属性集合。
func NewRedirectAttributes() RedirectAttributes {
	return RedirectAttributes{model: NewModel(), flashModel: NewModel()}
}

// Values 返回重定向属性副本。
func (a RedirectAttributes) Values() map[string]any {
	return a.model.Values()
}

// FlashValues 返回 Flash 属性副本。
func (a RedirectAttributes) FlashValues() map[string]any {
	return a.flashModel.Values()
}

const (
	// AttributeSessionStatus 保存当前 MVC 请求的 SessionStatus。
	AttributeSessionStatus = "goark.web.mvc.session.status"
)

// IsComplete 返回当前控制器 SessionAttributes 是否已完成。
func (s *SessionStatus) IsComplete() bool {
	return s != nil && s.complete
}

func mergeSessionAttributeNames(existing []string, names []string) []string {
	out := append([]string(nil), existing...)
	return appendSessionAttributeNames(out, names)
}

type forwardResult struct {
	target string
}

type viewControllerOptions struct {
	status int
}
