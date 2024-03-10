-- Create the settings table
CREATE TABLE IF NOT EXISTS settings (
    name TEXT PRIMARY KEY,
    value TEXT NOT NULL
);

-- Create the files table for storing source codes
CREATE TABLE IF NOT EXISTS files (
    id INTEGER PRIMARY KEY,
    participant_id TEXT NOT NULL,
    file_path TEXT NOT NULL,
    file_version INTEGER DEFAULT 1,
    content TEXT NOT NULL,
    created_at TEXT NOT NULL,
    UNIQUE(participant_id, file_path, file_version) ON CONFLICT REPLACE
);

-- Create the logs table
CREATE TABLE IF NOT EXISTS logs (
    id INTEGER PRIMARY KEY,
    participant_id TEXT NOT NULL,
    executed_command TEXT NOT NULL,
    error_code INTEGER NOT NULL,
    error_message TEXT NOT NULL,
    generated_output TEXT NOT NULL,
    error_line INTEGER NOT NULL,
    error_column INTEGER NOT NULL,
    file_path TEXT NOT NULL,
    file_version INTEGER NOT NULL,
    created_at TEXT NOT NULL
);
