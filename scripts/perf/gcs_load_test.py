#!/usr/bin/env python3
import argparse
import concurrent.futures
import hashlib
import json
import os
import statistics
import subprocess
import sys
import threading
import time
import uuid
import urllib.error
import urllib.parse
import urllib.request


MIB = 1024 * 1024


def parse_args():
    parser = argparse.ArgumentParser(description="Go Cloud Storage upload/download load tester")
    parser.add_argument("--base-url", default=os.getenv("GCS_BASE_URL", "http://localhost:8081"))
    parser.add_argument("--token", default=os.getenv("GCS_TOKEN", ""))
    parser.add_argument("--mode", required=True, choices=[
        "download-info",
        "download-direct",
        "download-range",
        "upload-normal",
        "upload-chunked",
    ])
    parser.add_argument("--file-id", default=os.getenv("GCS_DOWNLOAD_FILE_ID", ""))
    parser.add_argument("--upload-file", default=os.getenv("GCS_UPLOAD_FILE", ""))
    parser.add_argument("--parent-id", default=os.getenv("GCS_PARENT_ID", ""))
    parser.add_argument("--requests", type=int, default=int(os.getenv("GCS_REQUESTS", "100")))
    parser.add_argument("--concurrency", type=int, default=int(os.getenv("GCS_CONCURRENCY", "10")))
    parser.add_argument("--chunk-size", type=int, default=int(os.getenv("GCS_CHUNK_SIZE", str(10 * MIB))))
    parser.add_argument("--chunk-concurrency", type=int, default=int(os.getenv("GCS_CHUNK_CONCURRENCY", "3")))
    parser.add_argument("--range-size", type=int, default=int(os.getenv("GCS_RANGE_SIZE", str(MIB))))
    parser.add_argument("--no-randomize-upload", action="store_true")
    parser.add_argument("--monitor", default=os.getenv("GCS_MONITOR", "main,go-cloud-storage,minio,mysqld,redis-server"))
    parser.add_argument("--monitor-interval", type=float, default=float(os.getenv("GCS_MONITOR_INTERVAL", "1.0")))
    return parser.parse_args()


def auth_headers(token):
    headers = {}
    if token:
        headers["Authorization"] = f"Bearer {token}"
    return headers


def request_json(method, url, token="", payload=None, timeout=300):
    data = None
    headers = auth_headers(token)
    if payload is not None:
        data = json.dumps(payload).encode("utf-8")
        headers["Content-Type"] = "application/json"
    req = urllib.request.Request(url, data=data, headers=headers, method=method)
    with urllib.request.urlopen(req, timeout=timeout) as resp:
        body = resp.read()
        obj = json.loads(body.decode("utf-8"))
        if obj.get("code") != 200:
            raise RuntimeError(obj.get("message") or "request failed")
        return obj.get("data"), len(body)


def request_bytes(method, url, token="", headers=None, data=None, timeout=300):
    req_headers = auth_headers(token)
    if headers:
        req_headers.update(headers)
    req = urllib.request.Request(url, data=data, headers=req_headers, method=method)
    total = 0
    status = 0
    with urllib.request.urlopen(req, timeout=timeout) as resp:
        status = resp.status
        while True:
            chunk = resp.read(256 * 1024)
            if not chunk:
                break
            total += len(chunk)
    return status, total


def multipart_body(fields, file_field, filename, content, content_type="application/octet-stream"):
    boundary = "----gcsbench" + uuid.uuid4().hex
    chunks = []
    for key, value in fields.items():
        chunks.append(f"--{boundary}\r\n".encode())
        chunks.append(f'Content-Disposition: form-data; name="{key}"\r\n\r\n'.encode())
        chunks.append(str(value).encode())
        chunks.append(b"\r\n")
    chunks.append(f"--{boundary}\r\n".encode())
    chunks.append(
        f'Content-Disposition: form-data; name="{file_field}"; filename="{filename}"\r\n'.encode()
    )
    chunks.append(f"Content-Type: {content_type}\r\n\r\n".encode())
    chunks.append(content)
    chunks.append(b"\r\n")
    chunks.append(f"--{boundary}--\r\n".encode())
    body = b"".join(chunks)
    return body, f"multipart/form-data; boundary={boundary}"


def load_upload_bytes(path):
    if not path:
        raise ValueError("--upload-file is required for upload modes")
    with open(path, "rb") as f:
        return f.read()


