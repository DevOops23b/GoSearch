
CREATE TABLE IF NOT EXISTS users (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  username TEXT NOT NULL UNIQUE,
  email TEXT NOT NULL UNIQUE,
  password TEXT NOT NULL
);

-- Create a default user, The password is 'password' (MD5 hashed)
INSERT INTO users (username, email, password) 
    VALUES ('admin', 'keamonk1@stud.kea.dk', '5f4dcc3b5aa765d61d8327deb882cf99');


CREATE TABLE IF NOT EXISTS pages (
    title TEXT PRIMARY KEY UNIQUE,
    url TEXT NOT NULL UNIQUE,
    language TEXT NOT NULL CHECK(language IN ('en', 'da')) DEFAULT 'en',
    last_updated TIMESTAMP,
    content TEXT NOT NULL
);

-- Create virtual with fts.
DROP TABLE IF EXISTS pages_fts;

CREATE VIRTUAL TABLE pages_fts USING fts5(title, url, language);

-- Getting data from pages and putting it into pages_fts table.
INSERT INTO pages_fts (title, content, language)
SELECT title, content, language FROM pages;

-- Trigger to keep FTS table updated on INSERT (Changed this bc og sonarQube)
CREATE OR REPLACE TRIGGER pages_insert AFTER INSERT ON pages
BEGIN
  INSERT INTO pages_fts (title, content, language)
  VALUES (NEW.title, NEW.content, NEW.language);
END;

-- Trigger to keep FTS table updated on UPDATE
CREATE TRIGGER IF NOT EXISTS pages_au AFTER UPDATE ON pages
BEGIN
    DELETE FROM pages_fts WHERE title = OLD.title;
    INSERT INTO pages_fts (title, content, language) VALUES (NEW.title, NEW.content, NEW.language);
END;

-- Trigger to keep FTS table updated on DELETE
CREATE TRIGGER IF NOT EXISTS pages_ad AFTER DELETE ON pages
BEGIN
    DELETE FROM pages_fts WHERE title = OLD.title;
END;
