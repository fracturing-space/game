package service

import (
	"fmt"
	"maps"
	"strings"
)

// sequentialIDAllocator is test-only. Because it is process-local and
// in-memory, identifiers restart from 1 on each service start.
type sequentialIDAllocator struct {
	next map[string]uint64
}

func newSequentialIDAllocator() *sequentialIDAllocator {
	return &sequentialIDAllocator{
		next: make(map[string]uint64),
	}
}

func (a *sequentialIDAllocator) Session(commit bool) IDSession {
	next := make(map[string]uint64, len(a.next))
	maps.Copy(next, a.next)
	return &sequentialIDSession{
		allocator: a,
		next:      next,
		commit:    commit,
	}
}

type sequentialIDSession struct {
	allocator *sequentialIDAllocator
	next      map[string]uint64
	commit    bool
}

func (s *sequentialIDSession) NewID(prefix string) (string, error) {
	prefix = strings.TrimSpace(prefix)
	if prefix == "" {
		return "", fmt.Errorf("id prefix is required")
	}
	s.next[prefix]++
	return fmt.Sprintf("%s-%d", prefix, s.next[prefix]), nil
}

func (s *sequentialIDSession) Commit() {
	if s == nil || !s.commit || s.allocator == nil {
		return
	}
	s.allocator.next = s.next
}
