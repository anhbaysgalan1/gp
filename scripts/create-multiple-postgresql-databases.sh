#!/bin/bash

set -e
set -u

function create_user_and_database() {
	local database=$1
	echo "  Creating user and database '$database'"
	psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname="$POSTGRES_DB" <<-EOSQL
	    SELECT 'CREATE DATABASE $database' WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = '$database')\\gexec
	    DO
	    \$\$BEGIN
	        CREATE USER $database WITH PASSWORD '$POSTGRES_PASSWORD';
	        EXCEPTION WHEN duplicate_object THEN RAISE NOTICE '%, skipping', SQLERRM USING ERRCODE = SQLSTATE;
	    END\$\$;
	    GRANT ALL PRIVILEGES ON DATABASE $database TO $database;
	    ALTER USER $database CREATEDB;
EOSQL

	# Connect to the specific database and grant schema permissions
	echo "  Granting schema permissions for '$database'"
	psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname="$database" <<-EOSQL
	    GRANT ALL ON SCHEMA public TO $database;
	    GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO $database;
	    GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO $database;
	    GRANT ALL PRIVILEGES ON ALL FUNCTIONS IN SCHEMA public TO $database;
	    ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON TABLES TO $database;
	    ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON SEQUENCES TO $database;
	    ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON FUNCTIONS TO $database;
EOSQL
}

if [ -n "${POSTGRES_MULTIPLE_DATABASES:-}" ]; then
	echo "Multiple database creation requested: $POSTGRES_MULTIPLE_DATABASES"
	for db in $(echo $POSTGRES_MULTIPLE_DATABASES | tr ',' ' '); do
		create_user_and_database $db
	done
	echo "Multiple databases created"
fi