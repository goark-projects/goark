package web

import (
	"context"
	"net/http"
	"sort"
	"strings"

	"goark.dev/arkarta/servlet"
	arkweb "goark.dev/arkarta/web"
	"goark.dev/goark/web/cors"
)

const (
	headerCORSOrigin        = "Origin"
	headerCORSRequestMethod = "Access-Control-Request-Method"
)

// CORSMapping 描述指定路径和方法的局部 CORS 策略。
type CORSMapping struct {
	Pattern string
	Methods []string
	Config  cors.Config
}

// AddCORSMapping 添加路径级 CORS 映射，Router 构造时会聚合预检路由。
func (r *Registry) AddCORSMapping(pattern string, methods []string, config cors.Config) error {
	if r == nil {
		return ErrNilRegistry
	}
	pattern = strings.TrimSpace(pattern)
	cleanedMethods := normalizeCORSMethods(methods)
	if pattern == "" || len(cleanedMethods) == 0 {
		return ErrInvalidRoute
	}
	normalized := normalizeCORSConfig(config, cleanedMethods)
	if _, err := cors.New(normalized); err != nil {
		return err
	}
	r.corsMappings = append(r.corsMappings, CORSMapping{
		Pattern: pattern,
		Methods: cleanedMethods,
		Config:  normalized,
	})
	return nil
}

// CORSMappings 返回局部 CORS 映射快照。
func (r *Registry) CORSMappings() []CORSMapping {
	if r == nil {
		return nil
	}
	out := make([]CORSMapping, 0, len(r.corsMappings))
	for _, mapping := range r.corsMappings {
		out = append(out, cloneCORSMapping(mapping))
	}
	return out
}

func applyCORSMappings(routes []Route, mappings []CORSMapping) ([]Route, error) {
	if len(mappings) == 0 {
		return append([]Route(nil), routes...), nil
	}
	table, err := newCORSRouteTable(mappings)
	if err != nil {
		return nil, err
	}
	out := append([]Route(nil), routes...)
	explicitOptions := make(map[string]struct{}, len(out))
	for i := range out {
		method := normalizeMethod(out[i].Method)
		if method == http.MethodOptions {
			explicitOptions[out[i].Pattern] = struct{}{}
			if handler := table.preflight[out[i].Pattern]; handler != nil && out[i].Handler != nil {
				out[i].Handler = handler.withFallback(out[i].Handler)
			}
			continue
		}
		if filter := table.actualFilter(out[i].Pattern, method); filter != nil && out[i].Handler != nil {
			out[i].Handler = wrapCORSHandler(out[i].Handler, filter)
		}
	}
	for _, pattern := range table.patterns() {
		if _, ok := explicitOptions[pattern]; ok {
			continue
		}
		out = append(out, Route{
			Method:  http.MethodOptions,
			Pattern: pattern,
			Handler: table.preflight[pattern],
		})
	}
	return out, nil
}

type corsRouteKey struct {
	pattern string
	method  string
}

type corsRouteTable struct {
	actual    map[corsRouteKey]servlet.Filter
	preflight map[string]*corsPreflightHandler
}

func newCORSRouteTable(mappings []CORSMapping) (*corsRouteTable, error) {
	table := &corsRouteTable{
		actual:    make(map[corsRouteKey]servlet.Filter, len(mappings)),
		preflight: make(map[string]*corsPreflightHandler, len(mappings)),
	}
	for _, mapping := range mappings {
		filter, err := cors.New(mapping.Config)
		if err != nil {
			return nil, err
		}
		for _, method := range mapping.Methods {
			key := corsRouteKey{pattern: mapping.Pattern, method: method}
			table.actual[key] = filter
			handler := table.preflight[mapping.Pattern]
			if handler == nil {
				handler = newCORSPreflightHandler()
				table.preflight[mapping.Pattern] = handler
			}
			handler.filters[method] = filter
		}
	}
	return table, nil
}

func (t *corsRouteTable) actualFilter(pattern string, method string) servlet.Filter {
	if t == nil {
		return nil
	}
	return t.actual[corsRouteKey{pattern: pattern, method: method}]
}

func (t *corsRouteTable) patterns() []string {
	if t == nil || len(t.preflight) == 0 {
		return nil
	}
	patterns := make([]string, 0, len(t.preflight))
	for pattern := range t.preflight {
		patterns = append(patterns, pattern)
	}
	sort.Strings(patterns)
	return patterns
}

type corsPreflightHandler struct {
	filters  map[string]servlet.Filter
	fallback arkweb.Handler
}

func newCORSPreflightHandler() *corsPreflightHandler {
	return &corsPreflightHandler{filters: make(map[string]servlet.Filter)}
}

func (h *corsPreflightHandler) withFallback(fallback arkweb.Handler) arkweb.Handler {
	if h == nil {
		return fallback
	}
	clone := &corsPreflightHandler{
		filters:  h.filters,
		fallback: fallback,
	}
	return clone
}

