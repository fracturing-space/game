package service

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/fracturing-space/game/internal/caller"
	"github.com/fracturing-space/game/internal/campaign"
	"github.com/fracturing-space/game/internal/character"
	"github.com/fracturing-space/game/internal/command"
	"github.com/fracturing-space/game/internal/event"
	"github.com/fracturing-space/game/internal/participant"
	"github.com/fracturing-space/game/internal/scene"
	"github.com/fracturing-space/game/internal/session"
)

func TestLoadCampaignLockedUsesHotProjectionCache(t *testing.T) {
	t.Parallel()

	foldCalls := 0
	svc := newStubService(t, stubServiceModule{
		fold: func(state *campaign.State, envelope event.Envelope) error {
			foldCalls++
			return defaultStubFold(state, envelope)
		},
	})
	if _, err := svc.store.AppendCommits(context.Background(), "camp-1", []PreparedCommit{{
		Events: []event.Envelope{mustTestEventEnvelope(t, "camp-1", "seed")},
	}}, func() time.Time {
		return serviceTestClockTime
	}); err != nil {
		t.Fatalf("AppendCommits(seed) error = %v", err)
	}

	if _, _, err := svc.loadCampaignLocked(context.Background(), "camp-1"); err != nil {
		t.Fatalf("loadCampaignLocked(first) error = %v", err)
	}
	if got, want := foldCalls, 1; got != want {
		t.Fatalf("fold calls after first load = %d, want %d", got, want)
	}
	if _, _, err := svc.loadCampaignLocked(context.Background(), "camp-1"); err != nil {
		t.Fatalf("loadCampaignLocked(second) error = %v", err)
	}
	if got, want := foldCalls, 1; got != want {
		t.Fatalf("fold calls after hot-cache load = %d, want %d", got, want)
	}

	if _, err := svc.store.AppendCommits(context.Background(), "camp-1", []PreparedCommit{{
		Events: []event.Envelope{mustTestEventEnvelope(t, "camp-1", "next")},
	}}, func() time.Time {
		return serviceTestClockTime.Add(time.Minute)
	}); err != nil {
		t.Fatalf("AppendCommits(next) error = %v", err)
	}
	if _, _, err := svc.loadCampaignLocked(context.Background(), "camp-1"); err != nil {
		t.Fatalf("loadCampaignLocked(after append) error = %v", err)
	}
	if got, want := foldCalls, 3; got != want {
		t.Fatalf("fold calls after stale-cache repair = %d, want %d", got, want)
	}
}

func TestReadResourceCampaignSurfaces(t *testing.T) {
	t.Parallel()

	fixture := newActiveSessionCampaignFixture(t)

	artifacts := fixture.Service.artifacts.(*testArtifactStore)
	if err := artifacts.PutArtifact(context.Background(), Artifact{
		CampaignID: fixture.CampaignID,
		Path:       "story.md",
		Content:    "# Harbor\nThe bells toll.",
		UpdatedAt:  fixedRecordTime,
	}); err != nil {
		t.Fatalf("PutArtifact() error = %v", err)
	}

	tests := []struct {
		uri      string
		contains string
	}{
		{uri: "context://current", contains: `"subject_id": "subject-1"`},
		{uri: "campaign://" + fixture.CampaignID, contains: `"name": "Autumn Twilight"`},
		{uri: "campaign://" + fixture.CampaignID + "/participants", contains: `"participants"`},
		{uri: "campaign://" + fixture.CampaignID + "/characters", contains: `"luna"`},
		{uri: "campaign://" + fixture.CampaignID + "/sessions", contains: `"ACTIVE"`},
		{uri: "campaign://" + fixture.CampaignID + "/interaction", contains: `"active_session"`},
		{uri: "campaign://" + fixture.CampaignID + "/characters/char-1/sheet", contains: `"character"`},
		{uri: "campaign://" + fixture.CampaignID + "/sessions/sess-1/scenes", contains: `"scenes"`},
		{uri: "campaign://" + fixture.CampaignID + "/artifacts/story.md", contains: "# Harbor"},
	}

	for _, tc := range tests {
		got, err := fixture.Service.ReadResource(context.Background(), fixture.OwnerCaller, tc.uri)
		if err != nil {
			t.Fatalf("ReadResource(%s) error = %v", tc.uri, err)
		}
		if !strings.Contains(got, tc.contains) {
			t.Fatalf("ReadResource(%s) = %q, want substring %q", tc.uri, got, tc.contains)
		}
	}
	for _, uri := range []string{
		"campaign://" + fixture.CampaignID + "/characters/char-1/equipment",
		"campaign://" + fixture.CampaignID + "/characters/char-1/resources",
	} {
		if _, err := fixture.Service.ReadResource(context.Background(), fixture.OwnerCaller, uri); err == nil {
			t.Fatalf("ReadResource(%s) error = nil, want unsupported resource failure", uri)
		}
	}
}

