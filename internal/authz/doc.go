// Package authz defines the authorization capabilities and checks used across
// campaign operations.
//
// It exists to keep permission rules named, reusable, and transport-neutral.
// The package answers who may perform an action against the current campaign
// state without taking on broader request admission or command behavior.
//
// Start here when you need to change who is allowed to do something. If the
// problem is about mode gating or command planning routing, continue to
// internal/admission instead.
package authz
