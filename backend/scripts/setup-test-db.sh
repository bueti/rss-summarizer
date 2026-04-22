#!/bin/bash

# Setup test database for integration tests

set -e

echo "Setting up test database..."

# Configuration
DB_HOST="${DB_HOST:-localhost}"
DB_PORT="${DB_PORT:-5432}"
DB_USER="${DB_USER:-rss_user}"
DB_PASS="${DB_PASS:-rss_pass}"
TEST_DB="rss_summarizer_test"

# Check if PostgreSQL is running
if ! pg_isready -h "$DB_HOST" -p "$DB_PORT" > /dev/null 2>&1; then
    echo "Error: PostgreSQL is not running on $DB_HOST:$DB_PORT"
    echo "Please start PostgreSQL first (e.g., docker-compose up postgres)"
    exit 1
fi

# Drop test database if it exists
echo "Dropping existing test database (if any)..."
PGPASSWORD=$DB_PASS psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d postgres -c "DROP DATABASE IF EXISTS $TEST_DB;" 2>/dev/null || true

# Create test database
echo "Creating test database: $TEST_DB"
PGPASSWORD=$DB_PASS psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d postgres -c "CREATE DATABASE $TEST_DB;"

# Run migrations on test database
echo "Running migrations..."
cd "$(dirname "$0")/.."
DATABASE_URL="postgres://$DB_USER:$DB_PASS@$DB_HOST:$DB_PORT/$TEST_DB?sslmode=disable" go run cmd/migrate/main.go up

echo "✓ Test database setup complete!"
echo ""
echo "To run tests:"
echo "  go test ./internal/api/handlers/..."
echo ""
echo "Connection string for tests:"
echo "  postgres://$DB_USER:$DB_PASS@$DB_HOST:$DB_PORT/$TEST_DB?sslmode=disable"
