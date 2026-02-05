-- Добавление колонки status в таблицу events
ALTER TABLE events
ADD COLUMN IF NOT EXISTS status VARCHAR(50) DEFAULT 'NEW' NOT NULL;