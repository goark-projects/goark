package filter

import (
	"context"
	"net"
	"net/http"
	"strings"

	"goark.dev/arkarta/servlet"
)

const (
	// AttributeOriginalScheme 保存代理头改写前的请求协议。
	AttributeOriginalScheme = "goark.web.filter.forwarded.original_scheme"
	// AttributeOriginalHost 保存代理头改写前的请求主机。
	AttributeOriginalHost = "goark.web.filter.forwarded.original_host"
	// AttributeOriginalRemoteAddr 保存代理头改写前的远端地址。
	AttributeOriginalRemoteAddr = "goark.web.filter.forwarded.original_remote_addr"
)

type forwardedHeadersFilter struct{}

// ForwardedHeaders 按 RFC 7239 与 X-Forwarded-* 头修正请求外部视图。
func ForwardedHeaders() servlet.Filter {
	return forwardedHeadersFilter{}
}

func (forwardedHeadersFilter) Filter(ctx context.Context, req *servlet.Request, res servlet.Response, chain servlet.Chain) error {
	if req == nil {
		return servlet.NewHTTPError(http.StatusBadRequest, http.StatusText(http.StatusBadRequest), nil)
	}
	if chain == nil {
		return ErrNilChain
	}
	req.SetAttribute(AttributeOriginalScheme, req.Scheme())
	req.SetAttribute(AttributeOriginalHost, req.Host())
	req.SetAttribute(AttributeOriginalRemoteAddr, req.RemoteAddr())

	fields := forwardedFields(req.Header().Get("Forwarded"))
	scheme := firstNonEmpty(fields["proto"], firstHeaderValue(req.Header().Get("X-Forwarded-Proto")))
	host := firstNonEmpty(fields["host"], forwardedHost(req.Header()))
	remote := firstNonEmpty(fields["for"], firstForwardedFor(req.Header().Get("X-Forwarded-For")))

	if scheme = cleanScheme(scheme); scheme != "" {
		req.SetScheme(scheme)
	}
	if host = cleanHost(host); host != "" {
		req.SetHost(host)
	}
	if remote = cleanRemoteAddr(remote); remote != "" {
		req.SetRemoteAddr(remote)
	}
	return chain.Next(ctx, req, res)
}

func forwardedFields(value string) map[string]string {
	first := firstHeaderValue(value)
	if first == "" {
		return nil
	}
	out := make(map[string]string, 4)
	for _, part := range strings.Split(first, ";") {
		name, value, ok := strings.Cut(part, "=")
		if !ok {
			continue
		}
		name = strings.ToLower(strings.TrimSpace(name))
		value = strings.Trim(strings.TrimSpace(value), `"`)
		if name != "" && value != "" {
			out[name] = value
		}
	}
	return out
}

func forwardedHost(header servlet.Header) string {
	host := firstHeaderValue(header.Get("X-Forwarded-Host"))
	port := firstHeaderValue(header.Get("X-Forwarded-Port"))
	if host == "" || port == "" || strings.Contains(host, ":") {
		return host
	}
	return host + ":" + port
}

func firstForwardedFor(value string) string {
	candidate := firstHeaderValue(value)
	if candidate == "" {
		return ""
	}
	candidate = strings.Trim(candidate, `"`)
	if strings.HasPrefix(candidate, "[") {
		if host, _, err := net.SplitHostPort(candidate); err == nil {
			return strings.Trim(host, "[]")
		}
		return strings.Trim(candidate, "[]")
	}
	if host, _, err := net.SplitHostPort(candidate); err == nil {
		return host
	}
	return candidate
}

func firstHeaderValue(value string) string {
	if value == "" {
		return ""
	}
	item, _, _ := strings.Cut(value, ",")
	return strings.TrimSpace(item)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func cleanScheme(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	switch value {
	case "http", "https":
		return value
	default:
		return ""
	}
}

func cleanHost(value string) string {
	value = strings.TrimSpace(value)
	if value == "" || hasInvalidHeaderValueByte(value) || strings.ContainsAny(value, " /\\") {
		return ""
	}
	return value
}

func cleanRemoteAddr(value string) string {
	value = strings.Trim(strings.TrimSpace(value), `"`)
	if value == "" || hasInvalidHeaderValueByte(value) || strings.ContainsAny(value, " \t\r\n") {
		return ""
	}
	return value
}

func hasInvalidHeaderValueByte(value string) bool {
	for i := 0; i < len(value); i++ {
		if value[i] < 0x20 || value[i] == 0x7f {
			return true
		}
	}
	return false
}
