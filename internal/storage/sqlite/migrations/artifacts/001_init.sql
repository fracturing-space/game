CREATE TABLE IF NOT EXISTS campaign_artifacts (
	campaign_id TEXT NOT NULL,
	path TEXT NOT NULL,
	content TEXT NOT NULL,
	updated_at_ns INTEGER NOT NULL,
	PRIMARY KEY (campaign_id, path)
);
