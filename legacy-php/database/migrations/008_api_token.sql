CREATE TABLE api_token (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    nama TEXT NOT NULL,
    prefix TEXT NOT NULL,
    token_hash TEXT NOT NULL UNIQUE,
    akses TEXT NOT NULL DEFAULT 'baca' CHECK (akses IN ('baca', 'tulis')),
    created_at TEXT NOT NULL DEFAULT (datetime('now', 'localtime')),
    last_used_at TEXT
);

CREATE TABLE api_log (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    token_id INTEGER REFERENCES api_token(id) ON DELETE SET NULL,
    tool TEXT NOT NULL,
    args TEXT NOT NULL DEFAULT '',
    ok INTEGER NOT NULL DEFAULT 1,
    pesan TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL DEFAULT (datetime('now', 'localtime'))
);

CREATE INDEX idx_api_log_waktu ON api_log(created_at);
