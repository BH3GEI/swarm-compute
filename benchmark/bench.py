#!/usr/bin/env python3
"""
swarm-compute benchmark — measures speedup across different worker counts.

Usage:
    python3 benchmark/bench.py

Prerequisites:
    docker compose up -d  (all 5 workers running)

Uses PUT /api/workers to dynamically change the active worker pool
without restarting any containers.
"""

import json
import time
import urllib.request
import random
import sys
import os

CENTER = "http://localhost:8080"
ALL_WORKERS = [
    {"id": "worker-1", "addr": "http://worker-1:9000"},
    {"id": "worker-2", "addr": "http://worker-2:9000"},
    {"id": "worker-3", "addr": "http://worker-3:9000"},
    {"id": "worker-4", "addr": "http://worker-4:9000"},
    {"id": "worker-5", "addr": "http://worker-5:9000"},
]

def api(method, path, data=None):
    url = CENTER + path
    for attempt in range(3):
        try:
            if data is not None:
                req = urllib.request.Request(
                    url, json.dumps(data).encode(),
                    headers={"Content-Type": "application/json"}, method=method
                )
            else:
                req = urllib.request.Request(url, method=method)
            resp = urllib.request.urlopen(req, timeout=30)
            return json.loads(resp.read())
        except Exception as e:
            if attempt < 2:
                time.sleep(1)
            else:
                raise

def submit_and_wait(type_id, inp, timeout=120):
    job = api("POST", "/api/jobs/submit", {
        "typeId": type_id, "input": inp, "timeoutSec": timeout
    })
    job_id = job["jobId"]
    for _ in range(timeout * 4):
        time.sleep(0.25)
        try:
            job = api("GET", f"/api/jobs/{job_id}")
        except:
            continue
        if job["status"] in ("done", "failed", "cancelled"):
            break
    return job

def set_worker_count(n):
    """Dynamically set active workers via API."""
    workers = ALL_WORKERS[:n]
    api("PUT", "/api/workers", {"workers": workers})
    time.sleep(0.5)  # Brief pause for health checks

def run_benchmark(name, type_id, inp, worker_counts):
    print(f"\n{'='*60}")
    print(f"  Benchmark: {name}")
    print(f"{'='*60}")

    results = {}
    for n in worker_counts:
        print(f"  Workers={n}...", end=" ", flush=True)
        set_worker_count(n)

        times = []
        for trial in range(3):
            job = submit_and_wait(type_id, inp)
            if job["status"] != "done":
                print(f"FAILED: {job.get('error', '?')}")
                times.append(float('inf'))
                continue
            elapsed = job["finishedAt"] - job["createdAt"]
            times.append(elapsed)

        times.sort()
        median = times[1]  # median of 3
        results[n] = median
        print(f"{median}ms (trials: {times})")

    return results

def print_table(name, results):
    baseline = results.get(1, 1) or 1
    print(f"\n  {name}")
    print(f"  {'Workers':<10} {'Time (ms)':<12} {'Speedup':<10} {'Efficiency':<10}")
    print(f"  {'-'*42}")
    for n in sorted(results.keys()):
        t = results[n]
        speedup = baseline / t if t > 0 else 0
        eff = speedup / n * 100 if n > 0 else 0
        print(f"  {n:<10} {t:<12.0f} {speedup:<10.2f} {eff:<9.1f}%")

def main():
    print("=" * 60)
    print("  swarm-compute Benchmark Suite")
    print("=" * 60)

    try:
        api("GET", "/api/task-types")
    except Exception as e:
        print(f"ERROR: Center not reachable: {e}")
        print("Run 'docker compose up -d' first.")
        sys.exit(1)

    worker_counts = [1, 2, 3, 4, 5]

    # Benchmark 1: Sort 100K
    random.seed(42)
    sort_data = [random.randint(1, 10000000) for _ in range(100000)]
    sort_results = run_benchmark(
        "Distributed Sort (100K numbers)", "sort", {"data": sort_data}, worker_counts
    )

    # Benchmark 2: Pi 10M
    pi_results = run_benchmark(
        "Monte Carlo Pi (10M samples)", "pi-estimate", {"samples": 10000000}, worker_counts
    )

    # Benchmark 3: Prime [1, 1M]
    prime_results = run_benchmark(
        "Prime Count [1, 1000000]", "prime-count", {"from": 1, "to": 1000000}, worker_counts
    )

    # Restore all workers
    set_worker_count(5)

    # Summary
    print("\n" + "=" * 60)
    print("  BENCHMARK RESULTS SUMMARY")
    print("=" * 60)

    all_results = {
        "Distributed Sort (100K)": sort_results,
        "Monte Carlo Pi (10M)": pi_results,
        "Prime Count [1, 1M]": prime_results,
    }

    for name, results in all_results.items():
        print_table(name, results)

    # Write results.md
    rpath = os.path.join(os.path.dirname(os.path.abspath(__file__)), "results.md")
    with open(rpath, "w") as f:
        f.write("# swarm-compute Benchmark Results\n\n")
        f.write(f"**Date:** {time.strftime('%Y-%m-%d %H:%M:%S')}\n\n")

        for name, results in all_results.items():
            baseline = results.get(1, 1) or 1
            f.write(f"## {name}\n\n")
            f.write("| Workers | Time (ms) | Speedup | Efficiency |\n")
            f.write("|---------|-----------|---------|------------|\n")
            for n in sorted(results.keys()):
                t = results[n]
                sp = baseline / t if t > 0 else 0
                eff = sp / n * 100 if n > 0 else 0
                f.write(f"| {n} | {t:.0f} | {sp:.2f}x | {eff:.1f}% |\n")
            f.write("\n")

        f.write("## Speedup Chart\n\n```\n")
        f.write(f"{'Workers':<10}")
        for name in all_results:
            short = name.split("(")[0].strip()[:18]
            f.write(f"{short:<20}")
        f.write("\n")
        for n in worker_counts:
            f.write(f"{n:<10}")
            for name, results in all_results.items():
                baseline = results.get(1, 1) or 1
                t = results.get(n, baseline)
                sp = baseline / t if t > 0 else 0
                bar = "#" * int(sp * 5)
                f.write(f"{sp:.2f}x {bar:<14}")
            f.write("\n")
        f.write("```\n\n")

        f.write("### Analysis\n\n")
        f.write("- **Sort**: Network overhead from transferring 100K numbers limits speedup at higher worker counts.\n")
        f.write("- **Pi estimate**: Embarrassingly parallel workload — near-linear speedup expected.\n")
        f.write("- **Prime count**: CPU-bound with minimal data transfer — good parallelism.\n")
        f.write("- **Efficiency**: Percentage of ideal linear speedup (100% = perfect scaling).\n")
        f.write("- Overhead includes: HTTP serialization, health checks, task splitting, result aggregation.\n")

    print(f"\n  Results written to: {rpath}")
    print("  Done!")

if __name__ == "__main__":
    main()
