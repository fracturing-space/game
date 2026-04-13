// Package participant defines the participant contracts used for campaign
// membership.
//
// It owns participant command and event messages plus the replayed participant
// record stored inside campaign state and public snapshots. This package is the
// source of truth for participant identity, role, access, controller, and
// subject binding fields.
//
// Start here when you need to change participant data or validation. If you
// need to change how participants join or update campaign state, continue to
// internal/modules/participant.
package participant
