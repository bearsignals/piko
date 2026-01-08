# Piko Test Environment

## Test Repository Location

The test repository should be created at: `/tmp/piko-test`

## Setup Instructions

Run these commands to create a test environment:

```bash
# Create and initialize the test repo
mkdir -p /tmp/piko-test
cd /tmp/piko-test
git init
git config user.email "test@test.com"
git config user.name "Test"

# Create a sample docker-compose.yml
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
```

## Testing Commands

```bash
# Build piko
cd /Users/gwuah/Desktop/piko
go build ./cmd/piko

# Test init
cd /tmp/piko-test
/Users/gwuah/Desktop/piko/piko init

# Test create
/Users/gwuah/Desktop/piko/piko create test-env

# Test env
/Users/gwuah/Desktop/piko/piko env test-env
/Users/gwuah/Desktop/piko/piko env test-env --json

# Check containers
docker ps --filter "name=piko-piko-test"

# Cleanup
cd /tmp/piko-test/.piko/worktrees/test-env
docker compose -p piko-piko-test-test-env down
```

## Expected Port Allocation

For environment ID=1:
- Base port: `10000 + (1 * 100) = 10100`
- db:5432 → `10100 + (5432 % 100) = 10132`
- redis:6379 → `10100 + (6379 % 100) = 10179`