def upload_content(base, randomize, idx):
    if not randomize:
        return base
    return base + f"\nbench-{idx}-{uuid.uuid4().hex}".encode("utf-8")


def run_download_info(args, _idx):
    data, byte_count = request_json("GET", f"{args.base_url}/file/download-info/{args.file_id}", args.token)
    return byte_count, data


def fetch_direct_url(args):
    data, _ = request_json("GET", f"{args.base_url}/file/download-info/{args.file_id}", args.token)
    direct = data.get("directDownloadUrl")
    if not direct:
        raise RuntimeError("download-info did not return directDownloadUrl")
    return direct


def run_download_direct(args, _idx, direct_url):
    _status, byte_count = request_bytes("GET", direct_url)
    return byte_count, None


def run_download_range(args, idx):
    start = (idx * args.range_size) % max(args.range_size, 1)
    end = start + args.range_size - 1
    headers = {"Range": f"bytes={start}-{end}"}
    _status, byte_count = request_bytes("GET", f"{args.base_url}/file/download/{args.file_id}", args.token, headers=headers)
    return byte_count, None


def run_upload_normal(args, idx, base_content):
    content = upload_content(base_content, not args.no_randomize_upload, idx)
    file_hash = hashlib.sha256(content).hexdigest()
    filename = f"bench-{idx}-{uuid.uuid4().hex}.bin"
    body, content_type = multipart_body({
        "parentId": args.parent_id,
        "fileHash": file_hash,
    }, "file", filename, content)
    headers = {"Content-Type": content_type, "Content-Length": str(len(body))}
    _status, response_bytes = request_bytes("POST", f"{args.base_url}/file/upload", args.token, headers=headers, data=body)
    return len(content), response_bytes


def run_upload_chunked(args, idx, base_content):
    content = upload_content(base_content, not args.no_randomize_upload, idx)
    file_hash = hashlib.sha256(content).hexdigest()
    filename = f"bench-chunked-{idx}-{uuid.uuid4().hex}.bin"
    total_chunks = (len(content) + args.chunk_size - 1) // args.chunk_size
    init_payload = {
        "fileName": filename,
        "fileHash": file_hash,
        "fileSize": len(content),
        "parentId": args.parent_id,
        "chunkSize": args.chunk_size,
        "totalChunks": total_chunks,
    }
    init_data, _ = request_json("POST", f"{args.base_url}/file/chunk/init", args.token, init_payload)
    if init_data and init_data.get("finished"):
        return len(content), None

    def put_part(chunk_index):
        start = chunk_index * args.chunk_size
        end = min(len(content), start + args.chunk_size)
        chunk = content[start:end]
        body, content_type = multipart_body({
            "fileHash": file_hash,
            "chunkIndex": chunk_index,
        }, "chunk", f"{filename}.part{chunk_index}", chunk)
        headers = {"Content-Type": content_type, "Content-Length": str(len(body))}
        request_bytes("POST", f"{args.base_url}/file/chunk/upload", args.token, headers=headers, data=body)

    with concurrent.futures.ThreadPoolExecutor(max_workers=args.chunk_concurrency) as pool:
        list(pool.map(put_part, range(total_chunks)))

    merge_payload = {
        "fileHash": file_hash,
        "fileName": filename,
        "fileSize": len(content),
        "parentId": args.parent_id,
        "chunkSize": args.chunk_size,
        "totalChunks": total_chunks,
    }
    request_json("POST", f"{args.base_url}/file/chunk/merge", args.token, merge_payload)
    return len(content), None


