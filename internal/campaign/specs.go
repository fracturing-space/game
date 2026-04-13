package campaign

import (
	"github.com/fracturing-space/game/internal/command"
	"github.com/fracturing-space/game/internal/event"
)

// CreateCommandSpec is the typed campaign.create contract.
var CreateCommandSpec = command.NewCoreSpec(command.CoreSpecArgs[Create]{
	Message:   Create{},
	Scope:     command.ScopeNewCampaign,
	Normalize: normalizeCreate,
	Validate:  ValidateCreate,
})

// UpdateCommandSpec is the typed campaign.update contract.
var UpdateCommandSpec = command.NewCoreSpec(command.CoreSpecArgs[Update]{
	Message:   Update{},
	Scope:     command.ScopeCampaign,
	Normalize: normalizeUpdate,
	Validate:  ValidateUpdate,
})

// AIBindCommandSpec is the typed campaign.ai_bind contract.
var AIBindCommandSpec = command.NewCoreSpec(command.CoreSpecArgs[AIBind]{
	Message:   AIBind{},
	Scope:     command.ScopeCampaign,
	Normalize: normalizeAIBind,
	Validate:  ValidateAIBind,
})

// AIUnbindCommandSpec is the typed campaign.ai_unbind contract.
var AIUnbindCommandSpec = command.NewCoreSpec(command.CoreSpecArgs[AIUnbind]{
	Message: AIUnbind{},
	Scope:   command.ScopeCampaign,
})

// PlayBeginCommandSpec is the typed campaign.play.begin contract.
var PlayBeginCommandSpec = command.NewCoreSpec(command.CoreSpecArgs[PlayBegin]{
	Message: PlayBegin{},
	Scope:   command.ScopeCampaign,
})

// PlayEndCommandSpec is the typed campaign.play.end contract.
var PlayEndCommandSpec = command.NewCoreSpec(command.CoreSpecArgs[PlayEnd]{
	Message: PlayEnd{},
	Scope:   command.ScopeCampaign,
})

// PlayPauseCommandSpec is the typed campaign.play.pause contract.
var PlayPauseCommandSpec = command.NewCoreSpec(command.CoreSpecArgs[PlayPause]{
	Message:   PlayPause{},
	Scope:     command.ScopeCampaign,
	Normalize: normalizePlayPause,
})

// PlayResumeCommandSpec is the typed campaign.play.resume contract.
var PlayResumeCommandSpec = command.NewCoreSpec(command.CoreSpecArgs[PlayResume]{
	Message:   PlayResume{},
	Scope:     command.ScopeCampaign,
	Normalize: normalizePlayResume,
})

// CreatedEventSpec is the typed campaign.created contract.
var CreatedEventSpec = event.NewCoreSpec(Created{}, normalizeCreated, ValidateCreated)

// UpdatedEventSpec is the typed campaign.updated contract.
var UpdatedEventSpec = event.NewCoreSpec(Updated{}, normalizeUpdated, ValidateUpdated)

// AIBoundEventSpec is the typed campaign.ai_bound contract.
var AIBoundEventSpec = event.NewCoreSpec(AIBound{}, normalizeAIBound, ValidateAIBound)

// AIUnboundEventSpec is the typed campaign.ai_unbound contract.
var AIUnboundEventSpec = event.NewCoreSpec(AIUnbound{}, event.Identity[AIUnbound], nil)

// PlayBeganEventSpec is the typed campaign.play.began contract.
var PlayBeganEventSpec = event.NewCoreSpec(PlayBegan{}, normalizePlayBegan, ValidatePlayBegan)

// PlayPausedEventSpec is the typed campaign.play.paused contract.
var PlayPausedEventSpec = event.NewCoreSpec(PlayPaused{}, normalizePlayPaused, ValidatePlayPaused)

// PlayResumedEventSpec is the typed campaign.play.resumed contract.
var PlayResumedEventSpec = event.NewCoreSpec(PlayResumed{}, normalizePlayResumed, ValidatePlayResumed)

// PlayEndedEventSpec is the typed campaign.play.ended contract.
var PlayEndedEventSpec = event.NewCoreSpec(PlayEnded{}, normalizePlayEnded, ValidatePlayEnded)
