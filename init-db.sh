#!/bin/bash
set -e

echo "Initializing PostgreSQL database..."

# Perform all actions as the 'postgres' user
export PGUSER=postgres

# Get the IP address of the PostgreSQL container
POSTGRES_IP=$(hostname -i)

# Set the authentication method for local connections
export PGHOST=$POSTGRES_IP

# Wait for PostgreSQL to become available
until pg_isready -q -h $POSTGRES_IP -p 5432 -U $PGUSER; do
  echo "$(date) - waiting for database to start"
  sleep 2
done

echo "PostgreSQL is ready."

# Set the default value of DB_NAME to 'postgres' if not already set
DB_NAME=${DB_NAME:-postgres}

# Create the database if it doesn't exist
echo "Creating database $DB_NAME..."
psql -h $POSTGRES_IP -U $PGUSER -c "CREATE DATABASE $DB_NAME"

# Create the tables
echo "Creating tables..."
psql -h $POSTGRES_IP -U $PGUSER -d "$DB_NAME" -c "
CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    email VARCHAR(255) UNIQUE NOT NULL,
    password VARCHAR(255) NOT NULL
);

CREATE TABLE IF NOT EXISTS urls (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
    original_url TEXT NOT NULL,
    shortened_url VARCHAR(255) NOT NULL,
    visit_count INTEGER DEFAULT 0,
    UNIQUE(user_id, original_url),
    UNIQUE(shortened_url)
);"

echo "Database initialized!"