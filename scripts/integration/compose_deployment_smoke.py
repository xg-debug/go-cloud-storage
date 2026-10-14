#!/usr/bin/env python3
"""Exercise the shipped Compose topology (including IPAM and health dependencies).
Only published ports and test credentials change; service definitions come from
'docker compose config'. Removes only this invocation's project and volumes.
"""
import json
import os
from pathlib import Path
import secrets
import subprocess
import tempfile
from microservices_smoke import ROOT, Browser, free_port, wait


def main():
    with tempfile.TemporaryDirectory(prefix='cloudbox-deployment-') as directory:
        tmp = Path(directory)
        env = os.environ.copy()
        credentials = []
        for line in (ROOT/'deploy/.env.example').read_text().splitlines():
            if not line or line.startswith('#') or '=' not in line:
                continue
            key, _ = line.split('=', 1)
            value = secrets.token_hex(24)
            credentials.append(value)
            env[key] = value
        gateway_port, minio_port, console_port = free_port(), free_port(), free_port()
        env['GCS_MINIO_PUBLIC_ENDPOINT'] = f'127.0.0.1:{minio_port}'
        resolved = subprocess.run(['docker','compose','-f',str(ROOT/'deploy/compose.yaml'),'config','--format','json'],
                                  env=env,check=True,capture_output=True,text=True)
        manifest = json.loads(resolved.stdout)
        manifest.pop('name', None)
        for item in manifest.get('networks', {}).values():
            item.pop('name', None)
        for item in manifest.get('volumes', {}).values():
            item.pop('name', None)
        for role, mapping in {'gateway':{8080:gateway_port}, 'minio':{9000:minio_port,9001:console_port}}.items():
            for port in manifest['services'][role]['ports']:
                port['published'] = str(mapping[port['target']])
                port['host_ip'] = '127.0.0.1'
        path = tmp/'compose.json'
        path.write_text(json.dumps(manifest)); path.chmod(0o600)
        command = ['docker','compose','-p',f'cloudbox-deployment-{os.getpid()}','-f',str(path)]
        def run(*args, check=True):
            return subprocess.run(command+list(args),env=env,check=check,capture_output=True,text=True)
        try:
            print('Building and starting the shipped Compose topology...',flush=True)
            run('up','--build','-d','--wait','--wait-timeout','150')
            client = Browser(f'http://127.0.0.1:{gateway_port}')
            client.call('/api/register',{'email':'deployment@example.test','password':'DeploymentTest123!','password_confirm':'DeploymentTest123!'})
            client.call('/api/login',{'account':'deployment@example.test','password':'DeploymentTest123!'})
            root = client.call('/api/me')['rootFolderId']
            payload = b'shipped compose topology verification'
            uploaded = client.upload(payload,root)
            assert client.call('/api/file/preview-stream/'+uploaded['id']) == payload
            wait(lambda: client.call('/api/notification')['total'] == 1)
            print('PASS: original Compose health checks, IPAM, authentication, upload/download and event delivery.',flush=True)
        except Exception as error:
            output = run('logs','--no-color','--tail','80',check=False).stdout
            if isinstance(error, subprocess.CalledProcessError):
                output += (error.stdout or '') + (error.stderr or '')
            for secret in credentials:
                output = output.replace(secret, '[redacted]')
            diagnostics = ROOT/'backend/tmp/microservices-smoke'; diagnostics.mkdir(parents=True,exist_ok=True)
            log = diagnostics/'deployment.log'; log.write_text(output); log.chmod(0o600)
            raise RuntimeError(f'Deployment verification failed; see {log}') from None
        finally:
            run('down','--volumes','--remove-orphans','--rmi','local',check=False)


if __name__ == '__main__':
    main()
