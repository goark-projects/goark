package locale

import (
	"goark.dev/arkarta/servlet"
	arkweb "goark.dev/arkarta/web"
)

// FixedResolver 始终返回固定 Locale。
type FixedResolver struct {
	locale servlet.Locale
}

// NewFixedResolver 创建固定 Locale 解析器。
func NewFixedResolver(locale servlet.Locale) (*FixedResolver, error) {
	if !validLocale(locale) {
		return nil, ErrInvalidLocale
	}
	return &FixedResolver{locale: locale}, nil
}

// ResolveLocale 返回固定 Locale。
func (r *FixedResolver) ResolveLocale(*arkweb.Context) (servlet.Locale, bool) {
	if r == nil || !validLocale(r.locale) {
		return servlet.Locale{}, false
	}
	return r.locale, true
}
