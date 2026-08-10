# Upload/Download Load Testing

This folder contains a dependency-free load tester for Go Cloud Storage.

## Environment

Start the normal stack first: backend, MySQL, Redis, and MinIO. Then export:

```bash
export GCS_BASE_URL=http://localhost:8081
export GCS_TOKEN='your-jwt-access-token'
export GCS_DOWNLOAD_FILE_ID='file-id-owned-by-that-token-user'
export GCS_UPLOAD_FILE=/tmp/gcs-upload-1mb.bin
```

Create sample files:

```bash
dd if=/dev/urandom of=/tmp/gcs-upload-1mb.bin bs=1m count=1
dd if=/dev/urandom of=/tmp/gcs-upload-20mb.bin bs=1m count=20
```

## Runs

Download signing QPS, measuring app-server signing path:

```bash
python3 scripts/perf/gcs_load_test.py --mode download-info --requests 1000 --concurrency 50
```

Direct MinIO download throughput through backend-issued presigned URL:

```bash
python3 scripts/perf/gcs_load_test.py --mode download-direct --requests 200 --concurrency 20
```

Backend Range proxy path:

```bash
python3 scripts/perf/gcs_load_test.py --mode download-range --range-size 1048576 --requests 500 --concurrency 50
```

Normal upload path, capped by the backend at 10MB:

```bash
python3 scripts/perf/gcs_load_test.py --mode upload-normal --upload-file /tmp/gcs-upload-1mb.bin --requests 200 --concurrency 20
```

Chunked upload path:

```bash
python3 scripts/perf/gcs_load_test.py --mode upload-chunked --upload-file /tmp/gcs-upload-20mb.bin --requests 50 --concurrency 5 --chunk-concurrency 3
```

## Metrics

The script prints JSON with:

- `qps`
- `throughput_mib_s`
- `latency_ms.avg/p50/p95/p99/max`
- `errors`
- `process_monitor.cpu_*` and `process_monitor.rss_*`

For MinIO/Redis/MySQL, collect service-native metrics alongside the run:

```bash
redis-cli INFO stats
mysqladmin extended-status
mc admin info local
mc admin prometheus metrics local
```

When using Docker, also capture:

```bash
docker stats
```

For credible numbers, run each case at several concurrency levels such as `1, 5, 10, 20, 50, 100`, and record QPS together with throughput and P95/P99. Upload tests create real files; use a disposable test user or clean the target folder afterward.
