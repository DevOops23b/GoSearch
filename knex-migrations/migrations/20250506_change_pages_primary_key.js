exports.up = async function(knex) {
    await knex.schema.alterTable('pages', function(table) {
      table.dropPrimary();
      table.dropUnique(['url']);
      table.primary(['url']);
    });
};

exports.down = async function(knex) {
  await knex.schema.alterTable('pages', function(table) {
    table.dropPrimary();
    table.primary(['title']);
    table.unique(['url']);
  });
};
  