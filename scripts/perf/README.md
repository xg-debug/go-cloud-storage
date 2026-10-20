# 本地压力测试

测试工具只操作指定 API。上传会写入真实数据，建议使用下面的独立临时容器环境。**不要对已有业务账号直接执行容量矩阵。**

## 一键矩阵

先启动七服务隔离环境，关闭该环境的每用户限流以测量处理能力（不改变正式配置）：

```bash
python3 scripts/integration/microservices_smoke.py --containers \
  --rate-limit-rps 0 --hold-file /tmp/cloudbox-perf-ready.json
```

如本机无法下载基础镜像，参考 [验收记录](../../deploy/VERIFICATION.md) 的缓存镜像方式，追加上述参数。另一个终端执行：

```bash
python3 scripts/perf/run_local_matrix.py \
  --ready-file /tmp/cloudbox-perf-ready.json \
  --output scripts/perf/results/local-run
# 结束后停止临时环境并由启动脚本删除其容器与卷
rm /tmp/cloudbox-perf-ready.json
```

矩阵覆盖身份读取、目录列表、下载签名、1 MiB Range/直链、1 MiB 普通上传、20 MiB 分片上传。读请求并发 1/5/10/20/50，每档 10 秒、两轮（第二轮反序）；写请求为固定样本，另外执行 60 秒下载签名稳态检查。只使用一个临时账号，上传内容随机化，避免秒传冒充真实写入；默认写入不足 1 GiB。整个测试应在环境 30 分钟保留期限内完成。

脚本记录 `environment.json`、`results.json`、`docker-stats.jsonl`，不保存登录令牌、签名 URL 或账号密码；临时大文件结束后删除。测试环境全部在本机，数据和网络有热缓存效应，结果不能直接外推生产容量。

## 单项测试

`GCS_BASE_URL` 必须包含网关 `/api` 前缀，使用访问令牌 Bearer 认证。

```bash
export GCS_BASE_URL=http://127.0.0.1:8081/api
export GCS_TOKEN='测试账号的-access-token'
export GCS_DOWNLOAD_FILE_ID='至少-1MiB-的测试文件ID'
export GCS_PARENT_ID='测试目录ID'
python3 scripts/perf/gcs_load_test.py --mode download-info --duration 30 --concurrency 20 --monitor ''
python3 scripts/perf/gcs_load_test.py --mode download-range --range-size 1048576 --requests 500 --concurrency 20 --monitor ''
python3 scripts/perf/gcs_load_test.py --mode download-direct --requests 200 --concurrency 10 --monitor ''
python3 scripts/perf/gcs_load_test.py --mode upload-normal --upload-file /tmp/sample.txt --requests 50 --concurrency 5 --monitor ''
python3 scripts/perf/gcs_load_test.py --mode upload-chunked --upload-file /tmp/sample-20m.txt --requests 8 --concurrency 3 --chunk-size 5242880 --chunk-concurrency 3 --monitor ''
```

支持 `identity`、`file-list`，以及以上五种模式。普通上传不超过 10 MiB；随机化会追加少量字节，应留出边界余量。上传使用白名单允许的 `.txt` 文件名。Range 固定读取对象的第一段，并校验 HTTP 206 和返回字节数。

## 指标口径与限制

- `qps` 为成功业务操作数 / 含最后在途请求完成时间的墙钟时长；`attempt_qps` 包含失败尝试。
- 分片模式的 QPS 是完整文件数/秒，一次文件操作包含初始化、多个分片和合并，不是底层 HTTP 请求数。
- `latency_ms` 包含成功和失败，`success_latency_ms` 单独记录成功请求；`error_counts` 分别保留 429、502 和业务错误。HTTP 200 不等于业务成功。
- `throughput_mib_s` 使用成功操作的有效载荷字节；签名/列表的响应体吞吐没有文件带宽意义。
- 闭环 Python 线程负载发生器，每请求新建 urllib HTTP 连接，包含建连、序列化及上传内容生成成本，不是恒定到达率模型；不测排队等待之外的 coordinated omission，不能作为服务极限 QPS 声明。
- Docker CPU 100% 约等于一个逻辑核；同机负载发生器、Docker 虚拟机及后台应用共享资源。`process_monitor` 不适合从 macOS 直接观测容器进程，矩阵另存 Docker 指标与发生器 CPU。
- 生产模板每用户默认限流 50 RPS，容量测试关闭该限制的结果不代表用户可绕过限流。

回归测试：`python3 -m unittest discover -s scripts/perf -p 'test_*.py'`。

[后续功能优先级](FEATURE_PRIORITIES.md)。各次结果保存在 `results/`，报告会注明应用提交、限流策略、错误与未覆盖场景。

2026-10-20 的实测与问题记录见 [本地压测汇总](results/2026-10-20-SUMMARY.md)。

## 数据库池对照

完整矩阵结束后，可仅调整同一个临时环境的 user-service，保持最大连接 10，将空闲连接从 3 提高到 10；不修改正式配置：

```bash
python3 scripts/perf/tune_local_db_pool.py --ready-file /tmp/cloudbox-perf-ready.json --output /tmp/db-before.json --apply
python3 scripts/perf/run_local_matrix.py --ready-file /tmp/cloudbox-perf-ready.json \
  --output scripts/perf/results/db-pool-tuned --read-modes identity \
  --read-concurrency 20,50 --read-rounds 1 --read-duration 60 --skip-writes \
  --steady-mode identity --steady-concurrency 50 --steady-duration 120
python3 scripts/perf/tune_local_db_pool.py --ready-file /tmp/cloudbox-perf-ready.json --output /tmp/db-after.json
```

`python3 scripts/perf/summarize_local.py <结果目录>` 可重新生成 Markdown 表格。调优应依据实际并发与数据库总连接预算，不应盲目复制本机连接数到生产。
