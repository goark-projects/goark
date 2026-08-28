package view

import (
	"html/template"
	"io/fs"
	"path"
	"strings"

	arkweb "goark.dev/arkarta/web"
)

const (
	defaultTemplateSuffix      = ".html"
	defaultTemplateContentType = "text/html; charset=utf-8"
)

// TemplateOption 定制模板解析器。
type TemplateOption func(*templateConfig) error

type templateConfig struct {
	prefix      string
	suffix      string
	contentType string
	funcs       template.FuncMap
}

// TemplateResolver 基于 html/template 和 fs.FS 解析逻辑视图名。
type TemplateResolver struct {
	templates   *template.Template
	prefix      string
	suffix      string
	contentType string
}

// NewTemplateResolver 创建模板视图解析器。
func NewTemplateResolver(root fs.FS, options ...TemplateOption) (*TemplateResolver, error) {
	if root == nil {
		return nil, fs.ErrInvalid
	}
	config := templateConfig{
		suffix:      defaultTemplateSuffix,
		contentType: defaultTemplateContentType,
	}
	for _, option := range options {
		if option == nil {
			continue
		}
		if err := option(&config); err != nil {
			return nil, err
		}
	}
	names, err := templateNames(root, config.suffix)
	if err != nil {
		return nil, err
	}
	if len(names) == 0 {
		return nil, ErrNoTemplates
	}
	templates := template.New("")
	if len(config.funcs) > 0 {
		templates.Funcs(config.funcs)
	}
	if _, err := templates.ParseFS(root, names...); err != nil {
		return nil, err
	}
	return &TemplateResolver{
		templates:   templates,
		prefix:      config.prefix,
		suffix:      config.suffix,
		contentType: config.contentType,
	}, nil
}

// WithPrefix 设置逻辑视图名前缀。
func WithPrefix(prefix string) TemplateOption {
	return func(config *templateConfig) error {
		cleaned, err := cleanTemplatePrefix(prefix)
		if err != nil {
			return err
		}
		config.prefix = cleaned
		return nil
	}
}

// WithSuffix 设置逻辑视图名后缀。
func WithSuffix(suffix string) TemplateOption {
	return func(config *templateConfig) error {
		config.suffix = strings.TrimSpace(suffix)
		return nil
	}
}

// WithTemplateContentType 设置模板视图默认媒体类型。
func WithTemplateContentType(contentType string) TemplateOption {
	return func(config *templateConfig) error {
		contentType = cleanHeaderValue(contentType)
		if contentType != "" {
			config.contentType = contentType
		}
		return nil
	}
}

// WithFuncs 设置模板函数表。
func WithFuncs(funcs template.FuncMap) TemplateOption {
	return func(config *templateConfig) error {
		if len(funcs) == 0 {
			return nil
		}
		config.funcs = make(template.FuncMap, len(funcs))
		for name, fn := range funcs {
			config.funcs[name] = fn
		}
		return nil
	}
}

// ResolveView 解析逻辑视图名。
func (r *TemplateResolver) ResolveView(_ *arkweb.Context, name string) (View, bool, error) {
	if r == nil || r.templates == nil {
		return nil, false, ErrNilResolver
	}
	templateName, err := r.templateName(name)
	if err != nil {
		return nil, false, err
	}
	actualName, ok := r.lookupTemplateName(templateName)
	if !ok {
		return nil, false, nil
	}
	return templateView{
		templates:   r.templates,
		name:        actualName,
		contentType: r.contentType,
	}, true, nil
}

func (r *TemplateResolver) lookupTemplateName(name string) (string, bool) {
	if r.templates.Lookup(name) != nil {
		return name, true
	}
	base := path.Base(name)
	if base != name && r.templates.Lookup(base) != nil {
		return base, true
	}
	return "", false
}

func (r *TemplateResolver) templateName(name string) (string, error) {
	cleaned, err := cleanTemplateName(name)
	if err != nil {
		return "", err
	}
	if r.suffix != "" && !strings.HasSuffix(cleaned, r.suffix) {
		cleaned += r.suffix
	}
	if r.prefix != "" && !strings.HasPrefix(cleaned, r.prefix) {
		cleaned = path.Join(r.prefix, cleaned)
	}
	return cleaned, nil
}

type templateView struct {
	templates   *template.Template
	name        string
	contentType string
}

func (v templateView) ContentType() string {
	return v.contentType
}

func (v templateView) Render(ctx *arkweb.Context, model any) error {
	if ctx == nil || ctx.Response() == nil {
		return arkweb.ErrNilContext
	}
	return v.templates.ExecuteTemplate(ctx.Response().BodyWriter(), v.name, model)
}

func templateNames(root fs.FS, suffix string) ([]string, error) {
	names := make([]string, 0)
	err := fs.WalkDir(root, ".", func(name string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry == nil || entry.IsDir() {
			return nil
		}
		if suffix == "" || strings.HasSuffix(name, suffix) {
			names = append(names, name)
		}
		return nil
	})
	return names, err
}
