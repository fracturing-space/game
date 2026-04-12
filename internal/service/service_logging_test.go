package service

import (
	"context"
	"log/slog"
	"slices"
	"sync"
	"testing"

	"github.com/fracturing-space/game/internal/campaign"
	"github.com/fracturing-space/game/internal/character"
	"github.com/fracturing-space/game/internal/command"
	"github.com/fracturing-space/game/internal/event"
	"github.com/fracturing-space/game/internal/participant"
)

type recordedLog struct {
	level   slog.Level
	message string
	attrs   map[string]any
}

type recordingStore struct {
	mu      sync.Mutex
	records []recordedLog
}

type recordingHandler struct {
	store *recordingStore
	attrs []slog.Attr
}

func (h *recordingHandler) Enabled(context.Context, slog.Level) bool { return true }

func (h *recordingHandler) Handle(_ context.Context, record slog.Record) error {
	entry := recordedLog{
		level:   record.Level,
		message: record.Message,
		attrs:   make(map[string]any),
	}
	for _, attr := range h.attrs {
		entry.attrs[attr.Key] = attr.Value.Any()
	}
	record.Attrs(func(attr slog.Attr) bool {
		entry.attrs[attr.Key] = attr.Value.Any()
		return true
	})

	h.store.mu.Lock()
	defer h.store.mu.Unlock()
	h.store.records = append(h.store.records, entry)
	return nil
}

func (h *recordingHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &recordingHandler{
		store: h.store,
		attrs: append(append([]slog.Attr(nil), h.attrs...), attrs...),
	}
}

func (h *recordingHandler) WithGroup(string) slog.Handler { return h }

func (s *recordingStore) Records() []recordedLog {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]recordedLog(nil), s.records...)
}

func newRecordingLogger() (*slog.Logger, *recordingStore) {
	store := &recordingStore{}
	return slog.New(&recordingHandler{store: store}), store
}

func TestExecuteLiveLogsCommandAndStoredEvents(t *testing.T) {
	logger, store := newRecordingLogger()
	svc := newTestServiceWithLogger(t, logger)

	result, err := svc.CommitCommand(context.Background(), defaultCaller(), command.Envelope{
		Message: campaign.Create{Name: "Autumn Twilight", OwnerName: "louis"},
	})
	if err != nil {
		t.Fatalf("CommitCommand(create) error = %v", err)
	}

	records := store.Records()
	if got, want := len(records), 2; got != want {
		t.Fatalf("log count = %d, want %d", got, want)
	}
	assertLog(t, records[0], slog.LevelInfo, "command accepted")
	if got, want := records[0].attrs["component"], "service"; got != want {
		t.Fatalf("component = %v, want %q", got, want)
	}
	if got, want := records[0].attrs["subject_id"], "subject-1"; got != want {
		t.Fatalf("subject_id = %v, want %q", got, want)
	}
	if got, want := records[0].attrs["command_type"], string(campaign.CommandTypeCreate); got != want {
		t.Fatalf("command_type = %v, want %q", got, want)
	}
	if got, want := records[0].attrs["campaign_id"], result.State.ID; got != want {
		t.Fatalf("campaign_id = %v, want %q", got, want)
	}
	if got, want := records[0].attrs["target"], "live"; got != want {
		t.Fatalf("target = %v, want %q", got, want)
	}
	if got, want := records[0].attrs["event_count"], int64(2); got != want {
		t.Fatalf("event_count = %v, want %d", got, want)
	}
	if got, want := records[0].attrs["event_types"], []string{
		string(campaign.EventTypeCreated),
		string(participant.EventTypeJoined),
	}; !slices.Equal(got.([]string), want) {
		t.Fatalf("event_types = %v, want %v", got, want)
	}

	assertLog(t, records[1], slog.LevelInfo, "event batch stored")
	if got, want := records[1].attrs["campaign_id"], result.State.ID; got != want {
		t.Fatalf("campaign_id = %v, want %q", got, want)
	}
	if got, want := records[1].attrs["target"], "live"; got != want {
		t.Fatalf("target = %v, want %q", got, want)
	}
	if got, want := records[1].attrs["first_seq"], uint64(1); got != want {
		t.Fatalf("first_seq = %v, want %d", got, want)
	}
	if got, want := records[1].attrs["last_seq"], uint64(2); got != want {
		t.Fatalf("last_seq = %v, want %d", got, want)
	}
	if got, want := records[1].attrs["commit_seq"], uint64(1); got != want {
		t.Fatalf("commit_seq = %v, want %d", got, want)
	}
}

func TestExecuteRejectedLogsWarning(t *testing.T) {
	logger, store := newRecordingLogger()
	svc := newTestServiceWithLogger(t, logger)

	if _, err := svc.CommitCommand(context.Background(), defaultCaller(), command.Envelope{
		Message: campaign.Create{Name: "   ", OwnerName: "louis"},
	}); err == nil {
		t.Fatal("CommitCommand() error = nil, want failure")
	}

	records := store.Records()
	if got, want := len(records), 1; got != want {
		t.Fatalf("log count = %d, want %d", got, want)
	}
	assertLog(t, records[0], slog.LevelWarn, "command rejected")
	if got, want := records[0].attrs["command_type"], string(campaign.CommandTypeCreate); got != want {
		t.Fatalf("command_type = %v, want %q", got, want)
	}
	if _, ok := records[0].attrs["error"]; !ok {
		t.Fatal("error attr missing")
	}
}

func TestEventTypeHelpers(t *testing.T) {
	envelopes := []event.Envelope{
		{CampaignID: "camp-1", Message: campaign.Created{Name: "Autumn Twilight"}},
		{CampaignID: "camp-1", Message: character.Created{CharacterID: "char-1", ParticipantID: "part-1", Name: "luna"}},
	}
	if got, want := eventTypes(envelopes), []string{string(campaign.EventTypeCreated), string(character.EventTypeCreated)}; !slices.Equal(got, want) {
		t.Fatalf("eventTypes() = %v, want %v", got, want)
	}
}

func TestNilLoggerProducesNoPanic(t *testing.T) {
	svc := newTestService(t)
	if _, err := svc.CommitCommand(context.Background(), defaultCaller(), command.Envelope{
		Message: campaign.Create{Name: "Autumn Twilight", OwnerName: "louis"},
	}); err != nil {
		t.Fatalf("CommitCommand(create) error = %v", err)
	}
}

func assertLog(t *testing.T, record recordedLog, level slog.Level, message string) {
	t.Helper()
	if got, want := record.level, level; got != want {
		t.Fatalf("level = %v, want %v", got, want)
	}
	if got, want := record.message, message; got != want {
		t.Fatalf("message = %q, want %q", got, want)
	}
}
