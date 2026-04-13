// Package main starts the development gRPC server used to exercise the game
// service locally.
//
// This package exists to keep the executable wiring thin and easy to inspect.
// It owns only process-level logging and signal handling, while
// internal/cmd/server owns configuration parsing and local server lifecycle.
//
// Contributors changing game behavior should usually read internal/service,
// internal/modules/..., or internal/transport/grpc/gamev1 next. Reach for this
// package when you need to change how the development server is assembled.
package main
