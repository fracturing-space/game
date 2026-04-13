package service

import (
	"time"

	"github.com/fracturing-space/game/internal/campaign"
	"github.com/fracturing-space/game/internal/event"
	"github.com/fracturing-space/game/internal/readiness"
	"github.com/fracturing-space/game/internal/scene"
	"github.com/fracturing-space/game/internal/session"
)

// PreparedCommit is one accepted command batch ready to be persisted into the
// journal.
type PreparedCommit struct {
	Events []event.Envelope
}

// PlannedCommand is the fully planned result for one command.
type PlannedCommand struct {
	Accepted bool
	Events   []event.Envelope
	State    campaign.Snapshot
}

// Result is the accepted result for one executed command.
type Result struct {
	Accepted     bool
	Events       []event.Envelope
	StoredEvents []event.Record
	State        campaign.Snapshot
}

// Inspection exposes the stored timeline and replayed state.
type Inspection struct {
	Timeline []event.Record
	State    campaign.Snapshot
	HeadSeq  uint64
}

// CampaignSummary exposes the launcher-oriented campaign list row.
type CampaignSummary struct {
	CampaignID       string
	Name             string
	ReadyToPlay      bool
	HasAIBinding     bool
	HasActiveSession bool
	LastActivityAt   time.Time
}

// CommandPlan exposes one prepared command batch and its projected state.
type CommandPlan struct {
	Token      string
	CampaignID string
	BaseSeq    uint64
	Commits    []PreparedCommit
	State      campaign.Snapshot
}

// ExecutedPlan exposes the committed head after one stored plan is executed.
type ExecutedPlan struct {
	CampaignID string
	HeadSeq    uint64
	State      campaign.Snapshot
}

// EventStream exposes one authorized event feed.
type EventStream struct {
	Records <-chan event.Record
	Close   func()
}

// PlayReadiness exposes deterministic blockers preventing entry into PLAY.
type PlayReadiness = readiness.Report

func cloneEnvelopes(events []event.Envelope) []event.Envelope {
	if len(events) == 0 {
		return nil
	}
	cloned := make([]event.Envelope, 0, len(events))
	for _, next := range events {
		cloned = append(cloned, event.Envelope{
			CampaignID: next.CampaignID,
			Message:    cloneEventMessage(next.Message),
		})
	}
	return cloned
}

func clonePreparedCommits(commits []PreparedCommit) []PreparedCommit {
	if len(commits) == 0 {
		return nil
	}
	cloned := make([]PreparedCommit, 0, len(commits))
	for _, commit := range commits {
		cloned = append(cloned, PreparedCommit{Events: cloneEnvelopes(commit.Events)})
	}
	return cloned
}

func cloneEventRecords(records []event.Record) []event.Record {
	if len(records) == 0 {
		return nil
	}
	cloned := make([]event.Record, 0, len(records))
	for _, next := range records {
		cloned = append(cloned, cloneEventRecord(next))
	}
	return cloned
}

func cloneEventRecord(record event.Record) event.Record {
	return event.Record{
		Seq:        record.Seq,
		CommitSeq:  record.CommitSeq,
		RecordedAt: record.RecordedAt,
		Envelope: event.Envelope{
			CampaignID: record.CampaignID,
			Message:    cloneEventMessage(record.Message),
		},
	}
}

// cloneEventMessage must deep-clone every registered event message type that
// carries slice or map fields so returned snapshots and streams never alias
// mutable internal state.
func cloneEventMessage(message event.Message) event.Message {
	switch typed := message.(type) {
	case scene.Created:
		typed.CharacterIDs = append([]string(nil), typed.CharacterIDs...)
		return typed
	case scene.CastReplaced:
		typed.CharacterIDs = append([]string(nil), typed.CharacterIDs...)
		return typed
	case session.Started:
		typed.CharacterControllers = session.CloneAssignments(typed.CharacterControllers)
		return typed
	case session.Ended:
		typed.CharacterControllers = session.CloneAssignments(typed.CharacterControllers)
		return typed
	default:
		return message
	}
}
