// Package admission defines the pre-decision rules that decide whether a
// command may be attempted right now.
//
// It exists to keep request-time policy separate from domain behavior. This
// package combines authorization checks, allowed campaign modes, and command
// planning routing into one catalog that the service can apply before handing
// a command to a domain module.
//
// Admission is intentionally the first phase of authorization, not the only
// phase. It answers coarse capability questions such as "is the caller
// authenticated?" and "is this command allowed in the current play state?"
// Resource-specific ownership checks that depend on the concrete participant,
// character, or scene being targeted still happen inside module Decide logic.
//
// Contributors should start here when the question is "can this command run in
// this context?" Reusable permission predicates live in internal/authz, while
// command-specific behavior lives in internal/modules/....
package admission
