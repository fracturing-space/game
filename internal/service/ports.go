package service

import (
	"context"
	"time"

	"github.com/fracturing-space/game/internal/caller"
	"github.com/fracturing-space/game/internal/campaign"
	"github.com/fracturing-space/game/internal/command"
	"github.com/fracturing-space/game/internal/event"
)

// API is the inbound application surface implemented by the core service.
type API interface {
	CommitCommand(ctx context.Context, act caller.Caller, envelope command.Envelope) (Result, error)
	Inspect(ctx context.Context, act caller.Caller, campaignID string) (Inspection, error)
	ListCampaigns(ctx context.Context, act caller.Caller) ([]CampaignSummary, error)
	GetPlayReadiness(ctx context.Context, act caller.Caller, campaignID string) (PlayReadiness, error)
	PlanCommands(ctx context.Context, act caller.Caller, commands []command.Envelope) (CommandPlan, error)
	ExecutePlan(ctx context.Context, act caller.Caller, token string) (ExecutedPlan, error)
	ListEvents(ctx context.Context, act caller.Caller, campaignID string, afterSeq uint64) ([]event.Record, error)
	SubscribeEvents(ctx context.Context, act caller.Caller, campaignID string, afterSeq uint64) (EventStream, error)
	ReadResource(ctx context.Context, act caller.Caller, uri string) (string, error)
}

// Journal stores persisted campaign events. Implementations must be safe for
// concurrent use by multiple goroutines. Once AppendCommits returns
// successfully, appended records must be visible through List, ListAfter,
// HeadSeq, and SubscribeAfter. SubscribeAfter must not miss or duplicate
// committed records across the catch-up to live boundary for one campaign,
// delivered records must be safe for callers to mutate, and closing one
// subscription must close its records stream promptly.
type Journal interface {
	AppendCommits(context.Context, string, []PreparedCommit, func() time.Time) ([]event.Record, error)
	List(context.Context, string) ([]event.Record, bool, error)
	ListAfter(context.Context, string, uint64) ([]event.Record, bool, error)
	HeadSeq(context.Context, string) (uint64, bool, error)
	SubscribeAfter(context.Context, string, uint64) (EventSubscription, error)
}

// EventSubscription exposes one per-campaign committed-event tail stream.
type EventSubscription struct {
	Records <-chan event.Record
	Close   func()
}

// ProjectionSnapshot stores one rebuildable campaign read model at a journal
// sequence.
type ProjectionSnapshot struct {
	CampaignID     string
	HeadSeq        uint64
	State          campaign.State
	UpdatedAt      time.Time
	LastActivityAt time.Time
}

// ProjectionWatermark records how far projections have been applied for one
// campaign.
type ProjectionWatermark struct {
	CampaignID      string
	AppliedSeq      uint64
	ExpectedNextSeq uint64
	UpdatedAt       time.Time
}

// ProjectionStore persists rebuildable campaign projections and repair
// watermarks. Implementations must be safe for concurrent use by multiple
// goroutines, and returned values must not alias mutable internal state.
type ProjectionStore interface {
	GetProjection(context.Context, string) (ProjectionSnapshot, bool, error)
	SaveProjection(context.Context, ProjectionSnapshot) error
	GetWatermark(context.Context, string) (ProjectionWatermark, bool, error)
	SaveWatermark(context.Context, ProjectionWatermark) error
	SaveProjectionAndWatermark(context.Context, ProjectionSnapshot, ProjectionWatermark) error
	ListCampaignsBySubject(context.Context, string, int) ([]CampaignSummary, error)
}

// Artifact stores one campaign-scoped authored document.
type Artifact struct {
	CampaignID string
	Path       string
	Content    string
	UpdatedAt  time.Time
}

// ArtifactStore persists slower-moving campaign documents used for memory and
// retrieval surfaces. Implementations must be safe for concurrent use by
// multiple goroutines, and returned values must be safe for callers to mutate
// without affecting later reads.
type ArtifactStore interface {
	PutArtifact(context.Context, Artifact) error
	GetArtifact(context.Context, string, string) (Artifact, bool, error)
	ListArtifacts(context.Context, string) ([]Artifact, error)
}
