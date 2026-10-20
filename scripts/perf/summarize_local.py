#!/usr/bin/env python3
"""Build a reproducible Markdown report from recorded local benchmark JSON."""
import argparse
from collections import defaultdict
import json
from pathlib import Path

def main():
 p=argparse.ArgumentParser();p.add_argument('directory',type=Path);args=p.parse_args();root=args.directory
 rows=json.loads((root/'results.json').read_text());env=json.loads((root/'environment.json').read_text())
 groups=defaultdict(list)
 for row in rows:groups[(row['mode'],row['concurrency'],row['requested_duration_seconds'])].append(row)
 lines=['# 本地七服务压力测试结果','',f"应用提交：`{env['app_commit']}`；采样开始：{env['created_at']}。",'',
 '## 环境与方法','',f"- {env['os']}，本机 {env['host_cpu_count']} 逻辑核；Docker {env['docker']['ServerVersion']}，分配 {env['docker']['NCPU']} CPU / {env['docker']['MemTotal']/1024**3:.2f} GiB 内存。",
 f"- 七服务及 MySQL、Redis、RabbitMQ、MinIO 与 Python 负载发生器同机运行；单测试账号、1 GiB 配额；每用户限流 RPS={env['rate_limit_rps']}（0 表示仅此隔离环境关闭）。",
 f"- 数据库池配置：`{env.get('database_pools', {'all':{'max_open':10,'max_idle':3}})}`。具体窗口与样本数见下表；写入为固定样本，第二轮读并发顺序反转。",
 '- 读同一个热缓存的 1 MiB 对象，Range 固定前 1 MiB；普通上传约 1 MiB，分片上传约 20 MiB、分片 5 MiB、每文件分片并行度 3；写入字节随机化避免秒传。',
 '- 闭环并发、每请求新建 urllib HTTP 连接；QPS 为成功操作/墙钟秒，分片 QPS 为完成文件数/秒。包含业务错误校验，不把 HTTP 200 的错误响应计为成功。',
 '- 表中 QPS/吞吐范围为各轮最小至最大；P95/P99 取较慢一轮的分位数，不是合并样本的分位数。延迟包含失败尝试，原始 JSON 另有成功请求分位数。','',
 '## 实测结果','',
 '| 场景 | 并发 | 窗口/轮数 | 成功 QPS | P95 ms | P99 ms | 有效载荷 MiB/s | 错误/尝试 |',
 '| --- | ---: | --- | ---: | ---: | ---: | ---: | ---: |']
 names={'identity':'身份读取','file-list':'目录列表','download-info':'下载签名','download-range':'Range 1 MiB','download-direct':'MinIO 直链 1 MiB','upload-normal':'普通上传 1 MiB','upload-chunked':'分片上传 20 MiB'}
 def span(values):
  lo,hi=min(values),max(values)
  return f'{lo:.2f}' if len(values)==1 else f'{lo:.2f}–{hi:.2f}'
 for (mode,c,duration),items in groups.items():
  throughput=span([r['throughput_mib_s'] for r in items]) if mode in ['download-range','download-direct','upload-normal','upload-chunked'] else '—'
  window=f'{duration:g}s × {len(items)}' if duration else f"{items[0]['requests']} 次 × {len(items)}"
  lines.append(f"| {names[mode]} | {c} | {window} | {span([r['qps'] for r in items])} | {max(r['latency_ms']['p95'] for r in items):.2f} | {max(r['latency_ms']['p99'] for r in items):.2f} | {throughput} | {sum(r['errors'] for r in items)}/{sum(r['requests'] for r in items)} |")
 errors=defaultdict(int)
 for row in rows:
  for name,count in row['error_counts'].items():errors[name]+=count
 lines+=['',f"总计 {sum(r['requests'] for r in rows):,} 次操作，{sum(r['errors'] for r in rows):,} 次失败。错误分布：`{dict(errors)}`。",'','## 资源采样','',
 'Docker 约每 2 秒采样一次，CPU 100% 约等于一个逻辑核；峰值可能低估瞬时尖峰。以下为整轮最高采样值，不能直接推断某服务就是瓶颈。','',
 '| 容器 | CPU 最高 % | 内存最高 MiB |','| --- | ---: | ---: |']
 resources=defaultdict(lambda:[0,0])
 def mem(v):
  v=v.split('/')[0].strip()
  for unit,factor in [('GiB',1024),('MiB',1),('KiB',1/1024),('B',1/1024**2)]:
   if v.endswith(unit):return float(v[:-len(unit)])*factor
  return 0
 for line in (root/'docker-stats.jsonl').read_text().splitlines():
  row=json.loads(line);name=row['Name'].rsplit('-',2)[-2];entry=resources[name]
  entry[0]=max(entry[0],float(row['CPUPerc'].rstrip('%')));entry[1]=max(entry[1],mem(row['MemUsage']))
 for name,(cpu,memory) in sorted(resources.items()):lines.append(f'| {name} | {cpu:.2f} | {memory:.2f} |')
 lines+=['',f"负载发生器最高单场景进程 CPU：{max(r['client_cpu_percent'] for r in rows):.1f}%（100%≈一个核）。",'',
 '## 已发现的问题与边界','',
 '- 修复前网关在持续并发下出现 `connect: cannot assign requested address` 并返回 502。网关、RPC、存储客户端扩大了空闲连接池；新增真实并发批次连接复用测试。修复前原始样本另存 `../2026-10-20-before-pool-fix/`。',
 '- 修复前开启每用户 1000 RPS 限流，修复后容量矩阵关闭该限制，因此不能把两组 QPS 的差值全部归因于连接池修改。部署模板默认仍为每用户 50 RPS。',
 '- 小数据库池（maxOpen=10/maxIdle=3）的持续认证读取仍出现 503，且部分依赖错误被误报为“用户不存在”；该错误归类已在后续提交中修复并单独回归测试，性能样本仍来自上面列出的原应用提交。另行保存扩大空闲数据库池的对照结果，不能把本矩阵称为全档零错误。',
 '- 测试脚本修正了上传业务错误误计成功及不允许的 .bin 文件名；这些属于原压测工具问题。',
 '- 同机热缓存、单账号、短读窗口/小写样本，不代表生产最大 QPS、带宽或并发用户数。并发数是正在执行的操作数，不是在线用户数。',
 '- 未覆盖多机网络/TLS、真实大目录、大文件长时间稳态、多租户、SSE 长连接容量、混合读写、数小时内存稳定性或生产备份恢复。',
 '- 功能优先级见 [后续功能建议](../../FEATURE_PRIORITIES.md)。','']
 (root/'REPORT.md').write_text('\n'.join(lines))
 print(root/'REPORT.md')
if __name__=='__main__':main()
