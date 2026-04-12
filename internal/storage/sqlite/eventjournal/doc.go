// Package eventjournal provides the SQLite-backed journal adapter.
//
// It owns the durable event timeline for campaigns plus the in-process
// subscription bridge used by streaming reads. Contributors tracing write-path
// persistence or stored event replay usually start here before moving outward
// to the higher-level service orchestration.
package eventjournal
