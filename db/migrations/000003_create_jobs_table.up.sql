CREATE TABLE IF NOT EXISTS jobs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE,
    devices TEXT,
    action TEXT,
    run_at TEXT,
    interval TEXT,
    enabled INTEGER NOT NULL DEFAULT 1,
    last_check TEXT,
    last_triggered TEXT,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);
