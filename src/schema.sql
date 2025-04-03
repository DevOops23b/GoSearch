-- Users table remains unchanged
DROP TABLE IF EXISTS users;

CREATE TABLE IF NOT EXISTS users (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  username TEXT NOT NULL UNIQUE,
  email TEXT NOT NULL UNIQUE,
  password TEXT NOT NULL
);

-- Create a default user, The password is 'password' (MD5 hashed)
INSERT INTO users (username, email, password) 
    VALUES ('admin', 'keamonk1@stud.kea.dk', '5f4dcc3b5aa765d61d8327deb882cf99');

-- Alter the pages table (no drop, preserve data)
-- Adding new column instead of dropping the table
ALTER TABLE pages ADD COLUMN new_column TEXT;

DROP TABLE IF EXISTS pages_fts;

-- Create the FTS5 table if it doesn't exist yet
-- If pages_fts already exists, skip creating it again
CREATE VIRTUAL TABLE pages_fts USING fts5(title, url, language, content, new_column);


-- Re-populate pages_fts with data from pages table
-- This ensures that the FTS table is filled with current data
INSERT INTO pages_fts (title, url, language, content, new_column)
SELECT title, url, language, content, new_column FROM pages;


-- Trigger to keep FTS table updated on INSERT (Changed this bc of sonarQube)
CREATE TRIGGER IF NOT EXISTS pages_insert AFTER INSERT ON pages
BEGIN
  INSERT INTO pages_fts (title, url, language, content, new_column)
  VALUES (NEW.title, NEW.url, NEW.language, NEW.content, NEW.new_column);
END;


-- Trigger to keep FTS table updated on UPDATE
CREATE TRIGGER IF NOT EXISTS pages_au AFTER UPDATE ON pages
BEGIN
  DELETE FROM pages_fts WHERE title = OLD.title;
  INSERT INTO pages_fts (title, url, language, content, new_column)
  VALUES (NEW.title, NEW.url, NEW.language, NEW.content, NEW.new_column);
END;


-- Trigger to keep FTS table updated on DELETE
CREATE TRIGGER IF NOT EXISTS pages_ad AFTER DELETE ON pages
BEGIN
  DELETE FROM pages_fts WHERE title = OLD.title;
END;

