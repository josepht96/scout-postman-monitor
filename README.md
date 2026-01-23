# Scout - Postman Test Monitoring

Scout is a monitoring tool that executes Postman collections using Newman, stores test results in PostgreSQL, exposes Prometheus metrics, and provides a simple web UI for visualization.

## Features

- **Automated Test Execution**: Continuously runs Postman tests on a configurable schedule
- **Newman Integration**: Uses the Newman Node.js library to execute Postman collections
- **PostgreSQL Storage**: Stores both historical and latest test results
- **Prometheus Metrics**: Exposes test status and latency metrics for Grafana
- **Web Dashboard**: Simple, auto-refreshing UI to view test results
- **REST API**: JSON API for programmatic access to test results
- **Kubernetes Ready**: Includes Docker and Kubernetes deployment manifests

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                         Scout                                │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  ┌──────────┐      ┌──────────┐      ┌──────────────┐      │
│  │ Watcher  │─────▶│ Scheduler│─────▶│   Newman     │      │
│  │          │      │          │      │  (Node.js)   │      │
│  └──────────┘      └──────────┘      └──────────────┘      │
│       │                  │                   │              │
│       │                  ▼                   ▼              │
│       │            ┌──────────┐      ┌──────────────┐      │
│       │            │ Storage  │◀─────│  Executor    │      │
│       │            │(Postgres)│      │              │      │
│       │            └──────────┘      └──────────────┘      │
│       │                  │                                  │
│       │                  ▼                                  │
│       │            ┌──────────┐                            │
│       └───────────▶│  Metrics │                            │
│                    │(Prometheus)                           │
│                    └──────────┘                            │
│                          │                                  │
│                          ▼                                  │
│                    ┌──────────┐                            │
│                    │   API    │                            │
│                    │  Server  │                            │
│                    └──────────┘                            │
│                          │                                  │
└──────────────────────────┼──────────────────────────────────┘
                           │
                           ▼
                    ┌──────────────┐
                    │   Web UI     │
                    │   Grafana    │
                    └──────────────┘
```

## Prerequisites

- **Go 1.21+** - For building the application
- **Node.js 18+** - For running Newman
- **PostgreSQL 15+** - For data storage
- **Docker** (optional) - For containerized deployment
- **Kubernetes** (optional) - For cluster deployment

## Quick Start

### Option 1: Docker Compose (Recommended for Local Testing)

The fastest way to get Scout running locally:

```bash
# Start PostgreSQL and Scout
make up

# Or without make:
docker-compose up -d

# View logs
make logs

# Stop services
make down

# Clean up everything (including database)
make clean
```

Scout will be available at:
- **Web UI**: http://localhost:8080
- **API**: http://localhost:8080/api/results
- **Metrics**: http://localhost:8080/metrics

Add your Postman collections to the `collections/` directory and Scout will automatically pick them up!

### Option 2: Manual Installation

### 1. Install Dependencies

```bash
# Install Go dependencies
go mod download

# Install Newman dependencies
cd newman && npm install && cd ..
```

### 2. Setup PostgreSQL

```bash
# Using Docker
docker run -d \
  --name scout-postgres \
  -e POSTGRES_DB=scout \
  -e POSTGRES_USER=postgres \
  -e POSTGRES_PASSWORD=postgres \
  -p 5432:5432 \
  postgres:15-alpine

# Or use your existing PostgreSQL instance
```

### 3. Configure Environment

```bash
export DATABASE_URL="postgres://postgres:postgres@localhost:5432/scout?sslmode=disable"
export COLLECTIONS_DIR="./collections"
export INTERVAL="60s"
export PORT="8080"
```

### 4. Add Postman Collections

Place your Postman collection JSON files in the `collections/` directory:

```bash
cp your-collection.postman_collection.json collections/
```

An example collection is provided in `collections/example-api.postman_collection.json`.

### 5. Run Scout

```bash
# Build and run
go build -o scout ./cmd/scout
./scout

# Or run directly
go run ./cmd/scout
```

Scout will start on `http://localhost:8080`

## Usage

### Web Interface

Open your browser to `http://localhost:8080` to view the dashboard.

### API Endpoints

- `GET /` - Web UI
- `GET /health` - Health check
- `GET /metrics` - Prometheus metrics
- `GET /api/results` - Latest test results (JSON)
- `GET /api/collections` - List all collections (JSON)
- `GET /api/history?collection_id=1&limit=50` - Historical results (JSON)
- `GET /api/stats` - Scheduler statistics (JSON)
- `POST /api/run` - Trigger immediate test run

### Prometheus Metrics

Scout exposes the following Prometheus metrics at `/metrics`:

- `scout_test_status{collection, test_name, url, method}` - Test status (1=pass, 0=fail)
- `scout_test_latency_ms{collection, test_name, url, method}` - Response time in milliseconds
- `scout_collection_last_run_timestamp{collection}` - Last execution timestamp
- `scout_collection_duration_ms{collection}` - Collection execution duration
- `scout_collection_tests_total{collection, status}` - Total tests by status

