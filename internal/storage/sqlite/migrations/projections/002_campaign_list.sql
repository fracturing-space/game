CREATE TABLE IF NOT EXISTS projection_campaign_summaries (
	campaign_id TEXT PRIMARY KEY,
	name TEXT NOT NULL,
	ready_to_play INTEGER NOT NULL,
	has_ai_binding INTEGER NOT NULL,
	has_active_session INTEGER NOT NULL,
	last_activity_at_ns INTEGER NOT NULL,
	updated_at_ns INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS projection_campaign_subjects (
	subject_id TEXT NOT NULL,
	campaign_id TEXT NOT NULL,
	PRIMARY KEY (subject_id, campaign_id)
);

CREATE INDEX IF NOT EXISTS idx_projection_campaign_subjects_subject
	ON projection_campaign_subjects (subject_id, campaign_id);
