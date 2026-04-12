// Package campaign implements the engine module for campaign-owned
// behavior.
//
// It exists to turn validated campaign commands into events and to fold
// campaign events back into replayed state. This is where campaign lifecycle
// behavior lives, including creation, AI binding, and mode transitions.
// Some campaign commands deliberately emit events owned by other modules, such
// as participant join, session lifecycle, or scene lifecycle events, so one
// command can remain atomic on the single campaign timeline.
//
// Contributors should read internal/campaign first to learn the contract
// shapes, then come here to change the behavior behind those contracts.
package campaign
