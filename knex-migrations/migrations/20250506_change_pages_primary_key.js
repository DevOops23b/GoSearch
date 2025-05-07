exports.up = async function(knex) {
    await knex.schema.alterTable('pages', function(table) {
      table.dropPrimary();
      table.dropUnique(['url']);
      table.primary(['url']);
    });
};
  