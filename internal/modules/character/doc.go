// Package character implements the engine module for character behavior.
//
// It exists to own the decision and fold logic for character commands and
// events after the contracts have been validated. In practice, this is where
// character creation behavior is attached to the shared campaign aggregate.
//
// Contributors should read internal/character first for the message and record
// shapes, then use this package when changing how those contracts behave.
package character
