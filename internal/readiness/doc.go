// Package readiness defines deterministic play-readiness evaluation and blocker
// reporting for campaigns.
//
// It exists to answer a contributor-friendly question: why can't this campaign
// enter PLAY yet? The package reports stable blocker codes, messages, and
// resolution hints without taking on transport concerns or session command
// behavior itself.
//
// Start here when changing readiness blockers or their actionable metadata. If
// you are changing who may transition modes or when commands are admitted,
// inspect internal/authz or internal/admission as well.
package readiness
