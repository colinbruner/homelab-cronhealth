-- Add ping_token_hash to checks.
-- Empty string default: existing checks have no token until explicitly rotated.
-- The ping handler returns 401 with a helpful message when this is empty.
ALTER TABLE checks ADD COLUMN ping_token_hash TEXT NOT NULL DEFAULT '';
