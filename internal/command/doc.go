// Package command defines the generic typed command infrastructure shared by
// the game domain.
//
// It owns command envelopes, command definitions, and the catalog used to
// validate that a concrete message matches a registered command contract. Most
// contributors touch this package indirectly through package-specific command
// specs such as those in internal/campaign or internal/session.
//
// Read this package when you are adding a new command contract or need to
// understand how commands are registered before they reach the engine and
// service layers.
package command
