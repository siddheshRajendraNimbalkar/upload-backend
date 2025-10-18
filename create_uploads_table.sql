-- Create database first (if not exists)
-- CREATE DATABASE upload_db;

-- Connect to your database
-- \c upload_db

-- Create uploads table
CREATE TABLE IF NOT EXISTS uploads (
    file_id UUID PRIMARY KEY,           -- Unique ID for the file
    user_id UUID,                       -- Optional: track which user uploaded
    file_name TEXT NOT NULL,            -- Original file name
    total_chunks BIGINT NOT NULL,       -- Total number of chunks
    status TEXT NOT NULL DEFAULT 'in_progress', -- Status: in_progress/completed
    stored_path TEXT,                   -- Final stored file path
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP -- Upload creation time
);
