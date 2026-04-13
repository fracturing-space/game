// Package campaign defines the core campaign contracts for the aggregate the
// repository revolves around.
//
// It owns campaign-level command and event messages, replayed state, public
// snapshot shape, and helpers for campaign-specific invariants such as mode
// handling and participant lookups. This is the place to learn what a campaign
// is in this codebase.
//
// Contributors changing campaign data or contract shapes should start here.
// Contributors changing how campaign commands turn into events or how events
// fold into state should continue to internal/modules/campaign.
package campaign