func TestListCampaignsUsesProjectionSubjectIndex(t *testing.T) {
	t.Parallel()

	svc := newStubService(t, stubServiceModule{})
	projections := svc.projections.(*testProjectionStore)

	for i := range 12 {
		state := campaign.NewState()
		state.Exists = true
		state.CampaignID = "camp-" + mustIDSuffix(i+1)
		state.Name = "Campaign " + mustIDSuffix(i+1)
		ownerID := "owner-" + mustIDSuffix(i+1)
		state.Participants[ownerID] = participant.Record{
			ID:   "owner-" + mustIDSuffix(i+1),
			Name: "Owner", Access: participant.AccessOwner, SubjectID: "subject-1",
			Active: true,
		}
		if i == 0 {
			state.Participants["other"] = participant.Record{
				ID:   "other",
				Name: "Other", Access: participant.AccessMember, SubjectID: "subject-2",
				Active: true,
			}
		}
		if i%2 == 0 {
			state.AIAgentID = "agent-1"
		}
		if i%3 == 0 {
			state.Sessions["sess-1"] = session.Record{ID: "sess-1", Name: "Session 1", Status: session.StatusActive}
			state.ActiveSessionID = "sess-1"
			state.PlayState = campaign.PlayStateActive
		} else {
			state.PlayState = campaign.PlayStateSetup
		}
		if i == 11 {
			state.AIAgentID = "agent-1"
			state.Characters["char-1"] = character.Record{
				ID:            "char-1",
				ParticipantID: ownerID,
				Name:          "Luna", Active: true,
			}
			state.Sessions["sess-1"] = session.Record{ID: "sess-1", Name: "Session 1", Status: session.StatusActive}
			state.ActiveSessionID = "sess-1"
			state.ActiveSceneID = "scene-1"
			state.Scenes["scene-1"] = scene.Record{ID: "scene-1", SessionID: "sess-1", Name: "Opening Scene", Active: true}
			state.PlayState = campaign.PlayStateSetup
		}
		if err := projections.SaveProjection(context.Background(), ProjectionSnapshot{
			CampaignID:     state.CampaignID,
			HeadSeq:        uint64(i + 1),
			State:          state,
			UpdatedAt:      fixedRecordTime.Add(time.Duration(i) * time.Minute),
			LastActivityAt: fixedRecordTime.Add(time.Duration(i) * time.Minute),
		}); err != nil {
			t.Fatalf("SaveProjection(%d) error = %v", i, err)
		}
	}

	items, err := svc.ListCampaigns(context.Background(), caller.MustNewSubject("subject-1"))
	if err != nil {
		t.Fatalf("ListCampaigns() error = %v", err)
	}
	if got, want := len(items), listCampaignsLimit; got != want {
		t.Fatalf("ListCampaigns() len = %d, want %d", got, want)
	}
	if got, want := items[0].CampaignID, "camp-12"; got != want {
		t.Fatalf("ListCampaigns()[0].CampaignID = %q, want %q", got, want)
	}
	if got, want := items[len(items)-1].CampaignID, "camp-3"; got != want {
		t.Fatalf("ListCampaigns()[last].CampaignID = %q, want %q", got, want)
	}
	if !items[0].ReadyToPlay {
		t.Fatal("ListCampaigns()[0].ReadyToPlay = false, want true")
	}
	if got, want := items[0].HasAIBinding, true; got != want {
		t.Fatalf("ListCampaigns()[0].HasAIBinding = %t, want %t", got, want)
	}
	if got, want := items[0].HasActiveSession, true; got != want {
		t.Fatalf("ListCampaigns()[0].HasActiveSession = %t, want %t", got, want)
	}

	otherItems, err := svc.ListCampaigns(context.Background(), caller.MustNewSubject("subject-2"))
	if err != nil {
		t.Fatalf("ListCampaigns(subject-2) error = %v", err)
	}
	if got, want := len(otherItems), 1; got != want {
		t.Fatalf("ListCampaigns(subject-2) len = %d, want %d", got, want)
	}
	if got, want := otherItems[0].CampaignID, "camp-1"; got != want {
		t.Fatalf("ListCampaigns(subject-2)[0].CampaignID = %q, want %q", got, want)
	}

	emptyItems, err := svc.ListCampaigns(context.Background(), caller.MustNewSubject("subject-missing"))
	if err != nil {
		t.Fatalf("ListCampaigns(subject-missing) error = %v", err)
	}
	if len(emptyItems) != 0 {
		t.Fatalf("ListCampaigns(subject-missing) len = %d, want 0", len(emptyItems))
	}
}

