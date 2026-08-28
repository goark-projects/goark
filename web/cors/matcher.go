package cors

import (
	"net/http"
	"strings"
)

func (c compiledConfig) originAllowed(origin string) bool {
	if origin == "" {
		return false
	}
	if c.allowAllOrigins {
		return true
	}
	if _, ok := c.allowedOrigins[origin]; ok {
		return true
	}
	for _, pattern := range c.allowedOriginPatterns {
		if globMatch(pattern, origin) {
			return true
		}
	}
	return false
}

func (c compiledConfig) methodAllowed(method string) bool {
	method = strings.ToUpper(strings.TrimSpace(method))
	if method == "" {
		return false
	}
	if c.allowAllMethods {
		return true
	}
	if len(c.allowedMethodSet) == 0 {
		return method == http.MethodGet || method == http.MethodHead || method == http.MethodPost
	}
	_, ok := c.allowedMethodSet[method]
	return ok
}

func (c compiledConfig) headersAllowed(headers []string) bool {
	if len(headers) == 0 || c.allowAllHeaders {
		return true
	}
	if len(c.allowedHeaderSet) == 0 {
		return false
	}
	for _, header := range headers {
		if _, ok := c.allowedHeaderSet[strings.ToLower(header)]; !ok {
			return false
		}
	}
	return true
}

func requestedHeaders(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	seen := make(map[string]struct{}, len(parts))
	for _, part := range parts {
		header := http.CanonicalHeaderKey(strings.TrimSpace(part))
		if header == "" {
			continue
		}
		key := strings.ToLower(header)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, header)
	}
	return out
}

func globMatch(pattern, value string) bool {
	patternIndex := 0
	valueIndex := 0
	starIndex := -1
	starValueIndex := 0
	for valueIndex < len(value) {
		if patternIndex < len(pattern) && (pattern[patternIndex] == '?' || pattern[patternIndex] == value[valueIndex]) {
			patternIndex++
			valueIndex++
			continue
		}
		if patternIndex < len(pattern) && pattern[patternIndex] == '*' {
			starIndex = patternIndex
			starValueIndex = valueIndex
			patternIndex++
			continue
		}
		if starIndex >= 0 {
			patternIndex = starIndex + 1
			starValueIndex++
			valueIndex = starValueIndex
			continue
		}
		return false
	}
	for patternIndex < len(pattern) && pattern[patternIndex] == '*' {
		patternIndex++
	}
	return patternIndex == len(pattern)
}
