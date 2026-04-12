package service

import (
	"context"
	"errors"
	"maps"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/fracturing-space/game/internal/caller"
	"github.com/fracturing-space/game/internal/campaign"
	"github.com/fracturing-space/game/internal/command"
	"github.com/fracturing-space/game/internal/errs"
	"github.com/fracturing-space/game/internal/event"
	"github.com/fracturing-space/game/internal/scene"
	"github.com/fracturing-space/game/internal/session"
)

func TestRuntimeHelpersAdditionalCoverage(t *testing.T) {
	t.Parallel()

	if got := (*campaignSlotRegistry)(nil).Slot("camp-1"); got == nil {
		t.Fatal("nil registry Slot() = nil, want slot")
	}
	if got := (*campaignSlotRegistry)(nil).Acquire("camp-1", fixedRecordTime, time.Minute); got == nil {
		t.Fatal("nil registry Acquire() = nil, want slot")
	}
	if got := (*Service)(nil).campaignSlot("camp-1"); got == nil {
		t.Fatal("nil service campaignSlot() = nil, want slot")
	}
	if got := (&Service{}).campaignSlot("camp-1"); got == nil {
		t.Fatal("service without registry campaignSlot() = nil, want slot")
	}
	if got, release := (*Service)(nil).acquireCampaignSlot("camp-1"); got == nil {
		t.Fatal("nil service acquireCampaignSlot() = nil, want slot")
	} else {
		release()
	}

	var nilSlot *campaignSlot
	if _, ok := nilSlot.loadRuntime(1, fixedRecordTime, time.Minute); ok {
		t.Fatal("nil slot loadRuntime() ok = true, want false")
	}
	nilSlot.storeRuntime(1, campaign.NewState(), fixedRecordTime)
	if _, ok := nilSlot.loadPublished(); ok {
		t.Fatal("nil slot loadPublished() ok = true, want false")
	}
	nilSlot.storePublished(1, campaign.NewState(), fixedRecordTime)

	registry := newCampaignSlotRegistry()
	slot := registry.Slot("camp-1")
	if got := registry.Slot("camp-1"); got != slot {
		t.Fatal("Slot(canonical) should return same slot")
	}
	acquired := registry.Acquire("camp-1", fixedRecordTime, time.Minute)
	if acquired != slot {
		t.Fatal("Acquire(canonical) should return same slot")
	}
	registry.Release("camp-1", acquired, fixedRecordTime, time.Minute)

	state := campaign.NewState()
	state.Exists = true
	state.CampaignID = "camp-1"
	state.Name = "stored"

	slot.storeRuntime(1, state, fixedRecordTime)
	state.Name = "mutated"
	if _, ok := slot.loadRuntime(2, fixedRecordTime, time.Minute); ok {
		t.Fatal("loadRuntime(head mismatch) ok = true, want false")
	}
	runtimeState, ok := slot.loadRuntime(1, fixedRecordTime, time.Minute)
	if !ok {
		t.Fatal("loadRuntime() ok = false, want true")
	}
	runtimeState.Name = "changed after load"
	runtimeAgain, ok := slot.loadRuntime(1, fixedRecordTime, time.Minute)
	if !ok {
		t.Fatal("loadRuntime(second) ok = false, want true")
	}
	if got, want := runtimeAgain.Name, "stored"; got != want {
		t.Fatalf("runtime clone name = %q, want %q", got, want)
	}

	slot.storePublished(1, state, fixedRecordTime)
	published, ok := slot.loadPublished()
	if !ok {
		t.Fatal("loadPublished() ok = false, want true")
	}
	published.state.Name = "changed after load"
	publishedAgain, ok := slot.loadPublished()
	if !ok {
		t.Fatal("loadPublished(second) ok = false, want true")
	}
	if got, want := publishedAgain.state.Name, "mutated"; got != want {
		t.Fatalf("published clone name = %q, want %q", got, want)
	}
}