func TestListCampaignsErrors(t *testing.T) {
	t.Parallel()

	svc := newStubService(t, stubServiceModule{})
	if _, err := svc.ListCampaigns(context.Background(), caller.Caller{}); err == nil {
		t.Fatal("ListCampaigns(unauthenticated) error = nil, want failure")
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := svc.ListCampaigns(ctx, caller.MustNewSubject("subject-1")); !errors.Is(err, context.Canceled) {
		t.Fatalf("ListCampaigns(canceled) error = %v, want context canceled", err)
	}

	svc.projections = failingProjectionStore{
		base:    svc.projections,
		listErr: errors.New("list failed"),
	}
	if _, err := svc.ListCampaigns(context.Background(), caller.MustNewSubject("subject-1")); err == nil || !strings.Contains(err.Error(), "list failed") {
		t.Fatalf("ListCampaigns(list failure) error = %v, want list failure", err)
	}
}

func TestSubscribeEventsReplaysAndTails(t *testing.T) {
	t.Parallel()

	fixture := newCreatedCampaignFixture(t)
	stream, err := fixture.Service.SubscribeEvents(context.Background(), fixture.OwnerCaller, fixture.CampaignID, 0)
	if err != nil {
		t.Fatalf("SubscribeEvents() error = %v", err)
	}
	defer stream.Close()

	for i := range 2 {
		select {
		case record := <-stream.Records:
			if record.Seq != uint64(i+1) {
				t.Fatalf("initial record seq = %d, want %d", record.Seq, i+1)
			}
		case <-time.After(time.Second):
			t.Fatal("timed out waiting for replayed event")
		}
	}

	if _, err := fixture.Service.CommitCommand(context.Background(), fixture.OwnerCaller, command.Envelope{
		CampaignID: fixture.CampaignID,
		Message:    campaign.AIBind{AIAgentID: "agent-7"},
	}); err != nil {
		t.Fatalf("CommitCommand(ai bind) error = %v", err)
	}

	select {
	case record := <-stream.Records:
		if got, want := record.Seq, uint64(3); got != want {
			t.Fatalf("tailed record seq = %d, want %d", got, want)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for tailed event")
	}
}

func mustIDSuffix(value int) string {
	return strconv.Itoa(value)
}
