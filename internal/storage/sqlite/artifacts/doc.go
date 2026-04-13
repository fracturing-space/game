// Package artifacts provides the SQLite-backed artifact store adapter.
//
// It persists slower-moving authored campaign documents behind the service
// artifact-store port. This package is intentionally narrower than the journal
// and projection adapters: it owns path normalization and CRUD-style document
// persistence, not replay or subscription behavior.
package artifacts
