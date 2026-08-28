package filter

import "errors"

var (
	// ErrNilFilter 表示过滤器为空。
	ErrNilFilter = errors.New("goark/web/filter: filter is nil")
	// ErrNilChain 表示过滤器链为空。
	ErrNilChain = errors.New("goark/web/filter: chain is nil")
	// ErrInvalidFilterName 表示过滤器名称非法。
	ErrInvalidFilterName = errors.New("goark/web/filter: invalid filter name")
)
