// Package session defines the session lifecycle contracts stored inside a
// campaign.
//
// It owns the command and event messages for starting and ending sessions, plus
// the session record and character-controller assignment types that appear in
// replayed campaign state and read models. This package explains the session
// shape without owning the runtime rules around it.
//
// Start here when changing session data or validation. If you need to change
// how sessions affect campaign mode or replayed state, continue to
// internal/modules/session.
package session
