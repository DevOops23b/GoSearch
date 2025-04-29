module.exports = {
    development: {
      client: 'pg',
      connection: {
        host: process.env.DB_HOST || 'localhost',
        port: process.env.DB_PORT || 5432,
        user: process.env.DB_USER || 'youruser',
        password: process.env.DB_PASSWORD || 'yourpassword',
        database: process.env.DB_NAME || 'whoknows'
      },
      migrations: {
        tableName: 'knex_migrations',
        directory: './migrations'
      }
    },
    docker: {
      client: 'pg',
      connection: {
        host: process.env.DB_HOST || 'postgres',
        port: process.env.DB_PORT || 5432,
        user: process.env.DB_USER || 'youruser',
        password: process.env.DB_PASSWORD || 'yourpassword',
        database: process.env.DB_NAME || 'whoknows'
      },
      migrations: {
        tableName: 'knex_migrations',
        directory: './migrations'
      }
    }
  };