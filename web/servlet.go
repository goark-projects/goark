package web

import (
	"net/http"
	"strings"
	"time"

	"goark.dev/arkarta/servlet"
	arkweb "goark.dev/arkarta/web"
	"goark.dev/goark/web/stream"
)

// Servlet 表示可挂载到 Goark Web 部署中的 Arkarta Servlet。
type Servlet = servlet.Servlet

type servletMapping struct {
	pattern string
	name    string
	handler servlet.Servlet
	filters []servlet.Filter
}

// AddServlet 添加底层 Servlet 映射。
func (r *Registry) AddServlet(
	pattern string,
	name string,
	handler servlet.Servlet,
	filters ...servlet.Filter,
) error {
	if r == nil {
		return ErrNilRegistry
	}
	if isNilServlet(handler) {
		return ErrNilServlet
	}
	router := servlet.NewRouter()
	if err := router.Handle(pattern, handler); err != nil {
		return err
	}
	for _, mapping := range r.servlets {
		if mapping.pattern == pattern {
			return servlet.ErrDuplicateMapping
		}
	}
	if name == "" {
		name = pattern
	}
	r.servlets = append(r.servlets, servletMapping{
		pattern: pattern,
		name:    name,
		handler: handler,
		filters: cleanServletFilters(filters),
	})
	return nil
}

func (r *Registry) servletMappings() []servletMapping {
	if r == nil {
		return nil
	}
	mappings := make([]servletMapping, 0, len(r.servlets))
	for _, mapping := range r.servlets {
		mapping.filters = append([]servlet.Filter(nil), mapping.filters...)
		mappings = append(mappings, mapping)
	}
	return mappings
}

func cleanServletFilters(filters []servlet.Filter) []servlet.Filter {
	cleaned := make([]servlet.Filter, 0, len(filters))
	for _, filter := range filters {
		if !isNilFilter(filter) {
			cleaned = append(cleaned, filter)
		}
	}
	return cleaned
}

func servletMappingFilters(global []servlet.Filter, local []servlet.Filter) []servlet.Filter {
	if len(global) == 0 && len(local) == 0 {
		return nil
	}
	filters := make([]servlet.Filter, 0, len(global)+len(local))
	filters = append(filters, global...)
	filters = append(filters, local...)
	return filters
}

func isNilServlet(handler servlet.Servlet) bool {
	return isNilWebValue(handler)
}

// CheckNotModified 根据请求条件头判断资源是否未变化。
func CheckNotModified(ctx *arkweb.Context, etag string, lastModified time.Time) bool {
	if ctx == nil || ctx.Request() == nil || ctx.Response() == nil || ctx.Response().Committed() {
		return false
	}
	req := ctx.Request()
	if !conditionalMethodAllowed(req.Method()) {
		return false
	}
	validator := newConditionalValidator(etag, lastModified)
	if validator.empty() {
		return false
	}
	validator.write(ctx.Response().Header())
	if !validator.notModified(req.Header()) {
		return false
	}
	ctx.Response().Header().Delete("Content-Length")
	ctx.Response().SetStatus(http.StatusNotModified)
	return true
}

type conditionalValidator struct {
	etag         string
	lastModified time.Time
}

func newConditionalValidator(etag string, lastModified time.Time) conditionalValidator {
	return conditionalValidator{
		etag:         cleanETag(etag, false),
		lastModified: truncateHTTPTime(lastModified),
	}
}

func (v conditionalValidator) empty() bool {
	return v.etag == "" && v.lastModified.IsZero()
}

func (v conditionalValidator) write(header servlet.Header) {
	if v.etag != "" {
		header.Set("ETag", v.etag)
	}
	if !v.lastModified.IsZero() {
		header.Set("Last-Modified", v.lastModified.UTC().Format(http.TimeFormat))
	}
}

func (v conditionalValidator) notModified(header servlet.Header) bool {
	if len(header.Values("If-None-Match")) > 0 {
		return conditionalETagMatch(header.Values("If-None-Match"), v.etag)
	}
	return conditionalModifiedSince(header.Get("If-Modified-Since"), v.lastModified)
}

func conditionalETagMatch(values []string, etag string) bool {
	if etag == "" {
		return false
	}
	normalized := normalizeConditionalETag(etag)
	for _, value := range values {
		for _, item := range strings.Split(value, ",") {
			item = strings.TrimSpace(item)
			if item == "*" || normalizeConditionalETag(item) == normalized {
				return true
			}
		}
	}
	return false
}

func normalizeConditionalETag(value string) string {
	value = strings.TrimSpace(value)
	value = strings.TrimPrefix(value, "W/")
	return value
}

func conditionalModifiedSince(value string, lastModified time.Time) bool {
	if value == "" || lastModified.IsZero() {
		return false
	}
	since, err := http.ParseTime(value)
	if err != nil {
		return false
	}
	return !lastModified.After(truncateHTTPTime(since))
}

func conditionalMethodAllowed(method string) bool {
	return strings.EqualFold(method, http.MethodGet) || strings.EqualFold(method, http.MethodHead)
}

func truncateHTTPTime(value time.Time) time.Time {
	if value.IsZero() {
		return time.Time{}
	}
	return value.UTC().Truncate(time.Second)
}

// StreamWriter 是 Goark Web 流式写入器。
type StreamWriter = stream.Writer

// StreamFunc 是 Goark Web 流式写入函数。
type StreamFunc = stream.WriterFunc

// SSEWriter 是 Server-Sent Events 写入器。
type SSEWriter = stream.SSEWriter

// SSEFunc 是 Server-Sent Events 事件生产函数。
type SSEFunc = stream.SSEFunc

// SSEEvent 描述一个 Server-Sent Events 事件。
type SSEEvent = stream.Event

// Stream 创建指定媒体类型的流式响应。
func Stream(contentType string, write StreamFunc, options ...stream.Option) arkweb.Result {
	return stream.New(contentType, write, options...)
}

// TextStream 创建 text/plain 流式响应。
func TextStream(write StreamFunc, options ...stream.Option) arkweb.Result {
	return stream.Text(write, options...)
}

// BinaryStream 创建 application/octet-stream 流式响应。
func BinaryStream(write StreamFunc, options ...stream.Option) arkweb.Result {
	return stream.Binary(write, options...)
}

// SSE 创建 Server-Sent Events 响应。
func SSE(write SSEFunc, options ...stream.Option) arkweb.Result {
	return stream.Events(write, options...)
}
