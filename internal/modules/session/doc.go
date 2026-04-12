// Package session implements the engine module for session lifecycle
// behavior.
//
// It exists to own the command-to-event and event-to-state rules for starting
// and ending sessions within a campaign. That includes the session-side pieces
// of campaign mode transitions and character controller assignment defaults.
// Session command-level authorization currently stays in the admission catalog
// because the active rules are coarse owner checks rather than resource-level
// decisions inside Decide.
//
// Contributors should read internal/session first for the session contracts,
// then come here when changing how sessions start, end, or affect replayed
// campaign state.
package session
