package service

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/fracturing-space/game/internal/authz"
	"github.com/fracturing-space/game/internal/caller"
	"github.com/fracturing-space/game/internal/campaign"
	"github.com/fracturing-space/game/internal/canonical"
	"github.com/fracturing-space/game/internal/errs"
)

type campaignResourceRequest struct {
	ctx        context.Context
	uri        string
	campaignID string
	parts      []string
	state      campaign.State
	snapshot   campaign.Snapshot
}

type resourceHandler func(campaignResourceRequest) (any, error)

type rawResource string

type contextResource struct {
	Context contextSubjectResource `json:"context"`
}

type contextSubjectResource struct {
	SubjectID string `json:"subject_id"`
}

// ReadResource returns one authorized campaign resource as JSON or document
// content.
func (s *Service) ReadResource(ctx context.Context, act caller.Caller, uri string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}

	if canonical.IsBlank(uri) {
		return "", errs.InvalidArgumentf("resource uri is required")
	}
	if !canonical.IsExact(uri) {
		return "", errs.InvalidArgumentf("resource uri must not contain surrounding whitespace")
	}
	if uri == "context://current" {
		return marshalResource(contextResource{
			Context: contextSubjectResource{SubjectID: act.SubjectID},
		})
	}

	campaignID, parts, err := parseCampaignResourceURI(uri)
	if err != nil {
		return "", err
	}
	campaignID, err = normalizeCampaignID(campaignID)
	if err != nil {
		return "", err
	}
	snapshot, err := s.publishedCampaignSnapshot(ctx, campaignID)
	if err != nil {
		return "", err
	}
	state := snapshot.state.Clone()
	if err := authz.RequireReadCampaign(act, state); err != nil {
		return "", err
	}

	request := campaignResourceRequest{
		ctx:        ctx,
		uri:        uri,
		campaignID: campaignID,
		parts:      parts,
		state:      state,
		snapshot:   campaign.SnapshotOf(state),
	}
	handler, ok := s.resourceHandlers()[resourcePrefix(parts)]
	if !ok {
		return "", errs.InvalidArgumentf("unknown resource uri: %s", uri)
	}

	value, err := handler(request)
	if err != nil {
		return "", err
	}
	if text, ok := value.(rawResource); ok {
		return string(text), nil
	}
	return marshalResource(value)
}

func parseCampaignResourceURI(uri string) (string, []string, error) {
	if !strings.HasPrefix(uri, "campaign://") {
		return "", nil, errs.InvalidArgumentf("unknown resource uri: %s", uri)
	}

	path := strings.TrimPrefix(uri, "campaign://")
	parts := strings.Split(path, "/")
	if len(parts) == 0 || canonical.IsBlank(parts[0]) {
		return "", nil, errs.InvalidArgumentf("campaign id is required in resource uri")
	}
	for _, part := range parts {
		if !canonical.IsExact(part) {
			return "", nil, errs.InvalidArgumentf("resource uri segments must not contain surrounding whitespace")
		}
	}
	return parts[0], parts[1:], nil
}

func resourcePrefix(parts []string) string {
	if len(parts) == 0 {
		return ""
	}
	return parts[0]
}

func (s *Service) resourceHandlers() map[string]resourceHandler {
	return map[string]resourceHandler{
		"":             s.readCampaignResource,
		"participants": s.readParticipantsResource,
		"characters":   s.readCharactersResource,
		"sessions":     s.readSessionsResource,
		"interaction":  s.readInteractionResource,
		"artifacts":    s.readArtifactsResource,
	}
}

func marshalResource(value any) (string, error) {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}
