-- Add status constraint
ALTER TABLE uploads ADD CONSTRAINT status_check CHECK (status IN ('in_progress','completed','failed'));

-- Add useful indices
CREATE INDEX IF NOT EXISTS idx_uploads_user_created ON uploads (user_id, created_at DESC);

-- Add additional columns for better tracking
ALTER TABLE uploads ADD COLUMN IF NOT EXISTS size_bytes BIGINT DEFAULT 0;
ALTER TABLE uploads ADD COLUMN IF NOT EXISTS mime_type TEXT;
ALTER TABLE uploads ADD COLUMN IF NOT EXISTS sha256 TEXT;