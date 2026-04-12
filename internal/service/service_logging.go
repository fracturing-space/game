package service

import (
	"context"
	"log/slog"

	"github.com/fracturing-space/game/internal/caller"
	"github.com/fracturing-space/game/internal/command"
	"github.com/fracturing-space/game/internal/event"
)

func withServiceLogger(logger *slog.Logger) *slog.Logger {
	if logger == nil {
		return nil
	}
	return logger.With("component", "service")
}

func (s *Service) logCommandRejected(ctx context.Context, act caller.Caller, envelope command.Envelope, err error) {
	if s == nil || s.logger == nil || err == nil {
		return
	}
	s.logger.LogAttrs(
		ctx,
		slog.LevelWarn,
		"command rejected",
		append(commandLogAttrs(act, string(envelope.Type()), envelope.CampaignID), slog.Any("error", err))...,
	)
}

func (s *Service) logCommandAccepted(ctx context.Context, act caller.Caller, envelope command.Envelope, campaignID string, events []event.Envelope) {
	if s == nil || s.logger == nil {
		return
	}
	attrs := append(commandLogAttrs(act, string(envelope.Type()), campaignID),
		slog.String("target", "live"),
		slog.Int("event_count", len(events)),
		slog.Any("event_types", eventTypes(events)),
	)
	s.logger.LogAttrs(ctx, slog.LevelInfo, "command accepted", attrs...)
}

func (s *Service) logLiveEventBatch(ctx context.Context, campaignID string, records []event.Record) {
	if s == nil || s.logger == nil || len(records) == 0 {
		return
	}
	s.logger.LogAttrs(ctx, slog.LevelInfo, "event batch stored",
		appendNonEmptyAttrs(
			nil,
			nonEmptyStringAttr("campaign_id", campaignID),
			slog.String("target", "live"),
			slog.Int("event_count", len(records)),
			slog.Any("event_types", recordEventTypes(records)),
			slog.Uint64("first_seq", records[0].Seq),
			slog.Uint64("last_seq", records[len(records)-1].Seq),
			slog.Uint64("commit_seq", records[len(records)-1].CommitSeq),
		)...,
	)
}

func commandLogAttrs(act caller.Caller, commandType, campaignID string) []slog.Attr {
	return appendNonEmptyAttrs(
		nil,
		nonEmptyStringAttr("subject_id", act.SubjectID),
		nonEmptyStringAttr("ai_agent_id", act.AIAgentID),
		nonEmptyStringAttr("command_type", commandType),
		nonEmptyStringAttr("campaign_id", campaignID),
	)
}

func nonEmptyStringAttr(key, value string) slog.Attr {
	if value == "" {
		return slog.Attr{}
	}
	return slog.String(key, value)
}

func appendNonEmptyAttrs(dst []slog.Attr, attrs ...slog.Attr) []slog.Attr {
	for _, attr := range attrs {
		if attr.Equal(slog.Attr{}) {
			continue
		}
		dst = append(dst, attr)
	}
	return dst
}

func eventTypes(events []event.Envelope) []string {
	types := make([]string, 0, len(events))
	for _, next := range events {
		types = append(types, string(next.Type()))
	}
	return types
}

func recordEventTypes(records []event.Record) []string {
	types := make([]string, 0, len(records))
	for _, record := range records {
		types = append(types, string(record.Type()))
	}
	return types
}