## Docker Deployment

### Build Image

```bash
docker build -f deployments/Dockerfile -t scout:latest .
```

### Run Container

```bash
docker run -d \
  --name scout \
  -p 8080:8080 \
  -e DATABASE_URL="postgres://postgres:postgres@host.docker.internal:5432/scout?sslmode=disable" \
  -e INTERVAL="60s" \
  -v $(pwd)/collections:/app/collections \
  scout:latest
```

## Kubernetes Deployment

### Prerequisites

- Kubernetes cluster running
- kubectl configured
- PostgreSQL instance (or deploy using provided manifest)

### Deploy

```bash
# Deploy PostgreSQL (optional, for development)
kubectl apply -f deployments/kubernetes/postgres.yaml

# Deploy Scout
kubectl apply -f deployments/kubernetes/secret.yaml
kubectl apply -f deployments/kubernetes/configmap.yaml
kubectl apply -f deployments/kubernetes/deployment.yaml
kubectl apply -f deployments/kubernetes/service.yaml
```

### Access Scout

```bash
# Port forward to access locally
kubectl port-forward svc/scout 8080:80

# Or use NodePort/LoadBalancer service
# Edit deployments/kubernetes/service.yaml to uncomment desired service type
```

## Configuration

Scout is configured via environment variables:

| Variable | Description | Default |
|----------|-------------|---------|
| `DATABASE_URL` | PostgreSQL connection string | `postgres://postgres:postgres@localhost:5432/scout?sslmode=disable` |
| `COLLECTIONS_DIR` | Directory containing Postman collections | `collections` |
| `NEWMAN_SCRIPT_PATH` | Path to Newman executor script | `newman/executor.js` |
| `INTERVAL` | Test execution interval (Go duration format) | `60s` |
| `PORT` | HTTP server port | `8080` |

## Development

### Project Structure

```
scout/
├── cmd/scout/              # Main application entry point
├── internal/
│   ├── api/                # HTTP server and API handlers
│   ├── executor/           # Newman executor wrapper
│   ├── metrics/            # Prometheus metrics exporter
│   ├── scheduler/          # Test execution scheduler
│   ├── storage/            # PostgreSQL storage layer
│   └── watcher/            # Collection file watcher
├── newman/                 # Node.js Newman executor
│   ├── package.json
│   └── executor.js
├── web/                    # Static web UI
├── collections/            # Postman collections directory
├── deployments/            # Docker and K8s manifests
└── db/migrations/          # Database migrations
```

### Makefile Commands

Scout includes a Makefile with convenient commands:

```bash
# View all available commands
make help

# Docker Compose commands
make build        # Build Docker images
make up           # Start all services
make down         # Stop all services
make logs         # View logs
make logs-scout   # View Scout logs only
make logs-db      # View PostgreSQL logs only
make restart      # Restart services
make clean        # Stop and remove all volumes

# Development commands
make build-go     # Build Scout binary
make dev          # Run Scout locally (requires local Postgres)
make test         # Test Newman executor
make db-connect   # Connect to PostgreSQL CLI

# Go commands
make fmt          # Format Go code
make tidy         # Tidy Go modules
make test-go      # Run Go tests
```

### Running Tests

```bash
# Run Go tests
go test ./...
# Or with make
make test-go

# Test Newman executor
cd newman
node executor.js ../collections/example-api.postman_collection.json
# Or with make
make test
```

### Building

```bash
# Build for current platform
go build -o scout ./cmd/scout

# Build for Linux
GOOS=linux GOARCH=amd64 go build -o scout-linux ./cmd/scout

# Build Docker image
docker build -f deployments/Dockerfile -t scout:latest .
```

## Monitoring with Grafana

### Add Prometheus Data Source

1. Add Scout's `/metrics` endpoint to your Prometheus configuration
2. In Grafana, add Prometheus as a data source

### Example Prometheus Configuration

```yaml
scrape_configs:
  - job_name: 'scout'
    static_configs:
      - targets: ['localhost:8080']
    metrics_path: '/metrics'
    scrape_interval: 30s
```

### Example Grafana Queries

```promql
# Test pass rate
sum(scout_test_status) / count(scout_test_status) * 100

# Average response time per collection
avg(scout_test_latency_ms) by (collection)

# Failed tests
scout_test_status{} == 0
```

## Troubleshooting

### Node.js not found

Ensure Node.js is installed and accessible in your PATH:

```bash
node --version
```

### Database connection errors

Verify PostgreSQL is running and the connection string is correct:

```bash
psql "$DATABASE_URL" -c "SELECT 1"
```

### No collections found

Ensure `.json` files exist in the `COLLECTIONS_DIR`:

```bash
ls -la collections/
```

### Newman execution errors

Test Newman directly:

```bash
cd newman
node executor.js ../collections/example-api.postman_collection.json
```

## License

MIT

## Contributing

Contributions are welcome! Please open an issue or submit a pull request.

## Support

For issues and questions, please open a GitHub issue.