func TestPlannerAndReadBranches(t *testing.T) {
	t.Parallel()

	t.Run("commit command respects canceled context", func(t *testing.T) {
		svc := newStubService(t, stubServiceModule{})
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		if _, err := svc.CommitCommand(ctx, defaultCaller(), command.Envelope{Message: testNewCommand{Name: "next"}}); !errors.Is(err, context.Canceled) {
			t.Fatalf("CommitCommand(canceled) error = %v, want context canceled", err)
		}
	})

	t.Run("commit command rejects invalid commands", func(t *testing.T) {
		svc := newTestService(t)
		if _, err := svc.CommitCommand(context.Background(), defaultCaller(), command.Envelope{
			Message: campaign.Create{Name: "   ", OwnerName: "louis"},
		}); err == nil {
			t.Fatal("CommitCommand(invalid create) error = nil, want failure")
		}
	})

	t.Run("plan current locked respects canceled context", func(t *testing.T) {
		svc := newStubService(t, stubServiceModule{})
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		if _, err := svc.planCurrentLocked(ctx, defaultCaller(), command.Envelope{
			Message: testNewCommand{Name: "next"},
		}, svc.ids.Session(false)); !errors.Is(err, context.Canceled) {
			t.Fatalf("planCurrentLocked(canceled) error = %v, want context canceled", err)
		}
	})

	t.Run("plan current locked handles new campaign commands", func(t *testing.T) {
		svc := newStubService(t, stubServiceModule{})
		plan, err := svc.planCurrentLocked(context.Background(), defaultCaller(), command.Envelope{
			Message: testNewCommand{Name: "next"},
		}, svc.ids.Session(false))
		if err != nil {
			t.Fatalf("planCurrentLocked(new) error = %v", err)
		}
		if plan.campaignID == "" {
			t.Fatal("planCurrentLocked(new) campaign id = empty, want id")
		}
		if got, want := plan.finalState.Name, "next"; got != want {
			t.Fatalf("planCurrentLocked(new) final state name = %q, want %q", got, want)
		}
	})

	t.Run("plan current locked loads existing campaigns", func(t *testing.T) {
		fixture := newGMEnabledStubCampaignFixture(t, stubServiceModule{})
		plan, err := fixture.Service.planCurrentLocked(context.Background(), defaultCaller(), command.Envelope{
			CampaignID: fixture.CampaignID,
			Message:    testCampaignCommand{Name: "next"},
		}, fixture.Service.ids.Session(false))
		if err != nil {
			t.Fatalf("planCurrentLocked(existing) error = %v", err)
		}
		if got, want := plan.campaignID, fixture.CampaignID; got != want {
			t.Fatalf("planCurrentLocked(existing) campaign id = %q, want %q", got, want)
		}
		if got, want := plan.finalState.Name, "next"; got != want {
			t.Fatalf("planCurrentLocked(existing) final state name = %q, want %q", got, want)
		}
	})

	t.Run("plan current validated requires admission catalog", func(t *testing.T) {
		svc := newStubService(t, stubServiceModule{})
		svc.admission = nil

		validated, _, err := svc.commands.Validate(command.Envelope{Message: testNewCommand{Name: "next"}})
		if err != nil {
			t.Fatalf("Validate() error = %v", err)
		}
		if _, err := svc.planCurrentValidatedInSlot(context.Background(), nil, defaultCaller(), validated, svc.ids.Session(false)); err == nil || !strings.Contains(err.Error(), "admission rule is not registered") {
			t.Fatalf("planCurrentValidatedInSlot() error = %v, want missing admission rule", err)
		}
	})

	t.Run("plan current validated loads existing campaign state", func(t *testing.T) {
		fixture := newGMEnabledStubCampaignFixture(t, stubServiceModule{})
		validated, _, err := fixture.Service.commands.Validate(command.Envelope{
			CampaignID: fixture.CampaignID,
			Message:    testCampaignCommand{Name: "next"},
		})
		if err != nil {
			t.Fatalf("Validate() error = %v", err)
		}
		plan, err := fixture.Service.planCurrentValidatedInSlot(context.Background(), fixture.Service.campaignSlot(fixture.CampaignID), defaultCaller(), validated, fixture.Service.ids.Session(false))
		if err != nil {
			t.Fatalf("planCurrentValidatedInSlot() error = %v", err)
		}
		if got, want := plan.campaignID, fixture.CampaignID; got != want {
			t.Fatalf("campaign id = %q, want %q", got, want)
		}
	})

	t.Run("plan validated rejects empty event batches", func(t *testing.T) {
		svc := newStubService(t, stubServiceModule{
			decide: func(campaign.State, caller.Caller, command.Envelope, func(string) (string, error)) ([]event.Envelope, error) {
				return nil, nil
			},
		})
		validated, _, err := svc.commands.Validate(command.Envelope{
			CampaignID: "camp-1",
			Message:    testCampaignCommand{Name: "next"},
		})
		if err != nil {
			t.Fatalf("Validate() error = %v", err)
		}
		state := campaign.NewState()
		state.Exists = true
		state.CampaignID = "camp-1"
		if _, _, _, err := svc.planValidatedLocked(context.Background(), defaultCaller(), validated, state, svc.ids.Session(false)); err == nil || !strings.Contains(err.Error(), "accepted command must emit at least one event") {
			t.Fatalf("planValidatedLocked(empty) error = %v, want empty event failure", err)
		}
	})

	t.Run("plan validated rejects invalid and mixed events", func(t *testing.T) {
		state := campaign.NewState()
		state.Exists = true
		state.CampaignID = "camp-1"

		t.Run("invalid event", func(t *testing.T) {
			svc := newStubService(t, stubServiceModule{
				decide: func(campaign.State, caller.Caller, command.Envelope, func(string) (string, error)) ([]event.Envelope, error) {
					return []event.Envelope{{CampaignID: "camp-1", Message: unknownTestEvent{}}}, nil
				},
			})
			validated, _, err := svc.commands.Validate(command.Envelope{
				CampaignID: "camp-1",
				Message:    testCampaignCommand{Name: "next"},
			})
			if err != nil {
				t.Fatalf("Validate() error = %v", err)
			}
			if _, _, _, err := svc.planValidatedLocked(context.Background(), defaultCaller(), validated, state, svc.ids.Session(false)); err == nil {
				t.Fatal("planValidatedLocked(invalid event) error = nil, want failure")
			}
		})

		t.Run("mixed campaigns", func(t *testing.T) {
			svc := newStubService(t, stubServiceModule{
				decide: func(campaign.State, caller.Caller, command.Envelope, func(string) (string, error)) ([]event.Envelope, error) {
					return []event.Envelope{
						mustTestEventEnvelope(t, "camp-1", "first"),
						mustTestEventEnvelope(t, "camp-2", "second"),
					}, nil
				},
			})
			validated, _, err := svc.commands.Validate(command.Envelope{
				CampaignID: "camp-1",
				Message:    testCampaignCommand{Name: "next"},
			})
			if err != nil {
				t.Fatalf("Validate() error = %v", err)
			}
			if _, _, _, err := svc.planValidatedLocked(context.Background(), defaultCaller(), validated, state, svc.ids.Session(false)); err == nil || !strings.Contains(err.Error(), "single campaign") {
				t.Fatalf("planValidatedLocked(mixed campaigns) error = %v, want mismatch failure", err)
			}
		})
	})

	t.Run("plan validated reports fold failures", func(t *testing.T) {
		svc := newStubService(t, stubServiceModule{
			fold: func(*campaign.State, event.Envelope) error {
				return errors.New("fold failed")
			},
		})
		validated, _, err := svc.commands.Validate(command.Envelope{
			CampaignID: "camp-1",
			Message:    testCampaignCommand{Name: "next"},
		})
		if err != nil {
			t.Fatalf("Validate() error = %v", err)
		}
		state := campaign.NewState()
		state.Exists = true
		state.CampaignID = "camp-1"
		if _, _, _, err := svc.planValidatedLocked(context.Background(), defaultCaller(), validated, state, svc.ids.Session(false)); err == nil || !strings.Contains(err.Error(), "fold failed") {
			t.Fatalf("planValidatedLocked(fold failure) error = %v, want fold failure", err)
		}
	})

	t.Run("read functions enforce campaign access", func(t *testing.T) {
		fixture := newCreatedCampaignFixture(t)
		outsider := caller.MustNewSubject("subject-outsider")

		if _, err := fixture.Service.Inspect(context.Background(), outsider, fixture.CampaignID); err == nil {
			t.Fatal("Inspect(unbound caller) error = nil, want failure")
		}
		if _, err := fixture.Service.GetPlayReadiness(context.Background(), outsider, fixture.CampaignID); err == nil {
			t.Fatal("GetPlayReadiness(unbound caller) error = nil, want failure")
		}
		if _, err := fixture.Service.ListEvents(context.Background(), outsider, fixture.CampaignID, 0); err == nil {
			t.Fatal("ListEvents(unbound caller) error = nil, want failure")
		}
		if _, _, _, err := fixture.Service.readAuthorizedCampaignLocked(context.Background(), fixture.CampaignID, func(campaign.State) error {
			return errors.New("authorize failed")
		}); err == nil || !strings.Contains(err.Error(), "authorize failed") {
			t.Fatalf("readAuthorizedCampaignLocked() error = %v, want authorize failure", err)
		}
	})

	t.Run("read functions reject bad campaign ids and missing campaigns", func(t *testing.T) {
		svc := newTestService(t)
		if _, err := svc.Inspect(context.Background(), defaultCaller(), "   "); err == nil {
			t.Fatal("Inspect(blank id) error = nil, want failure")
		}
		if _, err := svc.GetPlayReadiness(context.Background(), defaultCaller(), "missing"); err == nil {
			t.Fatal("GetPlayReadiness(missing) error = nil, want failure")
		}
		if _, err := svc.ListEvents(context.Background(), defaultCaller(), "missing", 0); err == nil {
			t.Fatal("ListEvents(missing) error = nil, want failure")
		}
	})

	t.Run("play readiness can report ready", func(t *testing.T) {
		fixture := newPlayReadyCampaignFixture(t)
		report, err := fixture.Service.GetPlayReadiness(context.Background(), fixture.OwnerCaller, fixture.CampaignID)
		if err != nil {
			t.Fatalf("GetPlayReadiness() error = %v", err)
		}
		if !report.Ready() {
			t.Fatalf("GetPlayReadiness().blockers = %v, want ready report", report.Blockers)
		}
	})
}

