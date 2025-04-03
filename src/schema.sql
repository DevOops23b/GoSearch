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

-- Drop pages_fts if it exists (to clear old triggers, tables)
DROP TABLE IF EXISTS pages_fts;

-- Create the FTS5 table without 'new_column'
CREATE VIRTUAL TABLE pages_fts USING fts5(title, url, language, content);

-- Re-populate pages_fts with data from pages table
INSERT INTO pages_fts (title, url, language, content)
SELECT title, url, language, content FROM pages;

-- Trigger to keep FTS table updated on INSERT
CREATE TRIGGER IF NOT EXISTS pages_insert AFTER INSERT ON pages
BEGIN
  INSERT INTO pages_fts (title, url, language, content)
  VALUES (NEW.title, NEW.url, NEW.language, NEW.content);
END;

-- Trigger to keep FTS table updated on UPDATE
CREATE TRIGGER IF NOT EXISTS pages_au AFTER UPDATE ON pages
BEGIN
  DELETE FROM pages_fts WHERE title = OLD.title;
  INSERT INTO pages_fts (title, url, language, content)
  VALUES (NEW.title, NEW.url, NEW.language, NEW.content);
END;

-- Trigger to keep FTS table updated on DELETE
CREATE TRIGGER IF NOT EXISTS pages_ad AFTER DELETE ON pages
BEGIN
  DELETE FROM pages_fts WHERE title = OLD.title;
END;
