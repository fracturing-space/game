-- name: GetProjection :one
SELECT head_seq, state_blob, updated_at_ns, last_activity_at_ns
FROM projection_snapshots
WHERE campaign_id = ?;

-- name: PutProjection :exec
INSERT INTO projection_snapshots (campaign_id, head_seq, state_blob, updated_at_ns, last_activity_at_ns)
VALUES (?, ?, ?, ?, ?)
ON CONFLICT(campaign_id) DO UPDATE SET
	head_seq = excluded.head_seq,
	state_blob = excluded.state_blob,
	updated_at_ns = excluded.updated_at_ns,
	last_activity_at_ns = excluded.last_activity_at_ns;

-- name: GetWatermark :one
SELECT applied_seq, expected_next_seq, updated_at_ns
FROM projection_watermarks
WHERE campaign_id = ?;

-- name: PutWatermark :exec
INSERT INTO projection_watermarks (campaign_id, applied_seq, expected_next_seq, updated_at_ns)
VALUES (?, ?, ?, ?)
ON CONFLICT(campaign_id) DO UPDATE SET
	applied_seq = excluded.applied_seq,
	expected_next_seq = excluded.expected_next_seq,
	updated_at_ns = excluded.updated_at_ns;

-- name: PutCampaignSummary :exec
INSERT INTO projection_campaign_summaries (
	campaign_id, name, ready_to_play, has_ai_binding, has_active_session, last_activity_at_ns, updated_at_ns
)
VALUES (?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(campaign_id) DO UPDATE SET
	name = excluded.name,
	ready_to_play = excluded.ready_to_play,
	has_ai_binding = excluded.has_ai_binding,
	has_active_session = excluded.has_active_session,
	last_activity_at_ns = excluded.last_activity_at_ns,
	updated_at_ns = excluded.updated_at_ns;

-- name: DeleteCampaignSubjects :exec
DELETE FROM projection_campaign_subjects
WHERE campaign_id = ?;

-- name: PutCampaignSubject :exec
INSERT INTO projection_campaign_subjects (subject_id, campaign_id)
VALUES (?, ?)
ON CONFLICT(subject_id, campaign_id) DO NOTHING;

-- name: ListCampaignsBySubject :many
SELECT
	s.campaign_id,
	s.name,
	s.ready_to_play,
	s.has_ai_binding,
	s.has_active_session,
	s.last_activity_at_ns
FROM projection_campaign_subjects AS idx
JOIN projection_campaign_summaries AS s ON s.campaign_id = idx.campaign_id
WHERE idx.subject_id = ?
ORDER BY s.last_activity_at_ns DESC, s.campaign_id ASC
LIMIT ?;

-- name: ListProjectionSnapshotsForBackfill :many
SELECT p.campaign_id, p.head_seq, p.state_blob, p.updated_at_ns, p.last_activity_at_ns
FROM projection_snapshots AS p
LEFT JOIN projection_campaign_summaries AS s ON s.campaign_id = p.campaign_id
WHERE s.campaign_id IS NULL
ORDER BY p.campaign_id ASC;
