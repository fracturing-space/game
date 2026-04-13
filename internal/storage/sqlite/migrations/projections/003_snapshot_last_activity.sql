ALTER TABLE projection_snapshots
	ADD COLUMN last_activity_at_ns INTEGER NOT NULL DEFAULT 0;

UPDATE projection_snapshots
SET last_activity_at_ns = updated_at_ns
WHERE last_activity_at_ns = 0;
