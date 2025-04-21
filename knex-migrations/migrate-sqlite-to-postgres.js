const knexSqlite = require('knex')({
    client: 'better-sqlite3',
    connection: {
      filename: process.env.SQLITE_DB_PATH || '/app/src/whoknows.db'
    },
    useNullAsDefault: true
  });
  
  const knexPg = require('knex')({
    client: 'pg',
    connection: {
      host: process.env.DB_HOST || 'postgres',
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
      await knexPg('pages').insert(page).onConflict('title').ignore();
    }
  
    console.log('Migration complete');
    process.exit(0);
  }
  
  migrate();
  