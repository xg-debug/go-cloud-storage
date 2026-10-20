#!/usr/bin/env python3
"""Tune only the disposable smoke environment's user-service database pool."""
import argparse
import json
from pathlib import Path
import subprocess
import time
import urllib.request


def main():
    p=argparse.ArgumentParser(description=__doc__)
    p.add_argument('--ready-file',type=Path,required=True)
    p.add_argument('--output',type=Path,required=True)
    p.add_argument('--apply',action='store_true',help='Set user DB maxIdle=maxOpen=10 and restart only that disposable service')
    args=p.parse_args();ready=json.loads(args.ready_file.read_text());project=ready['project']
    if not project.startswith('cloudbox-smoke-') or not ready['gateway'].startswith('http://127.0.0.1:'):
        raise ValueError('Not an isolated local smoke environment')
    def snapshot():
        query="SHOW GLOBAL STATUS WHERE Variable_name IN ('Connections','Threads_connected','Threads_running','Max_used_connections','Aborted_connects','Connection_errors_max_connections');"
        raw=subprocess.check_output(['docker','exec',project+'-mysql-1','sh','-c','MYSQL_PWD="$MYSQL_ROOT_PASSWORD" mysql -uroot -N -e "'+query+'"'],text=True)
        return {'timestamp':time.time(),'mysql':{k:int(v) for k,v in (line.split('\t') for line in raw.splitlines())}}
    record={'before':snapshot()}
    if args.apply:
        name=project+'-user-1'
        mounts=json.loads(subprocess.check_output(['docker','inspect','--format','{{json .Mounts}}',name],text=True))
        source=Path(next(m['Source'] for m in mounts if m['Destination']=='/app/conf/go-cloud-storage.dev.yaml'))
        if not source.parent.parent.name.startswith('cloudbox-smoke-') or source.parent.name!='cfg-user':
            raise ValueError('Unexpected config source')
        cfg=json.loads(source.read_text());record['previous_pool']={k:cfg['database'][k] for k in ['maxOpenConns','maxIdleConns']}
        cfg['database'].update(maxOpenConns=10,maxIdleConns=10);source.write_text(json.dumps(cfg))
        subprocess.run(['docker','restart',name],check=True,stdout=subprocess.DEVNULL)
        deadline=time.monotonic()+30
        while True:
            try:
                with urllib.request.urlopen(f"http://127.0.0.1:{cfg['server']['port']}/healthz",timeout=1) as response:
                    if response.status==200:break
            except Exception:
                if time.monotonic()>deadline:raise
                time.sleep(.5)
        ready['database_pools']={'user':{'max_open':10,'max_idle':10},'other_services':{'max_open':10,'max_idle':3}}
        args.ready_file.write_text(json.dumps(ready))
        record['applied_pool']=ready['database_pools'];record['after_restart']=snapshot()
    args.output.write_text(json.dumps(record,indent=2));print(args.output)

if __name__=='__main__':main()
