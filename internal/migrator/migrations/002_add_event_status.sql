-- +goose Up
ALTER TABLE events
ADD COLUMN status VARCHAR(50) DEFAULT 'NEW' NOT NULL;

-- +goose Down
ALTER TABLE events DROP COLUMN status;