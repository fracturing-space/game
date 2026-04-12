package service

import (
	"crypto/rand"
	"encoding/base32"
	"fmt"
	"strings"
)

// IDSession allocates identifiers for one service planning session. The
// session seam exists so allocators that reserve transactional ID ranges can
// commit or discard them with the surrounding service operation.
type IDSession interface {
	NewID(prefix string) (string, error)
	Commit()
}

// IDAllocator returns transactional ID sessions for service planning.
type IDAllocator interface {
	Session(commit bool) IDSession
}

// NewOpaqueIDAllocator returns an allocator that generates non-repeating
// prefixed identifiers using UUIDv4 bytes encoded as lowercase base32.
func NewOpaqueIDAllocator() IDAllocator {
	return opaqueIDAllocator{}
}

type opaqueIDAllocator struct{}

// Session ignores commit intent because opaque IDs are generated on demand and
// never reserved ahead of time.
func (opaqueIDAllocator) Session(commit bool) IDSession {
	return opaqueIDSession{}
}

type opaqueIDSession struct{}

func (opaqueIDSession) NewID(prefix string) (string, error) {
	prefix = strings.TrimSpace(prefix)
	if prefix == "" {
		return "", fmt.Errorf("id prefix is required")
	}

	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", fmt.Errorf("read id entropy: %w", err)
	}

	// RFC 4122 variant and version bits for a v4 UUID.
	raw[6] = (raw[6] & 0x0f) | 0x40
	raw[8] = (raw[8] & 0x3f) | 0x80

	encoded := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(raw[:])
	return fmt.Sprintf("%s-%s", prefix, strings.ToLower(encoded)), nil
}

// Commit is intentionally a no-op for opaque random IDs because this allocator
// does not reserve or roll back identifier ranges.
func (opaqueIDSession) Commit() {}
