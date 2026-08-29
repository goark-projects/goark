package message

import (
	"strings"

	"goark.dev/arkarta/servlet"
)

const (
	// MediaTypeJSON 是默认 JSON 响应媒体类型。
	MediaTypeJSON = "application/json"
	// MediaTypeTextPlain 是默认文本响应媒体类型。
	MediaTypeTextPlain = "text/plain; charset=utf-8"
	// MediaTypeOctetStream 是默认二进制响应媒体类型。
	MediaTypeOctetStream = "application/octet-stream"
	// MediaTypeFormURLEncoded 是 URL 编码表单媒体类型。
	MediaTypeFormURLEncoded = "application/x-www-form-urlencoded"
)

// NegotiateContentType 按请求 Accept 头选择最合适的响应媒体类型。
func NegotiateContentType(request *servlet.Request, candidates ...string) (string, bool) {
	candidates = cleanMediaTypes(candidates)
	if len(candidates) == 0 {
		return "", false
	}
	if request == nil {
		return candidates[0], true
	}
	return request.NegotiateContentType(candidates...)
}

func cleanMediaTypes(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	out := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		key := strings.ToLower(value)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, value)
	}
	return out
}
