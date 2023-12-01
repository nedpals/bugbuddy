-- Create the settings table
CREATE TABLE IF NOT EXISTS settings (
    name TEXT PRIMARY KEY,
    value TEXT NOT NULL
);

-- Create the files table for storing source codes
CREATE TABLE IF NOT EXISTS files (
    id INTEGER PRIMARY KEY,
    participant_id TEXT NOT NULL,
    file_path TEXT NOT NULL UNIQUE,
    content TEXT NOT NULL,
    created_at TEXT NOT NULL
);

-- Create the logs table
CREATE TABLE IF NOT EXISTS logs (
    id INTEGER PRIMARY KEY,
    participant_id TEXT NOT NULL,
    executed_command TEXT NOT NULL,
    error_code TEXT NOT NULL,
    error_message TEXT NOT NULL,
    generated_output TEXT NOT NULL,
    file_path TEXT NOT NULL,
    created_at TEXT NOT NULL
);
