exports.up = function(knex) {
    return knex.schema
      .alterTable('users', function(table) {
        table.boolean('password_changed').notNullable().defaultTo(true);
      })
      .then(function() {
        // Update all existing users to have password_changed = true
        return knex('users').update({ password_changed: true });
      });
  };
  
  exports.down = function(knex) {
    return knex.schema
      .alterTable('users', function(table) {
        table.dropColumn('password_changed');
      });
  };