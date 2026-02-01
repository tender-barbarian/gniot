CREATE TABLE IF NOT EXISTS jobs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    devices TEXT,
    action TEXT,
    runat TEXT,
    interval TEXT
);
