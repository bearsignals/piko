#!/bin/bash
# Setup script for piko test environment
# Test repo location: /tmp/piko-test

set -e

TEST_DIR="/tmp/piko-test"

# Clean up if exists
rm -rf "$TEST_DIR"

# Create and initialize
mkdir -p "$TEST_DIR"
cd "$TEST_DIR"
git init
git config user.email "test@test.com"
git config user.name "Test"

# Create docker-compose.yml
cat > docker-compose.yml << 'EOF'
services:
  db:
    image: postgres:15-alpine
    environment:
      POSTGRES_PASSWORD: test
    ports:
      - "5432"

  redis:
    image: redis:7-alpine
    ports:
      - "6379"
EOF

# Create initial commit
echo "# Test Project" > README.md
git add .
git commit -m "Initial commit"

echo "✓ Test environment created at $TEST_DIR"
