# Scout Quick Start Guide

## 🚀 Get Started in 30 Seconds

```bash
# 1. Start Scout with Docker Compose
make up

# 2. Add your Postman collections
cp your-collection.json collections/

# 3. Open the dashboard
open http://localhost:8080
```

That's it! Scout is now monitoring your APIs.

## 📋 Common Commands

### Starting and Stopping

```bash
make up          # Start Scout and PostgreSQL
make down        # Stop all services
make restart     # Restart services
make clean       # Stop and remove everything (including data)
```

### Viewing Logs

```bash
make logs        # All logs
make logs-scout  # Scout only
make logs-db     # PostgreSQL only
```

### Rebuilding

```bash
make build           # Rebuild Docker images
make rebuild-scout   # Rebuild only Scout service
```

## 🔗 Access Points

| Service | URL |
|---------|-----|
| **Dashboard** | http://localhost:8080 |
| **API** | http://localhost:8080/api/results |
| **Metrics** | http://localhost:8080/metrics |
| **PostgreSQL** | localhost:5432 (scout/postgres/postgres) |

## 📁 Adding Collections

Just drop your Postman collection JSON files into the `collections/` directory:

```bash
collections/
├── example-api.postman_collection.json      # Included example
├── your-api.postman_collection.json         # Your tests
└── another-api.postman_collection.json      # More tests
```

Scout automatically discovers and runs all `.json` files in this directory.

## ⚙️ Configuration

Edit `docker-compose.yaml` to customize:

```yaml
environment:
  INTERVAL: "60s"        # Change test frequency (e.g., 30s, 2m, 5m)
  PORT: "8080"           # Change web server port
```

Then restart:

```bash
make restart
```

## 🔍 Debugging

### View Database

```bash
make db-connect

# Then run SQL queries:
SELECT * FROM collections;
SELECT * FROM latest_test_executions;
```

### Test Newman Manually

```bash
make test
# Or
cd newman && node executor.js ../collections/your-collection.json
```

### View Container Status

```bash
make status
docker-compose ps
```

## 📊 Prometheus + Grafana Setup

Scout exposes metrics at `/metrics`. To visualize in Grafana:

1. Add Prometheus scrape config:
```yaml
scrape_configs:
  - job_name: 'scout'
    static_configs:
      - targets: ['localhost:8080']
```

2. In Grafana, add Prometheus data source

3. Query examples:
```promql
# Test pass rate
sum(scout_test_status) / count(scout_test_status) * 100

# Average latency
avg(scout_test_latency_ms) by (collection)

# Failed tests
scout_test_status{} == 0
```

## 🐛 Troubleshooting

### "Port already in use"
```bash
# Change the port in docker-compose.yaml
ports:
  - "8081:8080"  # Use 8081 instead
```

### "Can't connect to database"
```bash
# Check if PostgreSQL is running
make logs-db

# Restart PostgreSQL
docker-compose restart postgres
```

### "No collections found"
```bash
# Verify collections directory
ls -la collections/

# Ensure files are .json
mv collection.txt collection.json
```

## 💡 Tips

- **Auto-refresh**: The UI refreshes every 30 seconds
- **Manual trigger**: Click "Run Tests Now" for immediate execution
- **History**: All test results are stored in PostgreSQL
- **Cleanup**: Use `make clean` to reset everything

## 📚 More Information

- Full documentation: [README.md](README.md)
- API endpoints: http://localhost:8080/api/results
- Example collection: `collections/example-api.postman_collection.json`

## 🎯 Next Steps

1. **Add your collections** to the `collections/` directory
2. **Configure test interval** in `docker-compose.yaml`
3. **Set up Grafana** for advanced visualization
4. **Deploy to Kubernetes** using manifests in `deployments/kubernetes/`

---

**Need help?** Check the [README.md](README.md) or open an issue on GitHub.
