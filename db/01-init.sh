#!/bin/bash
set -e
export PGPASSWORD=$APP_DB_PASS;
psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" <<-EOSQL
  CREATE USER $APP_DB_USER WITH PASSWORD '$APP_DB_PASS';
  CREATE DATABASE $APP_DB_NAME;
  GRANT ALL PRIVILEGES ON DATABASE $APP_DB_NAME TO $APP_DB_USER;
EOSQL

psql --username "$APP_DB_USER" --dbname "$APP_DB_NAME" -f  /docker-entrypoint-initdb.d/sql/create-tables.sql