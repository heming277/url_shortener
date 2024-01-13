-- migrations/002_create_urls_table.sql

CREATE TABLE IF NOT EXISTS urls (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id) ON DELETE CASCADE, 
    original_url TEXT NOT NULL,
    shortened_url VARCHAR(255) NOT NULL,
    visit_count INTEGER DEFAULT 0,
    UNIQUE(user_id, original_url),
    UNIQUE(shortened_url)
);
