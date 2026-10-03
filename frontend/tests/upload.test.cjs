const assert = require('node:assert/strict')
const fs = require('node:fs')
const path = require('node:path')
const test = require('node:test')

function deferred() {
  let resolve
  let reject
  const promise = new Promise((res, rej) => {
    resolve = res
    reject = rej
  })
  return { promise, resolve, reject }
}

function wait(ms = 0) {
  return new Promise(resolve => setTimeout(resolve, ms))
}

async function eventually(fn, timeout = 500) {
  const start = Date.now()
  for (;;) {
    try {
      fn()
      return
    } catch (err) {
      if (Date.now() - start > timeout) throw err
      await wait(5)
    }
  }
}

function loadUploadModule(mocks) {
  const filePath = path.join(__dirname, '..', 'src', 'store', 'modules', 'upload.js')
  let source = fs.readFileSync(filePath, 'utf8')
  source = source
    .replace("import { uploadFile, chunkUploadInit, chunkUploadPart, chunkUploadMerge, chunkUploadCancel } from '@/api/file'\n", '')
    .replace("import { sha256File } from '@/utils/sha256'\n", '')
    .replace('export default', 'module.exports =')

  const module = { exports: {} }
  const factory = new Function(
    'module',
    'exports',
    'uploadFile',
    'chunkUploadInit',
    'chunkUploadPart',
    'chunkUploadMerge',
    'chunkUploadCancel',
    'sha256File',
    source
  )
  factory(
    module,
    module.exports,
    mocks.uploadFile,
    mocks.chunkUploadInit,
    mocks.chunkUploadPart,
    mocks.chunkUploadMerge,
    mocks.chunkUploadCancel,
    mocks.sha256File
  )
  return module.exports
}

function makeFile(size, name = 'demo.bin') {
  return {
    name,
    size,
    slice(start, end) {
      return { start, end, size: end - start }
    }
  }
}

function createHarness(uploadModule) {
  const state = JSON.parse(JSON.stringify(uploadModule.state))
  const context = {
    state,
    commit(type, payload) {
      if (uploadModule.mutations[type]) {
        uploadModule.mutations[type](state, payload)
      }
    },
    dispatch(type, payload) {
      return Promise.resolve(uploadModule.actions[type](context, payload))
    }
  }
  return { state, context }
}

test('pause and immediate resume do not start a duplicate normal upload run', async () => {
  const uploads = []
  const firstUpload = deferred()
  const secondUpload = deferred()
  const uploadModule = loadUploadModule({
    sha256File: async () => 'hash-normal',
    uploadFile: (form, onProgress, config) => {
      const run = uploads.length === 0 ? firstUpload : secondUpload
      uploads.push({ form, config })
      return run.promise
    },
    chunkUploadInit: async () => ({ finished: false, uploadedChunks: [] }),
    chunkUploadPart: async () => {},
    chunkUploadMerge: async () => {},
    chunkUploadCancel: async () => {}
  })
  const { state, context } = createHarness(uploadModule)

  await uploadModule.actions.addToQueue(context, { files: [makeFile(1024)], parentId: 'root' })
  await eventually(() => assert.equal(uploads.length, 1))
  const taskId = state.tasks[0].id

  uploadModule.actions.pauseTask(context, taskId)
  uploadModule.actions.resumeTask(context, taskId)
  await wait()
  assert.equal(uploads.length, 1)

  firstUpload.reject(new DOMException('Aborted', 'AbortError'))
  await eventually(() => assert.equal(uploads.length, 2))
  secondUpload.resolve({})
  await eventually(() => assert.equal(state.tasks[0].status, 'completed'))
})

test('chunked retry waits until all stale parallel chunks have settled', async () => {
  const partRuns = []
  let initCalls = 0
  const mergeRun = deferred()
  const uploadModule = loadUploadModule({
    sha256File: async () => 'hash-chunked',
    uploadFile: async () => {},
    chunkUploadInit: async () => {
      initCalls += 1
      return { finished: false, uploadedChunks: [] }
    },
    chunkUploadPart: () => {
      const run = deferred()
      partRuns.push(run)
      return run.promise
    },
    chunkUploadMerge: () => mergeRun.promise,
    chunkUploadCancel: async () => {}
  })
  const { state, context } = createHarness(uploadModule)

  await uploadModule.actions.addToQueue(context, { files: [makeFile(30 * 1024 * 1024)], parentId: 'root' })
  await eventually(() => assert.equal(partRuns.length, 3))
  const taskId = state.tasks[0].id

  uploadModule.actions.cancelTask(context, taskId)
  await wait()
  uploadModule.actions.retryTask(context, taskId)
  await wait()
  assert.equal(initCalls, 1)
  assert.equal(state.tasks[0].status, 'pending')

  partRuns[0].resolve({})
  partRuns[1].resolve({})
  partRuns[2].resolve({})
  await eventually(() => assert.equal(initCalls, 2))

  partRuns.slice(3).forEach(run => run.resolve({}))
  mergeRun.resolve({})
  await eventually(() => assert.equal(state.tasks[0].status, 'completed'))
})

test('cancelAll waits for known chunk upload cleanup before logout continues', async () => {
  const cleanup = deferred()
  const uploadModule = loadUploadModule({
    sha256File: async () => 'hash-chunked',
    uploadFile: async () => {},
    chunkUploadInit: async () => ({ finished: false, uploadedChunks: [] }),
    chunkUploadPart: async () => {},
    chunkUploadMerge: async () => {},
    chunkUploadCancel: () => cleanup.promise
  })
  const { state, context } = createHarness(uploadModule)
  state.tasks.push({
    id: 'task-1',
    status: 'uploading',
    running: true,
    type: 'chunked',
    fileHash: 'hash-chunked',
    cancelController: new AbortController()
  })

  let done = false
  const cancelAll = uploadModule.actions.cancelAll(context).then(() => { done = true })
  await wait()
  assert.equal(done, false)
  assert.equal(state.tasks[0].cancelController.signal.aborted, true)
  cleanup.resolve({})
  await cancelAll
  assert.equal(done, true)
})