class ProcessSampler:
    def __init__(self, names, interval):
        self.names = [n.strip() for n in names.split(",") if n.strip()]
        self.interval = interval
        self.samples = []
        self.stop_event = threading.Event()
        self.thread = threading.Thread(target=self._run, daemon=True)

    def start(self):
        if self.names:
            self.thread.start()

    def stop(self):
        self.stop_event.set()
        if self.names:
            self.thread.join(timeout=2)

    def _run(self):
        while not self.stop_event.is_set():
            sample = self._sample_once()
            if sample:
                self.samples.append(sample)
            self.stop_event.wait(self.interval)

    def _sample_once(self):
        try:
            out = subprocess.check_output(["ps", "-axo", "pid=,pcpu=,pmem=,rss=,comm="], text=True)
        except Exception:
            return None
        totals = {"cpu": 0.0, "mem": 0.0, "rss": 0.0, "count": 0}
        for line in out.splitlines():
            parts = line.split(None, 4)
            if len(parts) < 5:
                continue
            _pid, cpu, mem, rss, comm = parts
            if not any(name in comm for name in self.names):
                continue
            totals["cpu"] += float(cpu)
            totals["mem"] += float(mem)
            totals["rss"] += int(rss) / 1024
            totals["count"] += 1
        return totals if totals["count"] else None

    def summary(self):
        if not self.samples:
            return {}
        return {
            "samples": len(self.samples),
            "cpu_avg_percent": statistics.mean(s["cpu"] for s in self.samples),
            "cpu_max_percent": max(s["cpu"] for s in self.samples),
            "rss_avg_mb": statistics.mean(s["rss"] for s in self.samples),
            "rss_max_mb": max(s["rss"] for s in self.samples),
            "process_count_max": max(s["count"] for s in self.samples),
        }


def percentile(values, pct):
    if not values:
        return 0
    ordered = sorted(values)
    idx = int((len(ordered) - 1) * pct / 100)
    return ordered[idx]


def main():
    args = parse_args()
    if args.mode.startswith("download") and not args.file_id:
        print("--file-id or GCS_DOWNLOAD_FILE_ID is required", file=sys.stderr)
        return 2
    if args.mode.startswith("upload") and not args.token:
        print("--token or GCS_TOKEN is required for upload modes", file=sys.stderr)
        return 2
    if args.mode in ("download-info", "download-range") and not args.token:
        print("--token or GCS_TOKEN is required for authenticated download modes", file=sys.stderr)
        return 2

    base_upload_content = None
    if args.mode.startswith("upload"):
        base_upload_content = load_upload_bytes(args.upload_file)

    direct_url = None
    if args.mode == "download-direct":
        direct_url = fetch_direct_url(args)

    sampler = ProcessSampler(args.monitor, args.monitor_interval)
    sampler.start()

    latencies = []
    errors = 0
    bytes_total = 0
    started = time.perf_counter()
    lock = threading.Lock()

    def one(idx):
        nonlocal errors, bytes_total
        t0 = time.perf_counter()
        try:
            if args.mode == "download-info":
                byte_count, _ = run_download_info(args, idx)
            elif args.mode == "download-direct":
                byte_count, _ = run_download_direct(args, idx, direct_url)
            elif args.mode == "download-range":
                byte_count, _ = run_download_range(args, idx)
            elif args.mode == "upload-normal":
                byte_count, _ = run_upload_normal(args, idx, base_upload_content)
            elif args.mode == "upload-chunked":
                byte_count, _ = run_upload_chunked(args, idx, base_upload_content)
            else:
                raise RuntimeError(f"unsupported mode {args.mode}")
            ok = True
        except Exception as exc:
            byte_count = 0
            ok = False
            print(f"request {idx} failed: {exc}", file=sys.stderr)
        elapsed = time.perf_counter() - t0
        with lock:
            latencies.append(elapsed)
            bytes_total += byte_count
            if not ok:
                errors += 1

    with concurrent.futures.ThreadPoolExecutor(max_workers=args.concurrency) as pool:
        list(pool.map(one, range(args.requests)))

    total_elapsed = time.perf_counter() - started
    sampler.stop()

    success = args.requests - errors
    result = {
        "mode": args.mode,
        "requests": args.requests,
        "success": success,
        "errors": errors,
        "concurrency": args.concurrency,
        "elapsed_seconds": total_elapsed,
        "qps": success / total_elapsed if total_elapsed > 0 else 0,
        "throughput_mib_s": (bytes_total / MIB) / total_elapsed if total_elapsed > 0 else 0,
        "bytes_total": bytes_total,
        "latency_ms": {
            "avg": statistics.mean(latencies) * 1000 if latencies else 0,
            "p50": percentile(latencies, 50) * 1000,
            "p95": percentile(latencies, 95) * 1000,
            "p99": percentile(latencies, 99) * 1000,
            "max": max(latencies) * 1000 if latencies else 0,
        },
        "process_monitor": sampler.summary(),
    }
    print(json.dumps(result, ensure_ascii=False, indent=2))
    return 0 if errors == 0 else 1


if __name__ == "__main__":
    raise SystemExit(main())
