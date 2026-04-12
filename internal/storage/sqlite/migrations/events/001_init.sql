CREATE TABLE IF NOT EXISTS campaign_events (
	campaign_id TEXT NOT NULL,
	seq INTEGER NOT NULL,
	commit_seq INTEGER NOT NULL,
	recorded_at_ns INTEGER NOT NULL,
	event_type TEXT NOT NULL,
	payload_blob BLOB NOT NULL,
	PRIMARY KEY (campaign_id, seq)
);

CREATE INDEX IF NOT EXISTS idx_campaign_events_commit_seq
	ON campaign_events (campaign_id, commit_seq);
