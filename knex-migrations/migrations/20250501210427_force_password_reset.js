exports.up = function(knex) {
    // Update all existing users to have password_changed = false
    return knex('users').update({ password_changed: false });
  };
  
  exports.down = function(knex) {
    // Restore default password_changed = true
    return knex('users').update({ password_changed: true });
  };