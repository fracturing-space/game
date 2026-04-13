CREATE TABLE IF NOT EXISTS projection_snapshots (
	campaign_id TEXT PRIMARY KEY,
	head_seq INTEGER NOT NULL,
	state_blob BLOB NOT NULL,
	updated_at_ns INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS projection_watermarks (
	campaign_id TEXT PRIMARY KEY,
	applied_seq INTEGER NOT NULL,
	expected_next_seq INTEGER NOT NULL,
	updated_at_ns INTEGER NOT NULL
);
