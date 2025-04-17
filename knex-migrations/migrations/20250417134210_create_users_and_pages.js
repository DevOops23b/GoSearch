exports.up = function(knex) {
    return knex.schema
      .createTable('users', function(table) {
        table.increments('id').primary();
        table.text('username').notNullable().unique();
        table.text('email').notNullable().unique();
        table.text('password').notNullable();
      })
      .then(function() {
        // Insert default admin user
        return knex('users').insert({
          username: 'admin',
          email: 'keamonk1@stud.kea.dk',
          password: '5f4dcc3b5aa765d61d8327deb882cf99'
        }).onConflict('username').ignore();
      })
      .then(function() {
        return knex.schema.createTable('pages', function(table) {
          table.text('title').primary();
          table.text('url').notNullable().unique();
          table.text('language').notNullable().defaultTo('en')
            .checkIn(['en', 'da']);
          table.timestamp('last_updated').defaultTo(knex.fn.now());
          table.text('content').notNullable();
        });
      });
  };
  
  exports.down = function(knex) {
    return knex.schema
      .dropTableIfExists('pages')
      .dropTableIfExists('users');
  };