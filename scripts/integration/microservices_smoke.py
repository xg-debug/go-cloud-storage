#!/usr/bin/env python3
"""Real seven-process smoke test with isolated disposable Docker infrastructure.
Requires Go, Python 3, Docker Compose; GCS_TEST_*_IMAGE overrides cached images.
Only this invocation's temporary Compose project and volumes are removed.
"""
import argparse
from datetime import datetime, timezone
import hashlib
import http.cookiejar
import json
import os
from pathlib import Path
import secrets
import socket
import subprocess
import tempfile
import time
import urllib.error
import urllib.request

ROOT = Path(__file__).resolve().parents[2]
ROLES = ['storage', 'user', 'auth', 'file', 'share', 'notification', 'gateway']

def free_port():
    with socket.socket() as s:
        s.bind(('127.0.0.1', 0))
        return s.getsockname()[1]

class ProcessExited(RuntimeError):
    pass

def wait(fn, timeout=90):
    deadline, last = time.monotonic() + timeout, None
    while time.monotonic() < deadline:
        try:
            if value := fn():
                return value
        except ProcessExited:
            raise
        except Exception as e:
            last = e
        time.sleep(.5)
    raise RuntimeError(f'timed out: {last}')

class Browser:
    def __init__(self, origin):
        self.origin, self.cookies = origin, http.cookiejar.CookieJar()
        self.client = urllib.request.build_opener(urllib.request.HTTPCookieProcessor(self.cookies))
    def call(self, path, value=None, method=None, raw=None, content_type=None, expect=200, headers=None, business_code=200):
        data = raw if raw is not None else (json.dumps(value).encode() if value is not None else None)
        headers = dict(headers or {})
        if data is not None:
            headers['Content-Type'] = content_type or 'application/json'
        csrf = next((c.value for c in self.cookies if c.name == 'csrf_token'), None)
        if csrf:
            headers['X-CSRF-Token'] = csrf
        try:
            res = self.client.open(urllib.request.Request(self.origin+path, data=data, headers=headers, method=method), timeout=20)
        except urllib.error.HTTPError as e:
            res = e
        with res:
            body = res.read()
            assert res.status == expect, (path, res.status, body[:300])
            if 'application/json' in res.headers.get('Content-Type', ''):
                result = json.loads(body)
                if expect == 200:
                    assert result.get('code', 200) == business_code, (path, result)
                return result.get('data', result)
            return body
    def upload(self, data, root, name='smoke.txt', expect=200, business_code=200):
        return self.multipart('/api/file/upload', data, name, 'file', {'fileHash': hashlib.sha256(data).hexdigest(), 'parentId': root}, expect, business_code)
    def multipart(self, path, data, name, field, fields, expect=200, business_code=200):
        boundary = 'smoke-'+secrets.token_hex(8)
        body = bytearray()
        for key, value in fields.items():
            body.extend(f'--{boundary}\r\nContent-Disposition: form-data; name="{key}"\r\n\r\n{value}\r\n'.encode())
        body.extend(f'--{boundary}\r\nContent-Disposition: form-data; name="{field}"; filename="{name}"\r\nContent-Type: text/plain\r\n\r\n'.encode())
        body.extend(data); body.extend(f'\r\n--{boundary}--\r\n'.encode())
        return self.call(path, raw=bytes(body), content_type=f'multipart/form-data; boundary={boundary}', expect=expect, business_code=business_code)

