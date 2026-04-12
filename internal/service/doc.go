// Package service orchestrates the replay-first game runtime.
//
// It exists to assemble the generic building blocks of the system into
// caller-scoped operations: command admission, replay, journaling, command
// planning or execution, reads, and identifier allocation. This package is the
// main place where the aggregate is loaded, checked, and executed end to end.
//
// The current public service surface spans campaign authoring, participant
// management, character lifecycle, play readiness, and internal command
// planning or execution orchestration. Production command behavior stays in
// the domain modules; this package wires those modules into one campaign-owned
// aggregate and exposes the end-to-end execution path used by transports.
//
// Contributors tracing an RPC through the domain usually end up here after
// reading the relevant contract and module packages. Transport adapters stay in
// internal/transport/grpc/gamev1, and domain-specific command behavior stays in
// internal/modules/....
package service
