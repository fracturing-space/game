// Package server parses command configuration and starts the local game gRPC
// service runtime.
//
// It keeps process concerns such as environment defaults, flag handling, and
// serve-loop shutdown out of cmd/server/main.go so the executable remains a
// thin lifecycle wrapper.
package server