func TestPlanAndIDHelpersAdditionalCoverage(t *testing.T) {
	t.Parallel()

	if store := (*Service)(nil).planStore(); store == nil || store.items == nil || store.byCampaign == nil {
		t.Fatal("nil service planStore() should allocate items")
	}

	svc := newTestService(t)
	svc.plans = nil
	if store := svc.planStore(); store == nil || store.items == nil || store.byCampaign == nil {
		t.Fatal("nil plans planStore() should allocate items")
	}

	if _, err := svc.ExecutePlan(context.Background(), defaultCaller(), "   "); !strings.Contains(err.Error(), "plan token is required") {
		t.Fatalf("ExecutePlan(blank token) error = %v, want blank token failure", err)
	}
	if _, err := svc.ExecutePlan(context.Background(), defaultCaller(), "plan-missing"); !errs.Is(err, errs.KindNotFound) {
		t.Fatalf("ExecutePlan(missing token) error = %v, want not found", err)
	}
	if _, err := (opaqueIDSession{}).NewID("   "); err == nil {
		t.Fatal("opaqueIDSession.NewID(blank) error = nil, want failure")
	}
	var session IDSession = opaqueIDSession{}
	session.Commit()

	var nilSvc *Service
	if err := nilSvc.storePlan(preparedPlan{token: "plan-1", campaignID: "camp-1"}, fixedRecordTime); err != nil {
		t.Fatalf("storePlan(nil service) error = %v", err)
	}
}

