#!/bin/bash
set -e

# Perform all actions as the 'postgres' user
export PGUSER=postgres

# Wait for PostgreSQL to become available.
echo "Waiting for PostgreSQL to start..."
while ! pg_isready -q -h localhost -p 5432 -U $PGUSER
do
  echo "$(date) - waiting for database to start"
  sleep 2
done

# Run the migration scripts
for file in /migrations/*.sql
do
  echo "Running $file..."
  psql -h localhost -U $PGUSER -d $DB_NAME -a -f "$file"
done

echo "Database initialized!"