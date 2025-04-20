CREATE TABLE IF NOT EXISTS users (
  id SERIAL PRIMARY KEY,
  username TEXT NOT NULL UNIQUE,
  email TEXT NOT NULL UNIQUE,
  password TEXT NOT NULL
);

INSERT INTO users (username, email, password) 
    VALUES ('{{ADMIN_USER}}', '{{ADMIN_EMAIL}}', '{{ADMIN_PASS_HASH}}')
    ON CONFLICT (username) DO NOTHING;

CREATE TABLE IF NOT EXISTS pages (
    title TEXT PRIMARY KEY,
    url TEXT NOT NULL UNIQUE,
    language TEXT NOT NULL CHECK(language IN ('en', 'da')) DEFAULT 'en',
    last_updated TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    content TEXT NOT NULL
);

-- Create an index on content and title for improved search performance
CREATE INDEX IF NOT EXISTS idx_pages_content ON pages USING GIN (to_tsvector('english', content));
CREATE INDEX IF NOT EXISTS idx_pages_title ON pages USING GIN (to_tsvector('english', title));
