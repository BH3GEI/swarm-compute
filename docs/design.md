# swarm-compute — Architecture & Design Document

**Author:** Yao Li ([@BH3GEI](https://github.com/BH3GEI))

## 1. System Overview

swarm-compute is a distributed computing framework that splits large computation
jobs into sub-tasks, dispatches them to a pool of worker nodes for parallel execution,
and aggregates the results back into a single output.

```
                            ┌─────────────────────────┐
                            │       Client (UI)        │
                            │  http://localhost:8082   │
                            └────────────┬────────────┘
                                         │ POST /api/jobs/submit
                                         │ GET  /api/jobs/{id}
                                         ▼
                            ┌─────────────────────────┐
                            │     Center (Scheduler)   │
                            │  http://localhost:8080   │
                            │                         │
                            │  ┌───────────────────┐  │
                            │  │    Job Runner      │  │
                            │  │  (goroutine pool)  │  │
                            │  └───────┬───────────┘  │
                            │          │               │
                            │  ┌───────▼───────────┐  │
                            │  │    Splitter        │  │
                            │  │  (split by type)   │  │
                            │  └───────┬───────────┘  │
                            │          │               │
                            │  ┌───────▼───────────┐  │
                            │  │   Health Check     │  │
                            │  │  (ping workers)    │  │
                            │  └───────┬───────────┘  │
                            └──────────┼──────────────┘
                     ┌─────────┬───────┼───────┬─────────┐
                     ▼         ▼       ▼       ▼         ▼
               ┌──────────┐ ┌────┐ ┌────┐ ┌────┐ ┌──────────┐
               │ Worker-1 │ │ W2 │ │ W3 │ │ W4 │ │ Worker-5 │
               │  :9000   │ │    │ │    │ │    │ │  :9000   │
               └────┬─────┘ └──┬─┘ └──┬─┘ └──┬─┘ └────┬─────┘
                    │          │      │      │         │
                    ▼          ▼      ▼      ▼         ▼
               [execute]  [execute] [execute] [execute] [execute]
                    │          │      │      │         │
                    └──────────┴──────┴──────┴─────────┘
                                      │
                            ┌─────────▼───────────┐
                            │     Aggregator       │
                            │  (merge by type)     │
                            └─────────┬───────────┘
                                      │
                                      ▼
                               Final Result
```

## 2. Components

### 2.1 Center (Scheduler)

The central coordinator responsible for:

- **Job lifecycle management**: Create → Split → Dispatch → Aggregate → Done
- **Worker registry**: Maintains list of available workers from `WORKER_ADDRS` env var
- **Health checking**: Pings workers before dispatch, skips unhealthy nodes
- **RabbitMQ integration**: Jobs are published to a durable RabbitMQ queue and consumed by a goroutine pool (JobRunner)
- **Persistence**: State saved to `/data/store.json` using atomic write (tmp + rename)

**Key source files:**

| File | Responsibility |
|------|---------------|
| `main.go` | HTTP server, routing, graceful shutdown |
| `scheduler.go` | Job execution pipeline, worker dispatch, retry logic |
| `mq.go` | RabbitMQ connection, publish, consume |
| `runner.go` | Job consumer pool — reads from RabbitMQ, processes concurrently |
| `splitter.go` | Task splitting strategies for each task type |
| `aggregator.go` | Result merging strategies for each task type |
| `store.go` | In-memory state with RWMutex + JSON persistence |
| `middleware.go` | Request logging, rate limiting, body size limits |
| `logger.go` | Structured JSON logging |

### 2.2 Worker

Stateless compute nodes that execute individual tasks. Each worker:

- Exposes `GET /ping` for health checks
- Exposes `POST /execute` to run tasks
- Routes tasks to the appropriate executor based on `typeId`
- Returns `{status, output, durationMs}`

Workers are identical and interchangeable — any worker can execute any task type.

### 2.3 Client (Web UI)

Single-page application (`job-submit.html`) that provides:

- Task type selection with pre-built input templates
- Async job submission with real-time polling
- Progress bar and per-task status cards
- Execution timeline (Gantt chart) showing parallel worker utilization
- Result visualization (bar charts for word-count, large-number display for pi/primes)
- Job cancellation
- System stats dashboard

### 2.4 Platform (Admin UI)

Simple admin page (`task-types.html`) displaying registered task types.

## 3. Data Flow

### 3.1 Job Submission (Async)

```
Client                    Center                        Workers
  │                         │                              │
  │  POST /api/jobs/submit  │                              │
  │ ──────────────────────► │                              │
  │  ◄── {jobId, "pending"} │                              │
  │                         │  [JobRunner picks from queue] │
  │                         │                              │
  │  GET /api/jobs/{id}     │  [1. Split input → N tasks]  │
  │ ──────────────────────► │                              │
  │  ◄── {status:"running"} │  [2. Ping /ping each worker] │
  │                         │  ◄──────────────────────────►│
  │                         │                              │
  │                         │  [3. POST /execute to each]  │
  │                         │  ────────────────────────────►│
  │                         │  ◄── {output, durationMs}    │
  │                         │                              │
  │                         │  [4. Aggregate outputs]      │
  │  GET /api/jobs/{id}     │                              │
  │ ──────────────────────► │                              │
  │  ◄── {status:"done",    │                              │
  │       result: {...}}    │                              │
```

### 3.2 Key Design: Submit Returns Immediately

The `POST /api/jobs/submit` endpoint creates the job record and publishes the job ID
to a durable RabbitMQ queue, then returns `{status: "pending"}` immediately (typically <10ms).
The JobRunner consumers pick jobs from the queue and process them asynchronously.
The client polls `GET /api/jobs/{id}` to track progress.

### 3.3 Why RabbitMQ?

| Feature | Go Channel | RabbitMQ |
|---------|-----------|----------|
| Persistence | Lost on crash | Durable queues survive broker restart |
| Delivery guarantee | None | At-least-once with manual ACK |
| Observability | None | Management UI at :15672 |
| Multi-consumer | Limited | Native support with fair dispatch |
| Horizontal scaling | Single process | Multiple Center instances possible |

The ~8ms overhead per job is negligible for compute-heavy workloads.

This design ensures:
- The HTTP request never times out, even for long-running jobs
- Multiple jobs can be submitted concurrently
- The client gets immediate feedback

## 4. Key Design Decisions

### 4.1 Map-Reduce Pattern

Each task type implements three functions:

```
SplitFunc:     input → [chunk1, chunk2, ..., chunkN]
ExecuteFunc:   chunk → partial_result
AggregateFunc: [partial1, partial2, ..., partialN] → final_result
```

The number of chunks equals the number of healthy workers, maximizing parallelism.
This is a simplified Map-Reduce where:
- **Map** = Split + Execute (parallel)
- **Reduce** = Aggregate (sequential, on Center)

### 4.2 Worker Health Checks

Before dispatching tasks, the scheduler pings all registered workers in parallel
(2-second timeout per ping). Only responsive workers receive tasks. This prevents:
- Sending tasks to crashed containers
- Blocking on unresponsive workers
- Wasting retry attempts

### 4.3 Fault Tolerance & Retry

```
For each task:
  attempt = 0
  while attempt <= maxRetry:
    send to worker[attempt % numWorkers]
    if success: break
    attempt++
  if all attempts failed: mark task as failed
```

On failure, the task is retried on a **different worker** (round-robin selection).
This handles transient failures and worker-specific issues. The entire job fails
only if a task exhausts all retries.

### 4.4 Concurrency Model

- **RWMutex**: Read operations (job queries, worker list) use `RLock()`, write
  operations (job creation, status updates) use `Lock()`. This allows concurrent
  reads without blocking.
- **JobRunner**: A pool of N goroutines (default 8) consuming from a buffered channel.
  Each goroutine processes one job at a time.
- **Context propagation**: Each job gets a `context.WithTimeout`. If the context
  expires or the job is cancelled, all in-flight HTTP calls to workers are aborted.
- **Goroutine leak protection**: `wg.Wait()` is wrapped in a `select` with `ctx.Done()`
  to avoid indefinite blocking.

### 4.5 Rate Limiting

Token bucket algorithm per client IP:
- General API: 50 requests/second, burst 100
- Job submission: 10 requests/second, burst 20

Prevents abuse without external dependencies (pure Go implementation).

## 5. Task Type System

### 5.1 Registry Pattern

Each task type is registered in three maps:

```go
var splitters   = map[string]SplitFunc{ ... }    // center/server/splitter.go
var executors   = map[string]ExecuteFunc{ ... }   // site/server/executor.go
var aggregators = map[string]AggregateFunc{ ... } // center/server/aggregator.go
```

### 5.2 Built-in Task Types

| Type | Split Strategy | Execution | Aggregation |
|------|---------------|-----------|-------------|
| `word-count` | Text → line chunks | Count word freq per chunk | Merge freq maps |
| `sort` | Array → equal chunks | Sort each chunk | K-way merge sort |
| `matrix-mul` | Matrix A → row blocks | Block × B multiplication | Concatenate row blocks |
| `pi-estimate` | N samples → N/W per worker | Monte Carlo random sampling | Sum inside/total, compute 4*inside/total |
| `grep` | Text → line chunks + pattern | Regex match per chunk | Merge + sort by line number |
| `prime-count` | [from,to] → sub-ranges | Trial division per range | Sum counts, merge samples |
| `hash-crack` | Charset → index ranges | Brute-force MD5 per range | Return first match |

### 5.3 Adding a New Task Type

1. Add executor in `site/server/exec_<name>.go`:
   ```go
   func executeMyTask(input map[string]interface{}) (interface{}, error) { ... }
   ```
2. Register in `site/server/executor.go`: `"my-task": executeMyTask`
3. Add splitter in `center/server/splitter.go`
4. Add aggregator in `center/server/aggregator.go`
5. Register TaskType in `center/server/main.go` `initBuiltinTaskTypes()`

## 6. Persistence

State is stored in-memory and persisted to `/data/store.json` using atomic writes:

```
1. Marshal state to JSON
2. Write to store.json.tmp
3. os.Rename(store.json.tmp, store.json)  // atomic on POSIX
```

This ensures the file is never partially written, even if the process crashes mid-write.

## 7. Docker Deployment

```yaml
center    → Go binary, port 8080, healthcheck via /api/task-types
worker-1  → Go binary, port 9000, healthcheck via /ping
worker-2  → same
...
worker-5  → same
client    → nginx serving static HTML/JS, port 8082
platform  → nginx serving admin HTML, port 8081
```

All containers run as non-root user `appuser` with CPU/memory limits.
Workers are configured via `WORKER_ADDRS` environment variable on Center.

## 8. API Reference

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/task-types` | List registered task types |
| POST | `/api/jobs/submit` | Submit a job (async) |
| GET | `/api/jobs` | List all jobs |
| GET | `/api/jobs/{id}` | Get job details + task statuses |
| POST | `/api/jobs/{id}/cancel` | Cancel a running job |
| GET | `/api/workers` | List registered workers |
| GET | `/api/stats` | System statistics |

### Job Submit Request

```json
{
  "typeId": "sort",
  "input": {"data": [5, 3, 1, 4, 2]},
  "maxRetry": 2,
  "timeoutSec": 60
}
```

### Job Response

```json
{
  "jobId": "job_abc123",
  "typeId": "sort",
  "status": "done",
  "tasks": [
    {"taskId": "task_1", "workerId": "worker-1", "status": "done", "durationMs": 5},
    {"taskId": "task_2", "workerId": "worker-2", "status": "done", "durationMs": 3}
  ],
  "result": [1, 2, 3, 4, 5],
  "createdAt": 1711270000000,
  "finishedAt": 1711270000050
}
```
