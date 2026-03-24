# swarm-compute

**Author:** Yao Li ([@BH3GEI](https://github.com/BH3GEI))

A production-grade distributed computing framework featuring RabbitMQ-based job queuing, task splitting, parallel execution across worker nodes, result aggregation, and fault tolerance.

## Architecture

```
Client (submit job)
    │
    ▼
Center ──► RabbitMQ (durable job queue)
    │           │
    │     ◄─────┘ (consume)
    ▼
JobRunner (goroutine pool)
    │
    ├── Splitter → N sub-tasks
    ├── Health Check (ping workers)
    ├── Dispatch to healthy workers
    │
    ▼
┌──────────┬──────────┬──────────┬──────────┬──────────┐
│ Worker-1 │ Worker-2 │ Worker-3 │ Worker-4 │ Worker-5 │
│ /execute │ /execute │ /execute │ /execute │ /execute │
└────┬─────┴────┬─────┴────┬─────┴────┬─────┴────┬─────┘
     └──────────┴──────────┴──────────┘
                    │
              Aggregator → Result
```

## Components

| Service | Port | Description |
|---------|------|-------------|
| RabbitMQ | 5672 / 15672 | Message broker for job queue (management UI at :15672) |
| Center | 8080 | Scheduler, splitter, aggregator, API server |
| Client | 8082 | Web UI: job submission, progress tracking, visualization |
| Platform | 8081 | Admin UI: task type registry |
| Worker 1-5 | 9000 | Stateless computation nodes |

## 7 Built-in Task Types

| Type | Input | Description |
|------|-------|-------------|
| `word-count` | Text | MapReduce-style word frequency counting |
| `sort` | Number array | Distributed merge-sort |
| `matrix-mul` | Matrices A×B | Row-block partitioned multiplication |
| `pi-estimate` | Sample count | Monte Carlo Pi estimation |
| `grep` | Text + regex | Distributed text search |
| `prime-count` | Number range | Segmented prime counting |
| `hash-crack` | MD5 hash + charset | Brute-force hash collision search |

## Key Features

- **RabbitMQ job queue**: Durable message queue with persistence and delivery guarantees
- **Async processing**: Submit returns immediately; poll for status
- **Job cancellation**: Cancel running jobs via API or UI
- **Worker health checks**: Ping before dispatch, skip unhealthy nodes
- **Fault tolerance**: Automatic retry on different workers
- **Rate limiting**: Token bucket per IP
- **Graceful shutdown**: SIGINT/SIGTERM → drain queue → save state
- **Structured JSON logging**: Request ID propagation
- **Atomic persistence**: Write-to-tmp + rename
- **Non-root containers**: Security hardened Docker images
- **Docker healthchecks**: All services monitored

## Quick Start

```bash
docker compose build --no-cache
docker compose up -d
docker compose ps  # verify all healthy (rabbitmq starts first)
```

## URLs

- **Submit jobs**: http://localhost:8082/job-submit.html
- **RabbitMQ Management**: http://localhost:15672 (guest/guest)
- **Platform admin**: http://localhost:8081/task-types.html
- **Center API**: http://localhost:8080

## API

```bash
# Submit job (async — returns jobId immediately)
curl -X POST http://localhost:8080/api/jobs/submit \
  -H "Content-Type: application/json" \
  -d '{"typeId":"sort","input":{"data":[45,12,89,3,67]}}'

# Poll job status
curl http://localhost:8080/api/jobs/{jobId}

# Cancel job
curl -X POST http://localhost:8080/api/jobs/{jobId}/cancel

# List workers / stats / task types
curl http://localhost:8080/api/workers
curl http://localhost:8080/api/stats
curl http://localhost:8080/api/task-types
```

## Documentation

| Document | Description |
|----------|-------------|
| [Project Report (PDF)](report/report.pdf) | Full report: architecture, RabbitMQ design, benchmarks, testing |
| [Project Report (Markdown)](report/report.md) | Same content in Markdown |
| [Architecture & Design](docs/design.md) | Detailed system design, data flow, concurrency model |
| [Benchmark Results](benchmark/results.md) | Speedup analysis across 1–5 workers with analysis |
| [Unit Test Records](report/test-records.txt) | 29 unit test cases output |
| [Integration Test Records](report/integration-test.txt) | 11 integration test cases (all pass) |
| [Benchmark Script](benchmark/bench.py) | Reproducible benchmark runner |

## Tests

```bash
# Unit tests (29 cases)
cd center/server && go test ./...
cd site/server && go test ./...

# Integration tests (requires docker compose up)
# See report/integration-test.txt for latest results
```