def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--containers', action='store_true', help='Build and run application Docker images instead of host binaries')
    parser.add_argument('--hold-file', type=Path, help='Write local connection info and keep test environment alive until this file is removed (max 30 minutes)')
    parser.add_argument('--migration-rehearsal', action='store_true', help='Exercise legacy schema import and reverse copy in this disposable project')
    parser.add_argument('--local-runtime-image', help='With --containers: cross-compile locally and use an existing CloudBox runtime image without fetching compiler images')
    args = parser.parse_args()
    if args.local_runtime_image and not args.containers: parser.error('--local-runtime-image requires --containers')
    containers = args.containers
    with tempfile.TemporaryDirectory(prefix='cloudbox-smoke-') as directory:
        tmp, password = Path(directory), secrets.token_hex(24)
        ports = {key: free_port() for key in ROLES+['mysql', 'redis', 'rabbitmq', 'minio']}
        env = os.environ.copy()
        build_arch = None
        if args.local_runtime_image:
            build_arch = subprocess.check_output(['docker','image','inspect','--format','{{.Architecture}}',args.local_runtime_image],text=True).strip()
            if build_arch not in ('arm64','amd64'): raise ValueError('unsupported local runtime architecture')
        env['MYSQL_ROOT_PASSWORD'] = password
        for role in ['user', 'file', 'share', 'notification']:
            env['GCS_'+role.upper()+'_DB_PASSWORD'] = password
        images = {k: os.environ.get('GCS_TEST_'+k.upper()+'_IMAGE', v) for k, v in {
            'mysql': 'mysql:8.0', 'redis': 'redis:7-alpine', 'rabbitmq': 'rabbitmq:3.13-management-alpine',
            'minio': ''}.items()}
        services = {k: {'image': image, 'ports': [f'127.0.0.1:{ports[k]}:{p}']} for (k, p), image in zip(
            [('mysql',3306), ('redis',6379), ('rabbitmq',5672), ('minio',9000)], images.values())}
        services['mysql'].update(environment={k: '${'+k+'}' for k in ['MYSQL_ROOT_PASSWORD']+[f'GCS_{r.upper()}_DB_PASSWORD' for r in ['user','file','share','notification']]},
                                 build={'context':str(ROOT), 'dockerfile':'deploy/mysql.Dockerfile', 'args':{'MYSQL_IMAGE':images['mysql']}})
        services['mysql'].pop('image')
        services['rabbitmq']['environment'] = {'RABBITMQ_DEFAULT_USER':'cloudbox', 'RABBITMQ_DEFAULT_PASS':password}
        if not images['minio']:
            services['minio'].pop('image')
            services['minio']['build'] = {'context': str(ROOT), 'dockerfile': 'deploy/minio.Dockerfile'}
        services['minio'].update(command=['server','/data'], environment={'MINIO_ROOT_USER':'cloudbox','MINIO_ROOT_PASSWORD':password})
        manifest = tmp/'compose.json'; manifest.write_text(json.dumps({'services':services})); manifest.chmod(0o600)
        base = ['docker','compose','-p',f'cloudbox-smoke-{os.getpid()}','-f',str(manifest)]
        processes, logs, service_env = {}, {}, {}
        def compose(*args, check=True):
            result = subprocess.run(base+list(args),env=env,check=False,stdout=subprocess.PIPE,stderr=subprocess.STDOUT,text=True)
            if check and result.returncode:
                (tmp/'compose-error.log').write_text(result.stdout.replace(password, '[redacted]'))
                raise RuntimeError('Compose command failed: '+result.stdout.replace(password, '[redacted]')[-2500:])
            return result
        def sql(database, query, user='root', check=True):
            return compose('exec','-T','-e','MYSQL_PWD='+password,'mysql','mysql','-N','-u'+user,database,'-e',query,check=check)
        def start(role):
            if containers:
                compose('up', '-d', '--no-deps', role)
                processes[role] = None
            else:
                logs[role] = open(tmp/f'{role}.log','ab')
                processes[role] = subprocess.Popen([str(tmp/role)],cwd=tmp/('cfg-'+role),env=service_env[role],stdout=logs[role],stderr=subprocess.STDOUT)
            print(f'Starting {role}-service...', flush=True)
            def ready():
                if (containers and not compose('ps', '--status', 'running', '-q', role).stdout.strip()) or (not containers and processes[role].poll() is not None):
                    raise ProcessExited(f'{role}-service exited at startup')
                return urllib.request.urlopen(f'http://127.0.0.1:{ports[role]}/healthz',timeout=1).status==200
            wait(ready)
        def stop(role):
            proc = processes.pop(role)
            if containers:
                compose('stop', '-t', '20', role)
                return
            proc.terminate()
            try:
                proc.wait(timeout=20)
            except subprocess.TimeoutExpired:
                proc.kill(); proc.wait()
            logs.pop(role).close()
        try:
            print('Preparing container integration...' if containers else 'Building seven service binaries...',flush=True)
            for role in ([] if containers and not args.local_runtime_image else ROLES):
                target = './cmd/gateway' if role=='gateway' else f'./services/{role}-service/cmd/server'
                build_env = env.copy()
                if args.local_runtime_image: build_env.update(GOOS='linux',GOARCH=build_arch,CGO_ENABLED='0')
                subprocess.run(['go','build','-o',str(tmp/role),target],cwd=ROOT/'backend',env=build_env,check=True)
            print('Starting isolated infrastructure...',flush=True)
            compose('up','-d','--build')
            def mysql_ready():
                result = sql('cloudbox_user','SELECT 1',check=False)
                if result.returncode:
                    raise RuntimeError(result.stdout.replace(password, '[redacted]')[:500])
                return True
            wait(mysql_ready, 90)
            wait(lambda: compose('exec','-T','rabbitmq','rabbitmq-diagnostics','-q','ping',check=False).returncode==0,120)
            wait(lambda: urllib.request.urlopen(f'http://127.0.0.1:{ports["minio"]}/minio/health/live',timeout=1).status==200)
            for role in ROLES:
                configdir = tmp/('cfg-'+role); configdir.mkdir()
                cfg = {'server':{'port':ports[role],'env':'prod','storagePath':'/tmp','allowedOrigins':['http://localhost:8080'],'trustedProxies':['127.0.0.1']},
                       'database':{'host':'127.0.0.1','port':ports['mysql'],'user':'cloudbox_'+role,'password':password,'dbname':'cloudbox_'+role,'maxOpenConns':10,'maxIdleConns':3},
                       'redis':{'enabled':True,'host':'127.0.0.1','port':ports['redis'],'db':{'auth':0,'file':1,'notification':2}.get(role,3)},
                       'storageService':{'url':f'http://127.0.0.1:{ports["storage"]}','token':password},
                       'rabbitmq':{'url':f'amqp://cloudbox:{password}@127.0.0.1:{ports["rabbitmq"]}/'},
                       'jwt':{'secret':password},'security':{'defaultQuotaGB':1,'maxFileSizeMB':50,'rateLimitRPS':1000}}
                file = configdir/'go-cloud-storage.dev.yaml'
                file.write_text(json.dumps(cfg)); file.chmod(0o600)
                se = {k:v for k,v in os.environ.items() if not k.startswith('GCS_') and k!='JWT_SECRET'}
                se.update(GCS_SERVICE_TOKEN=password,GCS_STORAGE_TOKEN=password,GCS_AUTO_MIGRATE='true',GIN_MODE='release',
                          GCS_MINIO_PUBLIC_ENDPOINT=f'127.0.0.1:{ports["minio"]}',GCS_MINIO_ENDPOINT=f'127.0.0.1:{ports["minio"]}',GCS_MINIO_ACCESS_KEY='cloudbox',GCS_MINIO_SECRET_KEY=password,GCS_MINIO_BUCKET='smoke',GCS_MINIO_REGION='us-east-1',
                          GCS_STORAGE_LISTEN=f'127.0.0.1:{ports["storage"]}',GCS_GATEWAY_LISTEN=f'127.0.0.1:{ports["gateway"]}')
                for peer in ['auth','user','file','share','notification']:
                    se['GCS_'+peer.upper()+'_URL'] = f'http://127.0.0.1:{ports[peer]}'
                if containers:
                    cfg['database'].update(host='mysql', port=3306)
                    cfg['redis'].update(host='redis', port=6379)
                    cfg['storageService']['url'] = f'http://storage:{ports["storage"]}'
                    cfg['rabbitmq']['url'] = f'amqp://cloudbox:{password}@rabbitmq:5672/'
                    # Non-secret config is readable by the image's unprivileged UID.
                    # Secrets stay in the temporary 0600 Compose manifest/environment.
                    del cfg['database']['password']
                    del cfg['storageService']['token']
                    cfg['rabbitmq']['url'] = ''
                    cfg['jwt']['secret'] = ''
                    file.write_text(json.dumps(cfg)); file.chmod(0o644)
                    se = {k:v for k,v in se.items() if k.startswith('GCS_') or k == 'GIN_MODE'}
                    se.update(GCS_DB_PASSWORD=password, GCS_JWT_SECRET=password,
                              GCS_RABBITMQ_URL=f'amqp://cloudbox:{password}@rabbitmq:5672/',
                              GCS_MINIO_ENDPOINT='minio:9000',
                              GCS_STORAGE_LISTEN=f':{ports["storage"]}', GCS_GATEWAY_LISTEN=f':{ports["gateway"]}')
                    for peer in ['auth','user','file','share','notification']:
                        se['GCS_'+peer.upper()+'_URL'] = f'http://{peer}:{ports[peer]}'
                    target = './cmd/gateway' if role == 'gateway' else f'./services/{role}-service/cmd/server'
                    services[role] = {'build': {'context': str(ROOT), 'dockerfile': 'deploy/Dockerfile', 'args': {'TARGET': target}},
                                      'environment': se, 'ports': [f'127.0.0.1:{ports[role]}:{ports[role]}'],
                                      'volumes': [f'{file}:/app/conf/go-cloud-storage.dev.yaml:ro']}
                if args.local_runtime_image:
                    context = tmp/('image-'+role); context.mkdir()
                    (context/'service').write_bytes((tmp/role).read_bytes()); (context/'service').chmod(0o755)
                    (context/'Dockerfile').write_text((ROOT/'scripts/integration/Dockerfile.prebuilt').read_text())
                    services[role]['build'] = {'context':str(context),'args':{'RUNTIME_IMAGE':args.local_runtime_image}}
                service_env[role] = se
            if containers:
                manifest.write_text(json.dumps({'services': services}))
                print('Building seven application images...', flush=True)
                result = compose('build', *ROLES, check=False)
                (tmp/'build.log').write_text(result.stdout.replace(password, '[redacted]'))
                if result.returncode:
                    raise RuntimeError('Application image build failed; see build.log')
            for role in ROLES:
                start(role)
            print('Checking register/login/upload/range/share/notification...',flush=True)
            browser = Browser(f'http://127.0.0.1:{ports["gateway"]}')
            browser.call('/api/register',{'email':'smoke@example.test','password':'TestPassword123!','password_confirm':'TestPassword123!'})
            browser.call('/api/login',{'account':'smoke@example.test','password':'TestPassword123!'})
            root = browser.call('/api/me')['rootFolderId']
            payload = b'microservices smoke payload\n'
            fid = browser.upload(payload,root)['id']
            listing = browser.call('/api/file/list', {'parentId':root,'page':1,'pageSize':20})
            row = next(item for item in listing['list'] if item['id']==fid)
            created = datetime.fromisoformat(row['created_at'].replace('Z','+00:00'))
            assert created.tzinfo is not None and abs((datetime.now(timezone.utc)-created).total_seconds()) < 60, 'timestamp lost timezone'

            print('Checking favorites and database uniqueness...',flush=True)
            for _ in range(2): browser.call('/api/favorite/'+fid,method='POST')
            favorites = browser.call('/api/favorite')
            assert favorites['total']==1 and favorites['favoriteList'][0]['file_id']==fid
            duplicate_file = sql('cloudbox_file',f"INSERT INTO file (id,user_id,name,size,size_str,is_dir,file_extension,file_url,oss_object_key,file_hash,parent_id,is_deleted) SELECT UUID(),user_id,name,size,size_str,is_dir,file_extension,file_url,oss_object_key,file_hash,parent_id,is_deleted FROM file WHERE id='{fid}'",check=False)
            assert duplicate_file.returncode != 0 and 'uniq_user_parent_name' in duplicate_file.stdout
            assert browser.call('/api/file/preview-stream/'+fid,headers={'Range':'bytes=0-7'},expect=206)==payload[:8]
            share = browser.call('/api/share',{'file_id':fid,'expire_days':1})
            duplicate_share = sql('cloudbox_share',f"INSERT INTO share (user_id,file_id,share_token,file_version) SELECT user_id,file_id,UUID(),file_version FROM share WHERE file_id='{fid}'",check=False)
            assert duplicate_share.returncode != 0 and 'uk_share_file' in duplicate_share.stdout
            browser.call('/api/s/'+share['share_token'])
            wait(lambda: browser.call('/api/notification')['total']==2)
            denied = sql('cloudbox_file','SELECT password FROM cloudbox_user.user LIMIT 1',user='cloudbox_file',check=False)
            assert denied.returncode!=0 and 'denied' in denied.stdout.lower(), 'schema isolation failed'
            print('Checking durable notifications after consumer restart...',flush=True)
            stop('notification'); browser.upload(b'queued notification',root,'offline.txt'); start('notification')
            wait(lambda: browser.call('/api/notification')['total']==3)
            print('Checking delayed share invalidation and restore...',flush=True)
            stop('share'); browser.call('/api/file/'+fid,method='DELETE'); browser.call('/api/recycle/'+fid+'/restore',method='PUT'); start('share')
            try:
                browser.call('/api/s/'+share['share_token'])
                raise RuntimeError('revoked share was accepted')
            except AssertionError as error:
                assert error.args[0][1]!=200
            fresh = browser.call('/api/share',{'file_id':fid,'expire_days':1})
            time.sleep(2); browser.call('/api/s/'+fresh['share_token'])
            print('Checking chunk resume across file-service restart and direct download...',flush=True)
            chunk_size = 5 * 1024 * 1024
            chunk_data = b'A' * chunk_size + b'B' * 1024
            digest = hashlib.sha256(chunk_data).hexdigest()
            request = {'fileName':'chunks.txt', 'fileHash':digest, 'parentId':root, 'fileSize':len(chunk_data), 'chunkSize':chunk_size, 'totalChunks':2}
            browser.call('/api/file/chunk/init',request)
            for index, part in enumerate([chunk_data[:chunk_size],chunk_data[chunk_size:]]):
                browser.multipart('/api/file/chunk/upload',part,'part.bin','chunk',{'fileHash':digest,'chunkIndex':index,'chunkHash':hashlib.sha256(part).hexdigest()})
                if index == 0:
                    stop('file'); start('file')
            merged = browser.call('/api/file/chunk/merge',request)
            download = browser.call('/api/file/download-info/'+merged['id'])
            with urllib.request.urlopen(download['directDownloadUrl'],timeout=30) as response:
                assert hashlib.sha256(response.read()).hexdigest() == digest
            print('Checking RabbitMQ outage keeps uploads durable...',flush=True)
            wait(lambda: sql('cloudbox_file','SELECT COUNT(*) FROM event_outbox WHERE delivered_at IS NULL').stdout.strip()=='0')
            compose('stop','-t','10','rabbitmq')
            browser.upload(b'broker unavailable',root,'broker-offline.txt')
            assert int(sql('cloudbox_file','SELECT COUNT(*) FROM event_outbox WHERE delivered_at IS NULL').stdout.strip()) >= 1
            compose('start','rabbitmq')
            wait(lambda: int(sql('cloudbox_notification', "SELECT COUNT(*) FROM notification WHERE message LIKE '%broker-offline.txt%'").stdout.strip()) == 1,120)
            print('Checking storage failure does not consume quota...',flush=True)
            used = sql('cloudbox_file','SELECT SUM(used) FROM storage_quota').stdout.strip()
            stop('storage')
            browser.upload(b'failed storage upload',root,'failure.txt',business_code=500)
            assert sql('cloudbox_file','SELECT SUM(used) FROM storage_quota').stdout.strip()==used
            start('storage')
            print('Checking durable object cleanup across storage outage and file restart...',flush=True)
            doomed = browser.upload(b'cleanup retry object',root,'cleanup.txt')['id']
            direct = browser.call('/api/file/download-info/'+doomed)['directDownloadUrl']
            browser.call('/api/file/'+doomed,method='DELETE')
            stop('storage')
            browser.call('/api/recycle/'+doomed,method='DELETE')
            wait(lambda: int(sql('cloudbox_file','SELECT COUNT(*) FROM object_deletion WHERE attempts > 0').stdout.strip()) > 0)
            with urllib.request.urlopen(direct,timeout=10) as response: assert response.read()==b'cleanup retry object'
            stop('file'); start('file'); start('storage')
            wait(lambda: sql('cloudbox_file','SELECT COUNT(*) FROM object_deletion').stdout.strip()=='0')
            try:
                urllib.request.urlopen(direct,timeout=10)
                raise AssertionError('purged object still exists')
            except urllib.error.HTTPError as error:
                assert error.code==404
            print('Checking refresh outage preserves cookies...',flush=True)
            refresh = next(c.value for c in browser.cookies if c.name=='refresh_token')
            stop('user')
            browser.call('/api/refresh-token',method='POST',expect=503)
            assert next(c.value for c in browser.cookies if c.name=='refresh_token')==refresh
            start('user'); browser.call('/api/refresh-token',method='POST')
            print('Checking auth outage preserves the session...',flush=True)
            stop('auth')
            browser.call('/api/me',expect=503)
            start('auth')
            assert browser.call('/api/me')['rootFolderId']==root
            print('Checking database restart and connection recovery...',flush=True)
            compose('stop','-t','10','mysql')
            browser.call('/api/me',expect=503)
            browser.call('/api/refresh-token',method='POST',expect=503)
            compose('start','mysql')
            wait(mysql_ready)
            wait(lambda: browser.call('/api/me')['rootFolderId']==root)
            wait(lambda: browser.call('/api/file/preview-stream/'+fid)==payload)
            if args.migration_rehearsal:
                from migration_rehearsal import run as rehearse
                # Include a recycled row, favorite and active share in the fixture.
                browser.call('/api/favorite/'+fid,method='POST')
                recycled = browser.upload(b'migration recycle',root,'migration-recycle.txt')['id']
                browser.call('/api/file/'+recycled,method='DELETE')
                wait(lambda: sql('cloudbox_file','SELECT COUNT(*) FROM event_outbox WHERE delivered_at IS NULL').stdout.strip()=='0')
                def verify_migration():
                    migrated = Browser(browser.origin)
                    migrated.call('/api/login',{'account':'smoke@example.test','password':'TestPassword123!'})
                    assert migrated.call('/api/me')['rootFolderId']==root
                    assert migrated.call('/api/file/preview-stream/'+fid)==payload
                    migrated.call('/api/s/'+fresh['share_token'])
                    assert migrated.call('/api/favorite')['total']==1
                    migrated.call('/api/recycle/'+recycled+'/restore',method='PUT')
                    migrated.upload(b'new write after migration',root,'post-migration.txt')
                rehearse(sql,stop,start,ROLES,verify_migration)
                browser.call('/api/login',{'account':'smoke@example.test','password':'TestPassword123!'})
            print('Checking password change revokes sessions in every service...',flush=True)
            old = next(c.value for c in browser.cookies if c.name=='access_token')
            browser.call('/api/user/password',{'oldPassword':'TestPassword123!','newPassword':'ChangedPassword456!'},method='PUT')
            for path in ['/api/me','/api/user/quota','/api/share','/api/notification']:
                browser.call(path,headers={'Authorization':'Bearer '+old},expect=401)
            print('PASS: seven services, isolated databases, chunks/direct download, durable events, dependency recovery and session revocation.',flush=True)
            if args.hold_file:
                args.hold_file.write_text(json.dumps({'gateway':browser.origin,'project':base[3]}))
                args.hold_file.chmod(0o600)
                print(f'Environment ready for browser testing: {args.hold_file}',flush=True)
                deadline = time.monotonic()+1800
                while args.hold_file.exists() and time.monotonic()<deadline:
                    time.sleep(1)
                args.hold_file.unlink(missing_ok=True)
        except Exception:
            diagnostics = ROOT/'backend/tmp/microservices-smoke';diagnostics.mkdir(parents=True,exist_ok=True)
            for role in ROLES + ['build', 'compose-error']:
                source = tmp/f'{role}.log'
                if source.exists():
                    dest = diagnostics/f'{role}.log';dest.write_text(source.read_text(errors='replace').replace(password, '[redacted]'));dest.chmod(0o600)
            infra = compose('logs', '--no-color', '--tail', '100', check=False)
            target = diagnostics/'infrastructure.log'
            target.write_text(infra.stdout.replace(password, '[redacted]')); target.chmod(0o600)
            print(f'Failure logs saved locally: {diagnostics}',flush=True)
            raise
        finally:
            for role in list(processes):
                stop(role)
            compose('down','--volumes','--remove-orphans','--rmi','local',check=False)

if __name__=='__main__':
    main()
