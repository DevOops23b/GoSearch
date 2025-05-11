const knexSqlite = require('knex')({
    client: 'better-sqlite3',
    connection: {
      filename: process.env.SQLITE_DB_PATH || '/app/src/whoknows.db' // ændre til '/app/src/whoknows.db' i docker og './src/whoknows.db' hvis det er lokalt.
    },
    useNullAsDefault: true
  });
  
  const knexPg = require('knex')({
    client: 'pg',
    connection: {
      host: process.env.DB_HOST || 'postgres', //ændre her 'localhost/postgres' alt efter om du kører filen lokalt eller fra docker
      user: process.env.DB_USER || 'youruser',
      password: process.env.DB_PASSWORD || 'yourpassword',
      database: process.env.DB_NAME || 'whoknows'
    }
  });
  
  async function migrate() {
    const users = await knexSqlite('users');
    for (const user of users) {
      await knexPg('users').insert(user).onConflict('id').ignore();
    }
  
    const pages = await knexSqlite('pages');
    for (const page of pages) {
      await knexPg('pages').insert(page).onConflict('url').ignore();
    }

    //This to update users id - to max id. Now the next user wont get a already used id
    await knexPg.raw(`SELECT setval('users_id_seq', (SELECT MAX(id) FROM users));`);
  
    console.log('Migration complete');
    process.exit(0);
  }
  
  migrate();
  