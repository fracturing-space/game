// Package errs defines classified domain failures that transports can map
// to stable protocol errors.
//
// It exists so domain code can report not found, conflict, invalid argument,
// and similar outcomes without depending on gRPC or any other transport. That
// keeps failure meaning stable even as adapters change.
//
// Start here when you want a domain operation to fail in a predictable,
// transport-neutral way. The gRPC mapping for these errors lives in
// internal/transport/grpc/gamev1.
package errs
