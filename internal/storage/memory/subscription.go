package memory

import (
	"context"

	"github.com/fracturing-space/game/internal/event"
	"github.com/fracturing-space/game/internal/storage/storeutil"
)

type subscriptionHandle = storeutil.SubscriptionHandle

func newSubscriptionHandle(ctx context.Context, afterSeq uint64, initial []event.Record) *subscriptionHandle {
	return storeutil.NewSubscriptionHandle(ctx, afterSeq, initial)
}
