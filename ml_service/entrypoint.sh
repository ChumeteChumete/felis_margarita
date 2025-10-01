#!/bin/bash
set -e

echo "Waiting for PostgreSQL..."
until PGPASSWORD=${POSTGRES_PASSWORD:-pass} psql -h postgres -U ${POSTGRES_USER:-app} -d ${POSTGRES_DB:-appdb} -c '\q' 2>/dev/null; do
  sleep 1
done
echo "PostgreSQL is ready"

echo "Running migrations..."
PGPASSWORD=${POSTGRES_PASSWORD:-pass} psql -h postgres -U ${POSTGRES_USER:-app} -d ${POSTGRES_DB:-appdb} < /migrations/0001_init.sql
echo "Migrations complete"

echo "Starting ML service..."
exec python server.py