CREATE TABLE IF NOT EXISTS jobs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    devices TEXT,
    action TEXT,
    run_at TEXT,
    interval TEXT
);
