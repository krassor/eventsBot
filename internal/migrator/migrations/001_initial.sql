-- Начальная миграция
CREATE TABLE IF NOT EXISTS events (
    id UUID PRIMARY KEY,
    name TEXT NOT NULL,
    photo TEXT,
    description TEXT,
    date TIMESTAMP WITH TIME ZONE,
    price DOUBLE PRECISION,
    currency TEXT,
    event_link TEXT,
    map_link TEXT,
    calendar_link_ios TEXT,
    calendar_link_android TEXT,
    tag TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_events_date ON events (date);

CREATE INDEX IF NOT EXISTS idx_events_tag ON events (tag);