func TestExecuteAndSnapshotAdditionalCoverage(t *testing.T) {
	t.Parallel()

	t.Run("execute new campaign returns head-seq errors", func(t *testing.T) {
		svc := newTestService(t)
		svc.store = failingJournal{
			base:       svc.store,
			headSeqErr: errors.New("head seq failed"),
		}
		if _, err := svc.CommitCommand(context.Background(), defaultCaller(), command.Envelope{
			Message: campaign.Create{Name: "Autumn Twilight", OwnerName: "louis"},
		}); err == nil || !strings.Contains(err.Error(), "head seq failed") {
			t.Fatalf("CommitCommand(create) error = %v, want head seq failure", err)
		}
	})

	t.Run("execute new campaign rejects duplicate ids", func(t *testing.T) {
		svc := newTestService(t)
		first, err := svc.CommitCommand(context.Background(), defaultCaller(), command.Envelope{
			Message: campaign.Create{Name: "Autumn Twilight", OwnerName: "louis"},
		})
		if err != nil {
			t.Fatalf("CommitCommand(first create) error = %v", err)
		}

		svc.ids = fixedSequenceAllocator{ids: []string{first.State.ID, "part-3", "part-4"}}
		if _, err := svc.CommitCommand(context.Background(), defaultCaller(), command.Envelope{
			Message: campaign.Create{Name: "Winter Dawn", OwnerName: "louis"},
		}); !errs.Is(err, errs.KindAlreadyExists) {
			t.Fatalf("CommitCommand(duplicate create) error = %v, want already exists", err)
		}
	})

	t.Run("execute propagates append and projection failures", func(t *testing.T) {
		t.Run("create append failure", func(t *testing.T) {
			svc := newTestService(t)
			svc.store = failingJournal{
				base:      svc.store,
				appendErr: errors.New("append failed"),
			}
			if _, err := svc.CommitCommand(context.Background(), defaultCaller(), command.Envelope{
				Message: campaign.Create{Name: "Autumn Twilight", OwnerName: "louis"},
			}); err == nil || !strings.Contains(err.Error(), "append failed") {
				t.Fatalf("CommitCommand(create append failure) error = %v, want append failure", err)
			}
		})

		t.Run("update projection failure", func(t *testing.T) {
			fixture := newCreatedCampaignFixture(t)
			fixture.Service.projections = failingProjectionStore{
				base:              fixture.Service.projections,
				saveProjectionErr: errors.New("projection save failed"),
			}
			if _, err := fixture.Service.CommitCommand(context.Background(), fixture.OwnerCaller, command.Envelope{
				CampaignID: fixture.CampaignID,
				Message:    campaign.Update{Name: "Autumn Eclipse"},
			}); err == nil || !strings.Contains(err.Error(), "projection save failed") {
				t.Fatalf("CommitCommand(update projection failure) error = %v, want projection failure", err)
			}
		})
	})

	t.Run("published snapshot reuses cached snapshot", func(t *testing.T) {
		fixture := newCreatedCampaignFixture(t)
		first, err := fixture.Service.publishedCampaignSnapshot(context.Background(), fixture.CampaignID)
		if err != nil {
			t.Fatalf("publishedCampaignSnapshot(first) error = %v", err)
		}
		first.state.Name = "changed after load"

		second, err := fixture.Service.publishedCampaignSnapshot(context.Background(), fixture.CampaignID)
		if err != nil {
			t.Fatalf("publishedCampaignSnapshot(second) error = %v", err)
		}
		if got, want := second.state.Name, "Autumn Twilight"; got != want {
			t.Fatalf("published snapshot name = %q, want %q", got, want)
		}
	})

	t.Run("published snapshot rejects invalid and missing campaigns", func(t *testing.T) {
		svc := newTestService(t)
		if _, err := svc.publishedCampaignSnapshot(context.Background(), "   "); err == nil {
			t.Fatal("publishedCampaignSnapshot(blank) error = nil, want failure")
		}
		if _, err := svc.publishedCampaignSnapshot(context.Background(), "missing"); err == nil {
			t.Fatal("publishedCampaignSnapshot(missing) error = nil, want failure")
		}
	})

	t.Run("plan rejects invalid commands and missing rules", func(t *testing.T) {
		fixture := newActiveSessionCampaignFixture(t)
		if _, err := fixture.Service.PlanCommands(context.Background(), fixture.GMCaller, []command.Envelope{{
			CampaignID: fixture.CampaignID,
			Message: scene.Create{
				Name:         "   ",
				CharacterIDs: []string{fixture.OwnerCharacterID},
			},
		}}); err == nil {
			t.Fatal("PlanCommands(invalid command) error = nil, want failure")
		}

		fixture = newActiveSessionCampaignFixture(t)
		fixture.Service.admission = nil
		if _, err := fixture.Service.PlanCommands(context.Background(), fixture.GMCaller, []command.Envelope{{
			CampaignID: fixture.CampaignID,
			Message: scene.Create{
				Name:         "Autumn Eclipse",
				CharacterIDs: []string{fixture.OwnerCharacterID},
			},
		}}); err == nil || !strings.Contains(err.Error(), "planning rule is not registered") {
			t.Fatalf("PlanCommands(missing rule) error = %v, want missing rule failure", err)
		}
	})

	t.Run("read and plan functions respect canceled context", func(t *testing.T) {
		fixture := newActiveSessionCampaignFixture(t)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		if _, err := fixture.Service.Inspect(ctx, fixture.OwnerCaller, fixture.CampaignID); !errors.Is(err, context.Canceled) {
			t.Fatalf("Inspect(canceled) error = %v, want context canceled", err)
		}
		if _, err := fixture.Service.GetPlayReadiness(ctx, fixture.OwnerCaller, fixture.CampaignID); !errors.Is(err, context.Canceled) {
			t.Fatalf("GetPlayReadiness(canceled) error = %v, want context canceled", err)
		}
		if _, err := fixture.Service.ListEvents(ctx, fixture.OwnerCaller, fixture.CampaignID, 0); !errors.Is(err, context.Canceled) {
			t.Fatalf("ListEvents(canceled) error = %v, want context canceled", err)
		}
		if _, err := fixture.Service.PlanCommands(ctx, fixture.GMCaller, []command.Envelope{{
			CampaignID: fixture.CampaignID,
			Message: scene.Create{
				Name:         "Autumn Eclipse",
				CharacterIDs: []string{fixture.OwnerCharacterID},
			},
		}}); !errors.Is(err, context.Canceled) {
			t.Fatalf("PlanCommands(canceled) error = %v, want context canceled", err)
		}
		if _, err := fixture.Service.ExecutePlan(ctx, fixture.GMCaller, "plan-1"); !errors.Is(err, context.Canceled) {
			t.Fatalf("ExecutePlan(canceled) error = %v, want context canceled", err)
		}
	})

	t.Run("read authorized campaign in slot allows and rejects", func(t *testing.T) {
		fixture := newCreatedCampaignFixture(t)
		slot := fixture.Service.campaignSlot(fixture.CampaignID)
		slot.mu.Lock()
		defer slot.mu.Unlock()

		campaignID, timeline, state, err := fixture.Service.readAuthorizedCampaignInSlot(context.Background(), slot, fixture.CampaignID, nil)
		if err != nil {
			t.Fatalf("readAuthorizedCampaignInSlot() error = %v", err)
		}
		if got, want := campaignID, fixture.CampaignID; got != want {
			t.Fatalf("campaign id = %q, want %q", got, want)
		}
		if len(timeline) == 0 || state.CampaignID != fixture.CampaignID {
			t.Fatalf("readAuthorizedCampaignInSlot() = (%d records, %q state), want populated state", len(timeline), state.CampaignID)
		}
		if _, _, _, err := fixture.Service.readAuthorizedCampaignInSlot(context.Background(), slot, fixture.CampaignID, func(campaign.State) error {
			return errors.New("slot authorize failed")
		}); err == nil || !strings.Contains(err.Error(), "slot authorize failed") {
			t.Fatalf("readAuthorizedCampaignInSlot(authorize) error = %v, want authorize failure", err)
		}
		if _, _, _, err := fixture.Service.readAuthorizedCampaignInSlot(context.Background(), slot, "   ", nil); err == nil {
			t.Fatal("readAuthorizedCampaignInSlot(blank) error = nil, want failure")
		}
		if _, _, _, err := fixture.Service.readAuthorizedCampaignLocked(context.Background(), "   ", nil); err == nil {
			t.Fatal("readAuthorizedCampaignLocked(blank) error = nil, want failure")
		}
		if _, _, _, err := fixture.Service.readAuthorizedCampaignLocked(context.Background(), "missing", nil); err == nil {
			t.Fatal("readAuthorizedCampaignLocked(missing) error = nil, want failure")
		}
	})
}

