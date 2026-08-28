package web

import (
	"goark.dev/arkarta/servlet"
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
func (r *Registry) AddServlet(pattern string, name string, handler servlet.Servlet, filters ...servlet.Filter) error {
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
