-- Add video_url column to events table
ALTER TABLE events ADD COLUMN IF NOT EXISTS video_url TEXT;