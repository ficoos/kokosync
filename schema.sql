CREATE TABLE IF NOT EXISTS progress (
    book_id TEXT NOT NULL,
    timestamp NUMERIC NOT NULL,
    fragment NUMERIC NOT NULL,
    selector TEXT NOT NULL,
    total_progress NUMERIC NOT NULL,
    device_id TEXT NOT NULL,
    added_to_library NUMERIC,
    char_offset NUMERIC NOT NULL,

    PRIMARY KEY (book_id, timestamp)
);

CREATE TABLE IF NOT EXISTS books (
    title TEXT NOT NULL,
    author TEXT NOT NULL,
    normalized_title TEXT NOT NULL,
    book_id TEXT NOT NULL,
    file_url TEXT NOT NULL,
    PRIMARY KEY (title, author)
);

CREATE VIRTUAL TABLE IF NOT EXISTS books_fts USING fts4(
    normalized_title,
    book_id UNINDEXED
);

CREATE TABLE IF NOT EXISTS document_hashs (
    book_id TEXT PRIMARY KEY,
    document_hash TEXT NOT NULL
);
