CREATE TABLE IF NOT EXISTS progress (
    document TEXT PRIMARY KEY,
    progress TEXT NOT NULL,
    percentage NUMERIC NOT NULL,
    device_id TEXT NOT NULL,
    device TEXT NOT NULL
);

