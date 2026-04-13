package readiness

import (
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/fracturing-space/game/internal/campaign"
	"github.com/fracturing-space/game/internal/participant"
)

// EvaluatePlay reports every deterministic blocker preventing a campaign from entering PLAY.
func EvaluatePlay(state campaign.State) Report {
	index := participantIndex(state)
	characterCounts := activeCharacterCountsByParticipant(state)
	blockers := make([]Blocker, 0, 2+len(index.boundParticipants))

	if !campaign.HasBoundAIAgent(state) {
		blockers = append(blockers, newActionableBlocker(
			RejectionCodePlayReadinessAIAgentRequired,
			"campaign play readiness requires ai agent binding",
			nil,
			aiAgentRequiredAction(index),
		))
	}
	if len(index.unboundParticipantIDs) != 0 {
		blockers = append(blockers, newActionableBlocker(
			RejectionCodePlayReadinessPlayerRequired,
			"campaign play readiness requires every active participant to have a bound subject id",
			nil,
			ownerManageParticipantsAction(index),
		))
	}
	for _, player := range index.boundParticipants {
		if characterCounts[player.ID] > 0 {
			continue
		}
		blockers = append(blockers, newActionableBlocker(
			RejectionCodePlayReadinessPlayerCharacterRequired,
			fmt.Sprintf("campaign play readiness requires player participant %s to have at least one active character", player.displayName()),
			map[string]string{
				"participant_id":   player.ID,
				"participant_name": player.Name,
			},
			createCharacterAction(player.ID),
		))
	}

	return Report{Blockers: blockers}
}

// EvaluatePlayTransition reports the first blocker preventing a campaign from entering PLAY.
func EvaluatePlayTransition(state campaign.State) *Rejection {
	report := EvaluatePlay(state)
	if report.Ready() {
		return nil
	}
	first := report.Blockers[0]
	return &Rejection{
		Code:    first.Code,
		Message: first.Message,
	}
}

// AsRejection reports whether err carries a play-readiness rejection.
func AsRejection(err error) (*Rejection, bool) {
	var rejection *Rejection
	if !errors.As(err, &rejection) {
		return nil, false
	}
	return rejection, true
}

func participantIndex(state campaign.State) indexedParticipants {
	index := indexedParticipants{}
	for _, record := range state.Participants {
		if !record.Active {
			continue
		}
		if record.Access == participant.AccessOwner {
			index.ownerIDs = append(index.ownerIDs, record.ID)
		}
		if record.SubjectID == "" {
			index.unboundParticipantIDs = append(index.unboundParticipantIDs, record.ID)
			continue
		}
		index.boundParticipants = append(index.boundParticipants, playerRecord{
			ID:   record.ID,
			Name: record.Name,
		})
	}
	index.ownerIDs = normalizeIDs(index.ownerIDs)
	index.unboundParticipantIDs = normalizeIDs(index.unboundParticipantIDs)
	slices.SortFunc(index.boundParticipants, func(a, b playerRecord) int {
		return strings.Compare(a.ID, b.ID)
	})
	return index
}

type indexedParticipants struct {
	ownerIDs              []string
	unboundParticipantIDs []string
	boundParticipants     []playerRecord
}

type playerRecord struct {
	ID   string
	Name string
}

func (p playerRecord) displayName() string {
	if p.Name != "" {
		return p.Name
	}
	return p.ID
}

func newBlocker(code, message string, metadata map[string]string) Blocker {
	cloned := make(map[string]string, len(metadata))
	maps.Copy(cloned, metadata)
	return Blocker{
		Code:     strings.TrimSpace(code),
		Message:  strings.TrimSpace(message),
		Metadata: cloned,
		Action:   Action{},
	}
}

func newActionableBlocker(code, message string, metadata map[string]string, action Action) Blocker {
	blocker := newBlocker(code, message, metadata)
	blocker.Action = cloneAction(action)
	return blocker
}

func aiAgentRequiredAction(index indexedParticipants) Action {
	return Action{
		ResponsibleParticipantIDs: append([]string{}, index.ownerIDs...),
		ResolutionKind:            ResolutionKindConfigureAIAgent,
	}
}

func ownerManageParticipantsAction(index indexedParticipants) Action {
	return Action{
		ResponsibleParticipantIDs: append([]string{}, index.ownerIDs...),
		ResolutionKind:            ResolutionKindManageParticipants,
	}
}

func createCharacterAction(participantID string) Action {
	return Action{
		ResponsibleParticipantIDs: []string{participantID},
		ResolutionKind:            ResolutionKindCreateCharacter,
		TargetParticipantID:       participantID,
	}
}

func cloneAction(input Action) Action {
	return Action{
		ResponsibleParticipantIDs: normalizeIDs(append([]string{}, input.ResponsibleParticipantIDs...)),
		ResolutionKind:            input.ResolutionKind,
		TargetParticipantID:       input.TargetParticipantID,
	}
}

func normalizeIDs(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	slices.Sort(result)
	if len(result) == 0 {
		return nil
	}
	return result
}

func activeCharacterCountsByParticipant(state campaign.State) map[string]int {
	if len(state.Characters) == 0 {
		return nil
	}
	counts := make(map[string]int, len(state.Characters))
	for _, record := range state.Characters {
		if !record.Active {
			continue
		}
		participantID := record.ParticipantID
		if participantID == "" {
			continue
		}
		counts[participantID]++
	}
	return counts
}
