-- name: GetEventHead :one
SELECT
	CAST(COUNT(*) AS INTEGER) AS event_count,
	CAST(COALESCE(MAX(seq), 0) AS INTEGER) AS head_seq,
	CAST(COALESCE(MAX(commit_seq), 0) AS INTEGER) AS head_commit_seq
FROM campaign_events
WHERE campaign_id = ?;

-- name: AppendEvent :exec
INSERT INTO campaign_events (
	campaign_id, seq, commit_seq, recorded_at_ns, event_type, payload_blob
) VALUES (?, ?, ?, ?, ?, ?);

-- name: ListEventsAfter :many
SELECT seq, commit_seq, recorded_at_ns, event_type, payload_blob
FROM campaign_events
WHERE campaign_id = ? AND seq > ?
ORDER BY seq;
