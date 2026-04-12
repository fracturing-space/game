// Package character defines the character contracts that are stored inside a
// campaign timeline.
//
// It owns the command and event messages for character creation plus the state
// record that appears in replayed campaign state and public snapshots. The
// package explains what a character looks like, not how character behavior is
// decided.
//
// Start here when you need to change character data or validation. If you need
// to change the behavior that emits or folds character events, read
// internal/modules/character next.
package character