func (h *corsPreflightHandler) Handle(ctx *arkweb.Context) (arkweb.Result, error) {
	if !isCORSPreflight(ctx) {
		if h != nil && h.fallback != nil {
			return h.fallback.Handle(ctx)
		}
		if h != nil {
			h.writeAllowHeader(ctx)
		}
		return arkweb.NoContent(), nil
	}
	method := normalizeMethod(ctx.Request().Header().Get(headerCORSRequestMethod))
	filter := h.filters[method]
	if filter == nil {
		return nil, servlet.NewHTTPError(http.StatusForbidden, http.StatusText(http.StatusForbidden), nil)
	}
	if err := filter.Filter(ctx.Context(), ctx.Request(), ctx.Response(), servlet.ChainFunc(func(context.Context, *servlet.Request, servlet.Response) error {
		return nil
	})); err != nil {
		return nil, err
	}
	return nil, nil
}

func (h *corsPreflightHandler) writeAllowHeader(ctx *arkweb.Context) {
	if h == nil || ctx == nil || ctx.Response() == nil {
		return
	}
	methods := make([]string, 0, len(h.filters)+1)
	seen := make(map[string]struct{}, len(h.filters)+1)
	for method := range h.filters {
		if _, ok := seen[method]; ok {
			continue
		}
		seen[method] = struct{}{}
		methods = append(methods, method)
	}
	if _, ok := seen[http.MethodOptions]; !ok {
		methods = append(methods, http.MethodOptions)
	}
	sort.Strings(methods)
	ctx.Response().Header().Set("Allow", strings.Join(methods, ", "))
}

func wrapCORSHandler(handler arkweb.Handler, filter servlet.Filter) arkweb.Handler {
	if handler == nil || filter == nil {
		return handler
	}
	return arkweb.HandlerFunc(func(ctx *arkweb.Context) (arkweb.Result, error) {
		if ctx == nil || ctx.Request() == nil || ctx.Response() == nil {
			return handler.Handle(ctx)
		}
		chain := &corsResultChain{ctx: ctx, handler: handler}
		if err := filter.Filter(ctx.Context(), ctx.Request(), ctx.Response(), chain); err != nil {
			return nil, err
		}
		return chain.result, nil
	})
}

type corsResultChain struct {
	ctx     *arkweb.Context
	handler arkweb.Handler
	result  arkweb.Result
}

func (c *corsResultChain) Next(context.Context, *servlet.Request, servlet.Response) error {
	result, err := c.handler.Handle(c.ctx)
	c.result = result
	return err
}

func isCORSPreflight(ctx *arkweb.Context) bool {
	if ctx == nil || ctx.Request() == nil {
		return false
	}
	req := ctx.Request()
	return strings.EqualFold(req.Method(), http.MethodOptions) &&
		strings.TrimSpace(req.Header().Get(headerCORSOrigin)) != "" &&
		strings.TrimSpace(req.Header().Get(headerCORSRequestMethod)) != ""
}

func normalizeCORSConfig(config cors.Config, methods []string) cors.Config {
	out := cloneCORSConfig(config)
	if len(out.AllowedOrigins) == 0 && len(out.AllowedOriginPatterns) == 0 {
		out.AllowedOrigins = []string{cors.AllOrigins}
	}
	if len(out.AllowedMethods) == 0 {
		out.AllowedMethods = append([]string(nil), methods...)
	}
	if len(out.AllowedHeaders) == 0 {
		out.AllowedHeaders = []string{cors.AllHeaders}
	}
	if out.MaxAge == 0 {
		out.MaxAge = cors.PermitDefaultValues().MaxAge
	}
	return out
}

func normalizeCORSMethods(methods []string) []string {
	out := make([]string, 0, len(methods))
	seen := make(map[string]struct{}, len(methods))
	for _, method := range methods {
		method = normalizeMethod(method)
		if method == "" {
			continue
		}
		if _, ok := seen[method]; ok {
			continue
		}
		seen[method] = struct{}{}
		out = append(out, method)
	}
	return out
}

func cloneCORSMapping(mapping CORSMapping) CORSMapping {
	return CORSMapping{
		Pattern: mapping.Pattern,
		Methods: append([]string(nil), mapping.Methods...),
		Config:  cloneCORSConfig(mapping.Config),
	}
}

func cloneCORSConfig(config cors.Config) cors.Config {
	return cors.Config{
		AllowedOrigins:        append([]string(nil), config.AllowedOrigins...),
		AllowedOriginPatterns: append([]string(nil), config.AllowedOriginPatterns...),
		AllowedMethods:        append([]string(nil), config.AllowedMethods...),
		AllowedHeaders:        append([]string(nil), config.AllowedHeaders...),
		ExposedHeaders:        append([]string(nil), config.ExposedHeaders...),
		AllowCredentials:      config.AllowCredentials,
		MaxAge:                config.MaxAge,
	}
}
