-- name: PutArtifact :exec
INSERT INTO campaign_artifacts (campaign_id, path, content, updated_at_ns)
VALUES (?, ?, ?, ?)
ON CONFLICT(campaign_id, path) DO UPDATE SET
	content = excluded.content,
	updated_at_ns = excluded.updated_at_ns;

-- name: GetArtifact :one
SELECT content, updated_at_ns
FROM campaign_artifacts
WHERE campaign_id = ? AND path = ?;

-- name: ListArtifactsByCampaign :many
SELECT path, content, updated_at_ns
FROM campaign_artifacts
WHERE campaign_id = ?
ORDER BY path;
