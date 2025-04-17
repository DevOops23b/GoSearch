module.exports = {
    development: {
      client: 'pg',
      connection: {
        host: 'localhost',
        port: 5432,
        user: 'youruser',
        password: 'yourpassword',
        database: 'whoknows'
      },
      migrations: {
        tableName: 'knex_migrations',
        directory: './migrations'
      }
    },
    docker: {
      client: 'pg',
      connection: {
        host: 'postgres',
        port: 5432,
        user: 'youruser',
        password: 'yourpassword',
        database: 'whoknows'
      },
      migrations: {
        tableName: 'knex_migrations',
        directory: './migrations'
      }
    }
  };