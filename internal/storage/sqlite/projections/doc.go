// Package projections provides the SQLite-backed projection store adapter.
//
// It persists rebuildable read models and projection repair watermarks behind
// the service projection-store port. This package is the main SQLite adapter to
// read when you want to understand how replayed campaign state becomes a
// query-friendly summary surface.
package projections
