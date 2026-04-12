package session

import (
	"github.com/fracturing-space/game/internal/campaign"
	"github.com/fracturing-space/game/internal/modules/util"
	"github.com/fracturing-space/game/internal/session"
)

func effectiveAssignments(state campaign.State, overrides []session.CharacterControllerAssignment) ([]session.CharacterControllerAssignment, error) {
	return util.EffectiveAssignments(state, overrides)
}
