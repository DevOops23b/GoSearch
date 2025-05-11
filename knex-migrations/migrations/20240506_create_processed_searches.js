exports.up = function(knex) {
    return knex.schema.createTable('processed_searches', function(table) {
      table.text('search_term').primary();
      table.timestamp('processed_at').defaultTo(knex.fn.now());
    });
  };
  
  exports.down = function(knex) {
    return knex.schema.dropTableIfExists('processed_searches');
  };
  