func TestCloneEventMessageAdditionalCoverage(t *testing.T) {
	t.Parallel()

	sceneCreated := cloneEventMessage(scene.Created{
		SceneID:      "scene-1",
		SessionID:    "sess-1",
		Name:         "Opening",
		CharacterIDs: []string{"char-1"},
	}).(scene.Created)
	sceneCreated.CharacterIDs[0] = "changed"
	sceneCreatedAgain := cloneEventMessage(scene.Created{
		SceneID:      "scene-1",
		SessionID:    "sess-1",
		Name:         "Opening",
		CharacterIDs: []string{"char-1"},
	}).(scene.Created)
	if got, want := sceneCreatedAgain.CharacterIDs[0], "char-1"; got != want {
		t.Fatalf("scene created clone = %q, want %q", got, want)
	}

	replaced := cloneEventMessage(scene.CastReplaced{
		SceneID:      "scene-1",
		CharacterIDs: []string{"char-1"},
	}).(scene.CastReplaced)
	replaced.CharacterIDs[0] = "changed"
	replacedAgain := cloneEventMessage(scene.CastReplaced{
		SceneID:      "scene-1",
		CharacterIDs: []string{"char-1"},
	}).(scene.CastReplaced)
	if got, want := replacedAgain.CharacterIDs[0], "char-1"; got != want {
		t.Fatalf("scene cast clone = %q, want %q", got, want)
	}

	started := cloneEventMessage(session.Started{
		SessionID:            "sess-1",
		Name:                 "Session 1",
		CharacterControllers: []session.CharacterControllerAssignment{{CharacterID: "char-1", ParticipantID: "part-1"}},
	}).(session.Started)
	started.CharacterControllers[0].CharacterID = "changed"
	startedAgain := cloneEventMessage(session.Started{
		SessionID:            "sess-1",
		Name:                 "Session 1",
		CharacterControllers: []session.CharacterControllerAssignment{{CharacterID: "char-1", ParticipantID: "part-1"}},
	}).(session.Started)
	if got, want := startedAgain.CharacterControllers[0].CharacterID, "char-1"; got != want {
		t.Fatalf("session started clone = %q, want %q", got, want)
	}

	ended := cloneEventMessage(session.Ended{
		SessionID:            "sess-1",
		Name:                 "Session 1",
		CharacterControllers: []session.CharacterControllerAssignment{{CharacterID: "char-1", ParticipantID: "part-1"}},
	}).(session.Ended)
	ended.CharacterControllers[0].CharacterID = "changed"
	endedAgain := cloneEventMessage(session.Ended{
		SessionID:            "sess-1",
		Name:                 "Session 1",
		CharacterControllers: []session.CharacterControllerAssignment{{CharacterID: "char-1", ParticipantID: "part-1"}},
	}).(session.Ended)
	if got, want := endedAgain.CharacterControllers[0].CharacterID, "char-1"; got != want {
		t.Fatalf("session ended clone = %q, want %q", got, want)
	}

	updated := campaign.Updated{Name: "Autumn Eclipse"}
	if got := cloneEventMessage(updated); got != updated {
		t.Fatalf("cloneEventMessage(default) = %#v, want %#v", got, updated)
	}
}

func TestCloneEventMessageCoversRegisteredEventsWithContainers(t *testing.T) {
	t.Parallel()

	want := map[reflect.Type]struct{}{
		reflect.TypeFor[scene.Created]():      {},
		reflect.TypeFor[scene.CastReplaced](): {},
		reflect.TypeFor[session.Started]():    {},
		reflect.TypeFor[session.Ended]():      {},
	}
	if got := registeredEventTypesWithContainers(); !maps.Equal(got, want) {
		t.Fatalf("registered event clone coverage = %v, want %v", got, want)
	}
}

func registeredEventTypesWithContainers() map[reflect.Type]struct{} {
	types := make(map[reflect.Type]struct{})
	for _, module := range DefaultModules() {
		for _, spec := range module.Events() {
			messageType := spec.Definition().MessageType
			if messageType.Kind() == reflect.Pointer {
				messageType = messageType.Elem()
			}
			if !messageTypeHasContainerFields(messageType) {
				continue
			}
			types[messageType] = struct{}{}
		}
	}
	return types
}

func messageTypeHasContainerFields(messageType reflect.Type) bool {
	for field := range messageType.Fields() {
		switch field.Type.Kind() {
		case reflect.Slice, reflect.Map:
			return true
		}
	}
	return false
}
