exports.up = function(knex) {
    return knex.raw(`
      CREATE INDEX IF NOT EXISTS idx_pages_content ON pages USING GIN (to_tsvector('english', content));
      CREATE INDEX IF NOT EXISTS idx_pages_title ON pages USING GIN (to_tsvector('english', title));
    `);
  };
  
  exports.down = function(knex) {
    return knex.raw(`
      DROP INDEX IF EXISTS idx_pages_content;
      DROP INDEX IF EXISTS idx_pages_title;
    `);
  };