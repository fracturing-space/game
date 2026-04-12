package service

import (
	"sync"
	"time"

	"github.com/fracturing-space/game/internal/caller"
	"github.com/fracturing-space/game/internal/campaign"
	"github.com/fracturing-space/game/internal/errs"
)

type planStore struct {
	mu         sync.Mutex
	items      map[string]preparedPlan
	byCampaign map[string]string
}

type preparedPlan struct {
	token      string
	caller     caller.Caller
	campaignID string
	baseSeq    uint64
	commits    []PreparedCommit
	state      campaign.State
	expiresAt  time.Time
}

func (s *Service) storePlan(plan preparedPlan, now time.Time) error {
	if s == nil {
		return nil
	}
	store := s.planStore()
	store.mu.Lock()
	defer store.mu.Unlock()
	store.sweepExpiredLocked(now)
	if token, ok := store.byCampaign[plan.campaignID]; ok {
		if _, exists := store.items[token]; exists {
			return errs.Conflictf("campaign %s already has an active plan", plan.campaignID)
		}
		delete(store.byCampaign, plan.campaignID)
	}
	store.items[plan.token] = plan
	store.byCampaign[plan.campaignID] = plan.token
	return nil
}

func (s *Service) consumePlan(token string, act caller.Caller, now time.Time) (preparedPlan, error) {
	store := s.planStore()
	store.mu.Lock()
	defer store.mu.Unlock()

	if store.items == nil {
		return preparedPlan{}, errs.NotFoundf("plan %s not found", token)
	}
	plan, ok := store.items[token]
	if !ok {
		return preparedPlan{}, errs.NotFoundf("plan %s not found", token)
	}
	if now.After(plan.expiresAt) {
		delete(store.items, token)
		delete(store.byCampaign, plan.campaignID)
		return preparedPlan{}, errs.NotFoundf("plan %s not found", token)
	}
	if !act.SameIdentity(plan.caller) {
		return preparedPlan{}, errs.FailedPreconditionf("plan %s belongs to a different caller", token)
	}
	delete(store.items, token)
	delete(store.byCampaign, plan.campaignID)
	return plan, nil
}

func (s *Service) planStore() *planStore {
	if s == nil || s.plans == nil {
		return &planStore{
			items:      make(map[string]preparedPlan),
			byCampaign: make(map[string]string),
		}
	}
	return s.plans
}

func (s *planStore) sweepExpiredLocked(now time.Time) {
	if s == nil || len(s.items) == 0 {
		return
	}
	for token, plan := range s.items {
		if now.After(plan.expiresAt) {
			delete(s.items, token)
			delete(s.byCampaign, plan.campaignID)
		}
	}
}
