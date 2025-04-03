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

DROP TABLE IF EXISTS pages;

CREATE TABLE IF NOT EXISTS pages (
    title TEXT PRIMARY KEY UNIQUE,
    url TEXT NOT NULL UNIQUE,
    language TEXT NOT NULL CHECK(language IN ('en', 'da')) DEFAULT 'en',
    last_updated TIMESTAMP,
    content TEXT NOT NULL
);

-- Create virtual with fts.
DROP TABLE IF EXISTS pages_fts;

CREATE VIRTUAL TABLE pages_fts USING fts5(title, url, content, language);

-- Getting data from pages and putting it into pages_fts table.
INSERT INTO pages_fts (title, url, content, language)
SELECT title, url, content, language FROM pages;

-- It can't make a "CREATE OR REPLACE", therefore we drop it first.
DROP TABLE IF EXISTS pages_insert;
-- creater a triigger to update pages_fts every time pages get updatet.
CREATE TRIGGER pages_insert AFTER INSERT ON pages
BEGIN
  INSERT INTO pages_fts (title, url, content, language)
  VALUES (NEW.title, NEW.url, NEW.content, NEW.language);
END;

-- This is to drop the table made in the earlier schema. just in case.
DROP TRIGGER IF EXISTS pages_au;

DROP TRIGGER IF EXISTS pages_update;

-- Trigger to keep FTS table updated on UPDATE
CREATE TRIGGER pages_update AFTER UPDATE ON pages
BEGIN
    DELETE FROM pages_fts WHERE title = OLD.title;
    INSERT INTO pages_fts (title, url, content, language) 
    VALUES (NEW.title, NEW.url, NEW.content, NEW.language);
END;

-- Again this is to drop the table made in the earlier schema. Just in case.
DROP TRIGGER IF EXISTS pages_ad;

DROP TRIGGER IF EXISTS pages_delete;

-- Trigger to keep FTS table updated on DELETE
CREATE TRIGGER pages_delete AFTER DELETE ON pages
BEGIN
    DELETE FROM pages_fts WHERE title = OLD.title;
END;
