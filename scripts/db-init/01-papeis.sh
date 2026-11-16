#!/bin/sh
set -eu

: "${APP_MIGRATOR_PASSWORD:?defina APP_MIGRATOR_PASSWORD}"
: "${APP_RW_PASSWORD:?defina APP_RW_PASSWORD}"

psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" <<'SQL'
\getenv migrator_password APP_MIGRATOR_PASSWORD
\getenv rw_password APP_RW_PASSWORD

CREATE ROLE app_migrator LOGIN PASSWORD :'migrator_password'
    NOSUPERUSER NOCREATEDB NOCREATEROLE;
CREATE ROLE app_rw LOGIN PASSWORD :'rw_password'
    NOSUPERUSER NOCREATEDB NOCREATEROLE;

ALTER ROLE app_rw SET statement_timeout = '5s';

GRANT CREATE ON SCHEMA public TO app_migrator;
SQL