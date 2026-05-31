-- Add source field to alerts table
ALTER TABLE alerts ADD COLUMN IF NOT EXISTS source TEXT;
