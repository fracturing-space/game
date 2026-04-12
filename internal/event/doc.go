// Package event defines the generic typed event infrastructure for campaign
// timelines.
//
// It owns event envelopes, persisted event records, and the catalog used to
// validate registered event contracts. Domain packages such as
// internal/campaign and internal/participant declare specific event messages on
// top of these generic building blocks.
//
// Read this package when you are adding a new event contract or tracing how the
// service stores and validates timeline events during replay and inspection.
package event
