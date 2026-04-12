// Package memory provides in-memory storage adapters for the game service
// ports.
//
// It exists for tests, local experiments, and other ephemeral runtimes where a
// durable SQLite-backed store would add unnecessary setup. The package mirrors
// the service storage ports closely, so contributors can usually read it as the
// simplest reference implementation before looking at the SQLite adapters.
//
// Start here when tracing storage behavior in tests or when you want the
// smallest implementation of journaling, projections, artifacts, and event
// subscriptions.
package memory
