package container

import (
	"context"
	"strings"
	"sync"
)

type resolutionKey struct{}

type resolutionState struct {
	mu     sync.Mutex
	path   []string
	active map[string]int
}

func ensureResolutionState(ctx context.Context) (context.Context, *resolutionState) {
	if state, ok := ctx.Value(resolutionKey{}).(*resolutionState); ok && state != nil {
		return ctx, state
	}
	state := &resolutionState{active: make(map[string]int)}
	return context.WithValue(ctx, resolutionKey{}, state), state
}

func (s *resolutionState) enter(name string) (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if index, exists := s.active[name]; exists {
		cycle := append([]string(nil), s.path[index:]...)
		cycle = append(cycle, name)
		return strings.Join(cycle, " -> "), true
	}
	s.active[name] = len(s.path)
	s.path = append(s.path, name)
	return "", false
}

func (s *resolutionState) exit(name string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	index, exists := s.active[name]
	if !exists {
		return
	}
	delete(s.active, name)
	if index >= 0 && index < len(s.path) {
		s.path = s.path[:index]
	}
}
