package flash

import (
	"context"
	"net/http"
	"time"

	"goark.dev/arkarta/servlet"
	"goark.dev/arkarta/servlet/session"
)

const (
	// SessionAttributeMaps 保存等待匹配的 FlashMap 队列。
	SessionAttributeMaps = "goark.web.mvc.flash.maps"
	// DefaultTimeout 是 FlashMap 默认存活时间，对齐 Spring MVC 默认值。
	DefaultTimeout = 180 * time.Second
)

// Option 定制 Flash 过滤器。
type Option func(*Filter)

// Filter 在 Session 边界保存并消费一次性 FlashMap。
type Filter struct {
	accessor *session.Accessor
	timeout  time.Duration
	now      func() time.Time
}

// WithTimeout 设置 FlashMap 过期时间。
func WithTimeout(timeout time.Duration) Option {
	return func(filter *Filter) {
		if timeout > 0 {
			filter.timeout = timeout
		}
	}
}

// NewFilter 创建 Flash 过滤器。
func NewFilter(accessor *session.Accessor, options ...Option) (*Filter, error) {
	if accessor == nil {
		return nil, ErrNilAccessor
	}
	filter := &Filter{
		accessor: accessor,
		timeout:  DefaultTimeout,
		now:      time.Now,
	}
	for _, option := range options {
		if option != nil {
			option(filter)
		}
	}
	return filter, nil
}

// NewSessionFilter 使用 Session Manager 创建 Flash 过滤器。
func NewSessionFilter(manager session.Manager, options ...Option) (*Filter, error) {
	accessor, err := session.NewAccessor(manager)
	if err != nil {
		return nil, err
	}
	return NewFilter(accessor, options...)
}

// Filter 执行请求级 FlashMap 取出和响应级保存。
func (f *Filter) Filter(ctx context.Context, req *servlet.Request, res servlet.Response, chain servlet.Chain) error {
	if chain == nil {
		return servlet.ErrNilHandler
	}
	if f == nil || f.accessor == nil {
		return ErrNilAccessor
	}
	if err := f.restoreInput(ctx, req); err != nil {
		return err
	}
	err := chain.Next(ctx, req, res)
	if err != nil {
		return err
	}
	return f.saveOutput(ctx, req, res)
}

func (f *Filter) restoreInput(ctx context.Context, req *servlet.Request) error {
	if req == nil {
		return session.ErrNilRequest
	}
	current, ok, err := f.accessor.Get(ctx, req, nil, false)
	if err != nil || !ok {
		return err
	}
	maps, ok := sessionFlashMaps(current)
	if !ok {
		return nil
	}
	now := f.now()
	var input Map
	remaining := make([]Map, 0, len(maps))
	for _, item := range maps {
		if item.expired(now) {
			continue
		}
		if item.matches(req) {
			input.AddAllAttributes(item.Values())
			continue
		}
		remaining = append(remaining, (&item).clone())
	}
	if (&input).Len() > 0 {
		setInputMap(req, input)
	}
	return replaceSessionFlashMaps(current, remaining)
}

func (f *Filter) saveOutput(ctx context.Context, req *servlet.Request, res servlet.Response) error {
	if req == nil || res == nil || !isRedirectResponse(res) {
		return nil
	}
	output, ok := existingOutputMap(req)
	if !ok || output.Len() == 0 {
		return nil
	}
	if output.TargetPath() == "" && !output.SetTargetLocation(res.Header().Get("Location")) {
		return nil
	}
	current, _, err := f.accessor.Get(ctx, req, res, true)
	if err != nil {
		return err
	}
	now := f.now()
	entry := output.startExpiration(now, f.timeout)
	maps, _ := sessionFlashMaps(current)
	maps = append(activeFlashMaps(maps, now), entry)
	return current.SetAttribute(SessionAttributeMaps, maps)
}

func sessionFlashMaps(current session.Session) ([]Map, bool) {
	if current == nil {
		return nil, false
	}
	value, ok := current.Attribute(SessionAttributeMaps)
	if !ok {
		return nil, false
	}
	maps, ok := value.([]Map)
	if !ok {
		return nil, false
	}
	return append([]Map(nil), maps...), true
}

func replaceSessionFlashMaps(current session.Session, maps []Map) error {
	if current == nil {
		return nil
	}
	if len(maps) == 0 {
		return current.RemoveAttribute(SessionAttributeMaps)
	}
	return current.SetAttribute(SessionAttributeMaps, maps)
}

func activeFlashMaps(maps []Map, now time.Time) []Map {
	if len(maps) == 0 {
		return nil
	}
	out := make([]Map, 0, len(maps))
	for _, item := range maps {
		if !item.expired(now) {
			out = append(out, (&item).clone())
		}
	}
	return out
}

func (m Map) matches(req *servlet.Request) bool {
	if req == nil {
		return false
	}
	if m.targetPath != "" && m.targetPath != req.Path() {
		return false
	}
	if len(m.targetParams) == 0 {
		return true
	}
	requestValues := req.Query()
	for name, expected := range m.targetParams {
		actual := requestValues[name]
		if !containsAllValues(actual, expected) {
			return false
		}
	}
	return true
}

func containsAllValues(actual []string, expected []string) bool {
	for _, value := range expected {
		if !containsValue(actual, value) {
			return false
		}
	}
	return true
}

func containsValue(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}

func isRedirectResponse(res servlet.Response) bool {
	if res == nil {
		return false
	}
	status := res.Status()
	return status >= http.StatusMultipleChoices &&
		status < http.StatusBadRequest &&
		res.Header().Get("Location") != ""
}

var _ servlet.Filter = (*Filter)(nil)
