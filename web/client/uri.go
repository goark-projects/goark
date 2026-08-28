package client

import (
	"net/url"
	"strings"
)

func resolveURL(base *url.URL, target string, pathVariables map[string]string, query url.Values) (string, error) {
	target = expandPathVariables(strings.TrimSpace(target), pathVariables)
	if target == "" {
		return "", ErrInvalidRequest
	}
	parsed, err := url.Parse(target)
	if err != nil {
		return "", ErrInvalidRequest
	}
	if !parsed.IsAbs() {
		if base == nil {
			return "", ErrInvalidBaseURL
		}
		parsed = mergeURL(base, parsed)
	}
	values := parsed.Query()
	for name, items := range query {
		for _, item := range items {
			values.Add(name, item)
		}
	}
	parsed.RawQuery = values.Encode()
	return parsed.String(), nil
}

func expandPathVariables(target string, variables map[string]string) string {
	if len(variables) == 0 {
		return target
	}
	for name, value := range variables {
		target = strings.ReplaceAll(target, "{"+name+"}", url.PathEscape(value))
	}
	return target
}

func mergeURL(base *url.URL, target *url.URL) *url.URL {
	out := *base
	if target.Scheme != "" {
		out.Scheme = target.Scheme
	}
	if target.Host != "" {
		out.Host = target.Host
	}
	out.Path = joinPath(base.Path, target.Path)
	out.RawPath = ""
	out.RawQuery = target.RawQuery
	out.Fragment = target.Fragment
	return &out
}

func joinPath(basePath string, targetPath string) string {
	basePath = strings.TrimRight(basePath, "/")
	if targetPath == "" {
		if basePath == "" {
			return "/"
		}
		return basePath
	}
	if !strings.HasPrefix(targetPath, "/") {
		targetPath = "/" + targetPath
	}
	if basePath == "" || basePath == "/" {
		return targetPath
	}
	return basePath + targetPath
}
