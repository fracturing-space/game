// Package gamev1 adapts the game service to gRPC.
//
// It exists to translate between protobuf requests and responses and the
// internal service layer used by the rest of the repository. This package owns
// transport concerns such as metadata extraction, status mapping, streaming,
// and proto-level validation.
//
// Handlers here should stay thin. They convert between protobuf messages and
// domain types, while domain invariants and behavior continue to live in the
// shared command, event, module, and service packages. Contributors changing
// RPC shape start here; contributors changing game rules should usually keep
// reading deeper into internal/service and internal/modules/....
package gamev1
