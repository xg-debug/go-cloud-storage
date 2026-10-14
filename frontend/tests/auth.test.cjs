const assert = require('node:assert/strict')
const fs = require('node:fs')
const path = require('node:path')
const test = require('node:test')
function loadAuth(client) {
  const source = fs.readFileSync(path.join(__dirname, '../src/services/authSession.js'), 'utf8')
    .replace(/^import .*\n/gm, '').replace(/export /g, '')
  return new Function('axios', 'API_BASE_URL', source + '\nreturn {refreshSession,isSessionInvalid};')({create:()=>client}, '/')
}
test('refresh outage remains retryable; invalid session remains distinguishable', async () => {
  let reply = {response:{status:503}}
  let count = 0
  const auth = loadAuth({post:async()=>{count++; if (reply.response) throw reply; return reply}})
  await assert.rejects(auth.refreshSession(), e => !auth.isSessionInvalid(e))
  reply = {data:{code:200}}
  await auth.refreshSession()
  assert.equal(count,2)
  reply = {data:{code:401,message:'revoked'}}
  await assert.rejects(auth.refreshSession(), auth.isSessionInvalid)
  assert.equal(auth.isSessionInvalid(new Error('network disconnected')),false)
})
test('concurrent callers share refresh request', async () => {
  let finish, calls=0
  const auth=loadAuth({post:()=>{calls++;return new Promise(resolve=>{finish=resolve})}})
  const a=auth.refreshSession(), b=auth.refreshSession()
  assert.equal(a,b);assert.equal(calls,1)
  finish({data:{code:200}});await a
})

test('request interceptor preserves auth on transient refresh failure and clears revoked sessions', async () => {
  const source = fs.readFileSync(path.join(__dirname,'../src/utils/request.js'),'utf8')
    .replace(/^import .*\n/gm,'')
    .replace(/function showMessage\(type, message\) \{[\s\S]*?\n\}/, 'function showMessage() {}')
    .replace('export default service','return service')
  for (const invalid of [false,true]) {
    let rejected,commits=[],routes=[]
    const client={interceptors:{request:{use(){}},response:{use(ok,fail){rejected=fail}}}}
    new Function('axios','router','store','API_BASE_URL','refreshSession','isSessionInvalid',source)(
      {create:()=>client},{push:p=>routes.push(p)},{commit:(...args)=>commits.push(args)},'/',
      async()=>{throw new Error('refresh failure')},()=>invalid)
    await assert.rejects(rejected({response:{status:401},config:{}}))
    assert.equal(commits.some(c=>c[0]==='clearAuth'),invalid)
    assert.equal(routes.includes('/login'),invalid)
  }
})
