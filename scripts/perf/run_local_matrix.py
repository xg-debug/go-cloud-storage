#!/usr/bin/env python3
"""Benchmark an isolated environment created by microservices_smoke.py --hold-file.
Credentials stay in memory. Results contain no tokens, signed URLs or passwords.
"""
import argparse
import contextlib
import importlib.util
import io
import json
import os
from pathlib import Path
import platform
import subprocess
import sys
import threading
import time
from unittest.mock import patch
import gcs_load_test as bench

ROOT=Path(__file__).resolve().parents[2]
MIB=1024*1024

def main():
    parser=argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--ready-file',type=Path,required=True)
    parser.add_argument('--output',type=Path,required=True)
    parser.add_argument('--read-duration',type=float,default=10)
    args=parser.parse_args()
    ready=json.loads(args.ready_file.read_text())
    if not ready['gateway'].startswith('http://127.0.0.1:') or not ready['project'].startswith('cloudbox-smoke-'):
        raise ValueError('Only the disposable local smoke environment is accepted')
    args.output.mkdir(parents=True,exist_ok=True)
    spec=importlib.util.spec_from_file_location('smoke',ROOT/'scripts/integration/microservices_smoke.py')
    smoke=importlib.util.module_from_spec(spec);spec.loader.exec_module(smoke)
    browser=smoke.Browser(ready['gateway'])
    email='perf-'+str(int(time.time()))+'@example.test'
    password='LocalPerf-'+os.urandom(12).hex()+'!'
    browser.call('/api/register',{'email':email,'password':password,'password_confirm':password})
    browser.call('/api/login',{'account':email,'password':password})
    root=browser.call('/api/me')['rootFolderId']
    token=next(c.value for c in browser.cookies if c.name=='access_token')
    sample=os.urandom(MIB//2).hex().encode()
    fid=browser.upload(sample,root,'perf-download-1m.txt')['id']
    small=args.output/'sample-1m.bin';large=args.output/'sample-20m.bin'
    small.write_bytes(sample);large.write_bytes(os.urandom(10*MIB).hex().encode())
    docker=json.loads(subprocess.check_output(['docker','info','--format','{{json .}}'],text=True))
    environment={'created_at':time.strftime('%Y-%m-%dT%H:%M:%S%z'),'app_commit':subprocess.check_output(['git','rev-parse','HEAD'],cwd=ROOT,text=True).strip(),
        'os':platform.platform(),'python':platform.python_version(),'host_cpu_count':os.cpu_count(),
        'docker':{k:docker.get(k) for k in ['ServerVersion','NCPU','MemTotal','Architecture']},
        'topology':'7 application containers + MySQL/Redis/RabbitMQ/MinIO on same host as generator',
        'users':1,'rate_limit_rps':ready.get('rate_limit_rps',1000),'quota_gib':1,'file_bytes':MIB,
        'method':'Closed loop, fresh urllib HTTP connections, hot fixed 1MiB object/range; two read rounds; unique upload bytes; chunk QPS counts complete files.'}
    (args.output/'environment.json').write_text(json.dumps(environment,indent=2))
    containers=subprocess.check_output(['docker','ps','-q','--filter','label=com.docker.compose.project='+ready['project']],text=True).split()
    done=threading.Event();current={'case':'warmup'}
    def monitor():
        with (args.output/'docker-stats.jsonl').open('w') as out:
            while not done.is_set():
                began=time.time();case=current['case']
                result=subprocess.run(['docker','stats','--no-stream','--format','{{json .}}',*containers],capture_output=True,text=True,timeout=15)
                for line in result.stdout.splitlines():
                    try: out.write(json.dumps({'timestamp':began,'case':case,**json.loads(line)})+'\n')
                    except json.JSONDecodeError: pass
                out.flush();done.wait(1)
    thread=threading.Thread(target=monitor,daemon=True);thread.start()
    results=[]
    def run(mode,concurrency,requests=20,duration=0,round_id=0):
        case=f'{mode}-c{concurrency}-r{round_id}';current['case']=case
        argv=['bench','--base-url',ready['gateway']+'/api','--token',token,'--parent-id',root,'--file-id',fid,
            '--mode',mode,'--concurrency',str(concurrency),'--requests',str(requests),'--duration',str(duration),'--monitor','']
        if mode.startswith('upload'):argv+=['--upload-file',str(large if mode=='upload-chunked' else small),'--chunk-size',str(5*MIB),'--chunk-concurrency','3']
        output=io.StringIO();cpu=time.process_time()
        with patch('sys.argv',argv),contextlib.redirect_stdout(output):bench.main()
        result=json.loads(output.getvalue());result.update(case=case,round=round_id,client_cpu_percent=(time.process_time()-cpu)/result['elapsed_seconds']*100)
        return result
    try:
        for mode in ['identity','file-list','download-info','download-range','download-direct']:
            run(mode,1,requests=10)
            for round_id in [1,2]:
                # Reverse second round to expose order/thermal effects.
                for concurrency in ([1,5,10,20,50] if round_id==1 else [50,20,10,5,1]):
                    result=run(mode,concurrency,duration=args.read_duration,round_id=round_id)
                    results.append(result);(args.output/'results.json').write_text(json.dumps(results,indent=2))
                    print(f"{result['case']}: QPS={result['qps']:.2f} P95={result['latency_ms']['p95']:.1f}ms errors={result['errors']}/{result['requests']}",flush=True)
        for mode,levels,count in [('upload-normal',[1,5,10,20],64),('upload-chunked',[1,3,5],8)]:
            for concurrency in levels:
                result=run(mode,concurrency,requests=count,round_id=1);results.append(result)
                (args.output/'results.json').write_text(json.dumps(results,indent=2))
                print(f"{result['case']}: QPS={result['qps']:.2f} P95={result['latency_ms']['p95']:.1f}ms errors={result['errors']}/{result['requests']}",flush=True)
        result=run('download-info',20,duration=60,round_id=3);results.append(result)
        (args.output/'results.json').write_text(json.dumps(results,indent=2))
        print('PASS: matrix completed; inspect errors rather than treating completion as zero-error acceptance.',flush=True)
    finally:
        done.set();thread.join(timeout=20);small.unlink(missing_ok=True);large.unlink(missing_ok=True)

if __name__=='__main__':main()
