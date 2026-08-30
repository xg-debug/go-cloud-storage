import { uploadFile, chunkUploadInit, chunkUploadPart, chunkUploadMerge, chunkUploadCancel } from '@/api/file'
import { sha256File } from '@/utils/sha256'

const CHUNK_SIZE = 10 * 1024 * 1024
const CHUNK_THRESHOLD = 10 * 1024 * 1024
const MAX_CONCURRENT_CHUNKS = 3
const HASH_CHUNK_SIZE = 4 * 1024 * 1024

let taskIdCounter = 0

function genId() {
  return `upload_${Date.now()}_${++taskIdCounter}`
}

function isAbortError(err) {
  return err?.name === 'AbortError' || err?.name === 'CanceledError' || err?.code === 'ERR_CANCELED' || err?.message === 'cancelled' || err?.message === 'canceled'
}

function getAbortSignal(state, taskId) {
  return state.tasks.find(t => t.id === taskId)?.cancelController?.signal
}

// 上传请求统一静默错误（业务错误由队列面板展示，避免 toast 刷屏）
function silentConfig(state, taskId) {
  return { signal: getAbortSignal(state, taskId), silentError: true }
}

async function calcSHA256(file, state, taskId, commit, progressStart = 0, progressSpan = 5) {
  return sha256File(file, {
    chunkSize: HASH_CHUNK_SIZE,
    signal: getAbortSignal(state, taskId),
    onProgress: ratio => {
      const t = state.tasks.find(t => t.id === taskId)
      if (!t || t.status === 'paused' || t.status === 'cancelled') return
      commit('UPDATE_TASK', { id: taskId, updates: { progress: Math.round(progressStart + ratio * progressSpan) } })
    }
  })
}

export default {
  namespaced: true,
  state: {
    tasks: [],
    activeCount: 0,
    maxConcurrent: 2
  },
  getters: {
    hasActive: state => state.tasks.some(t => t.status === 'uploading' || t.status === 'pending'),
    pendingCount: state => state.tasks.filter(t => t.status === 'pending').length,
    uploadingCount: state => state.tasks.filter(t => t.status === 'uploading').length
  },
  mutations: {
    ADD_TASK(state, task) {
      state.tasks.unshift(task)
    },
    UPDATE_TASK(state, { id, updates }) {
      const idx = state.tasks.findIndex(t => t.id === id)
      if (idx !== -1) {
        state.tasks[idx] = { ...state.tasks[idx], ...updates }
      }
    },
    REMOVE_TASK(state, id) {
      state.tasks = state.tasks.filter(t => t.id !== id)
    },
    SET_ACTIVE_COUNT(state, count) {
      state.activeCount = count
    }
  },
  actions: {
    async addToQueue({ commit, dispatch }, { files, parentId }) {
      for (const file of files) {
        if (!file || !file.name) continue
        const isChunked = file.size >= CHUNK_THRESHOLD
        const task = {
          id: genId(),
          fileName: file.name,
          fileSize: file.size,
          progress: 0,
          status: 'pending',
          type: isChunked ? 'chunked' : 'normal',
          file,
          parentId,
          fileHash: '',
          uploadedChunks: [],
          totalChunks: isChunked ? Math.ceil(file.size / CHUNK_SIZE) : 0,
          error: null,
          cancelController: null,
          chunkQueue: []
        }
        commit('ADD_TASK', task)
      }
      dispatch('processQueue')
    },

    async processQueue({ state, dispatch, commit }) {
      const pending = state.tasks.filter(t => t.status === 'pending')
      const active = state.tasks.filter(t => t.status === 'uploading').length
      const slots = state.maxConcurrent - active
      for (let i = 0; i < Math.min(slots, pending.length); i++) {
        dispatch('uploadTask', pending[i].id)
      }
      // 仅当有任务上传成功时才自动刷新文件列表
      if (state.tasks.length > 0) {
        const hasActive = state.tasks.some(t => t.status === 'uploading' || t.status === 'pending')
        if (!hasActive) {
          const hasCompleted = state.tasks.some(t => t.status === 'completed')
          if (hasCompleted) {
            commit('file/setNeedRefresh', true, { root: true })
            commit('file/setNeedRefreshStorage', true, { root: true })
          }
        }
      }
    },

    async uploadTask({ commit, dispatch, state }, taskId) {
      const task = state.tasks.find(t => t.id === taskId)
      if (!task || task.status === 'uploading' || task.status === 'completed') return

      const cancelController = new AbortController()
      commit('UPDATE_TASK', { id: taskId, updates: { status: 'uploading', error: null, cancelController } })

      try {
        if (task.type === 'chunked') {
          await dispatch('uploadChunked', taskId)
        } else {
          await dispatch('uploadNormal', taskId)
        }
        // 成功后释放大对象与分片列表引用（仅保留任务摘要用于展示）
        commit('UPDATE_TASK', { id: taskId, updates: { status: 'completed', progress: 100, file: null, uploadedChunks: [], cancelController: null } })
      } catch (err) {
        const latestTask = state.tasks.find(t => t.id === taskId)
        if (latestTask?.status === 'paused') {
          commit('UPDATE_TASK', { id: taskId, updates: { cancelController: null } })
        } else if (isAbortError(err)) {
          commit('UPDATE_TASK', { id: taskId, updates: { status: 'cancelled' } })
        } else if (err?.message === 'paused') {
          // status already set by pauseTask
        } else {
          commit('UPDATE_TASK', { id: taskId, updates: { status: 'failed', error: err?.message || '上传失败' } })
        }
      } finally {
        dispatch('processQueue')
      }
    },

    async uploadNormal({ commit, state }, taskId) {
      const task = state.tasks.find(t => t.id === taskId)
      if (!task) return

      const fileHash = await calcSHA256(task.file, state, taskId, commit, 0, 5)
      commit('UPDATE_TASK', { id: taskId, updates: { fileHash } })

      await new Promise((resolve, reject) => {
        const form = new FormData()
        form.append('file', task.file)
        form.append('parentId', task.parentId)
        form.append('fileHash', fileHash)

        uploadFile(form, (e) => {
          const t = state.tasks.find(t => t.id === taskId)
          if (!t || t.status === 'paused' || t.status === 'cancelled') return
          const pct = 5 + Math.round((e.loaded * 95) / e.total)
          commit('UPDATE_TASK', { id: taskId, updates: { progress: pct } })
        }, silentConfig(state, taskId)).then(() => resolve()).catch(reject)
      })
    },

    async uploadChunked({ commit, state, dispatch }, taskId) {
      let task = state.tasks.find(t => t.id === taskId)
      if (!task) return

      const fileHash = await calcSHA256(task.file, state, taskId, commit, 0, 5)
      commit('UPDATE_TASK', { id: taskId, updates: { fileHash } })

      // Check for pause/cancel
      const checkState = () => {
        const t = state.tasks.find(t => t.id === taskId)
        if (!t) throw new Error('cancelled')
        if (t.status === 'paused') throw new Error('paused')
        if (t.status === 'cancelled') throw new Error('cancelled')
      }

      checkState()

      // Init
      const initRes = await chunkUploadInit({
        fileHash,
        parentId: task.parentId,
        fileName: task.fileName,
        fileSize: task.fileSize,
        chunkSize: CHUNK_SIZE,
        totalChunks: Math.ceil(task.fileSize / CHUNK_SIZE)
      }, silentConfig(state, taskId))

      checkState()

      if (initRes.finished) {
        commit('UPDATE_TASK', { id: taskId, updates: { progress: 100 } })
        return
      }

      // 服务端返回已上传分片（断点续传：会话内/页面内重试时跳过已传分片）
      const uploadedSet = new Set(initRes.uploadedChunks || [])
      const totalChunks = Math.ceil(task.fileSize / CHUNK_SIZE)
      commit('UPDATE_TASK', { id: taskId, updates: { totalChunks, uploadedChunks: [...uploadedSet] } })

      // Build chunk list: skip already-uploaded
      const pendingChunks = []
      for (let i = 0; i < totalChunks; i++) {
        if (!uploadedSet.has(i)) pendingChunks.push(i)
      }

      let finishedCount = uploadedSet.size
      const updateProgress = () => {
        commit('UPDATE_TASK', { id: taskId, updates: { progress: 5 + Math.round((finishedCount / totalChunks) * 90) } })
      }
      updateProgress()

      // 单个分片上传（含瞬时错误自动重试）
      const uploadChunkWithRetry = async (form) => {
        const maxRetries = 2
        const delays = [1000, 3000]
        for (let attempt = 0; ; attempt++) {
          try {
            await chunkUploadPart(form, () => {}, silentConfig(state, taskId))
            return
          } catch (e) {
            checkState() // 暂停/取消优先于重试
            if (isAbortError(e) || e?.message === 'paused' || e?.message === 'cancelled') throw e
            if (attempt >= maxRetries) throw e
            await new Promise(res => setTimeout(res, delays[attempt]))
            checkState()
          }
        }
      }

      // Upload with concurrency limit
      let activeCount = 0
      let chunkIdx = 0
      let stopped = false

      // 关键修复：所有分片已上传（仅剩合并）时不能进入并发循环，
      // 否则 Promise 永不 resolve，任务卡死并阻塞整个上传队列。
      if (pendingChunks.length > 0) {
        await new Promise((resolve, reject) => {
          const uploadNext = async () => {
            while (!stopped && chunkIdx < pendingChunks.length && activeCount < MAX_CONCURRENT_CHUNKS) {
              const idx = pendingChunks[chunkIdx++]
              activeCount++

              const doChunk = async () => {
                try {
                  checkState()
                  const start = idx * CHUNK_SIZE
                  const end = Math.min(task.file.size, start + CHUNK_SIZE)
                  const chunk = task.file.slice(start, end)

                  const form = new FormData()
                  form.append('fileHash', fileHash)
                  form.append('chunkIndex', idx)
                  form.append('chunk', chunk)

                  await uploadChunkWithRetry(form)

                  checkState()
                  uploadedSet.add(idx)
                  finishedCount++
                  updateProgress()
                  commit('UPDATE_TASK', {
                    id: taskId,
                    updates: { uploadedChunks: [...uploadedSet].sort((a, b) => a - b) }
                  })
                } catch (e) {
                  // 最终失败：取消在途分片，避免继续浪费带宽
                  const t = state.tasks.find(t => t.id === taskId)
                  if (t && t.status !== 'paused' && t.status !== 'cancelled') {
                    t.cancelController?.abort()
                  }
                  stopped = true
                  reject(e)
                  return
                } finally {
                  activeCount--
                  if (!stopped && chunkIdx < pendingChunks.length) uploadNext()
                  else if (activeCount === 0 && !stopped) resolve()
                }
              }
              doChunk()
            }
          }
          uploadNext()
        })
      }

      checkState()

      // Merge
      commit('UPDATE_TASK', { id: taskId, updates: { progress: 98 } })
      await chunkUploadMerge({
        fileHash,
        fileName: task.fileName,
        fileSize: task.fileSize,
        parentId: task.parentId,
        totalChunks,
        chunkSize: CHUNK_SIZE
      }, silentConfig(state, taskId))

      checkState()
      commit('UPDATE_TASK', { id: taskId, updates: { progress: 100 } })
    },

    pauseTask({ commit, state }, taskId) {
      const task = state.tasks.find(t => t.id === taskId)
      if (!task || task.status !== 'uploading') return
      commit('UPDATE_TASK', { id: taskId, updates: { status: 'paused' } })
      if (task.cancelController) {
        task.cancelController.abort()
      }
    },

    resumeTask({ commit, state, dispatch }, taskId) {
      const task = state.tasks.find(t => t.id === taskId)
      if (!task || task.status !== 'paused') return
      commit('UPDATE_TASK', { id: taskId, updates: { status: 'pending' } })
      dispatch('processQueue')
    },

    cancelTask({ commit, state, dispatch }, taskId) {
      const task = state.tasks.find(t => t.id === taskId)
      if (!task) return

      if (task.status === 'uploading' && task.fileHash) {
        // Attempt server-side cancel for chunked uploads
        chunkUploadCancel(task.fileHash).catch(() => {})
      }

      if (task.cancelController) {
        task.cancelController.abort()
      }

      // 注意：不置空 file —— 取消后用户可能重试，需要保留文件引用。
      // 文件引用在任务成功完成或从队列移除时才释放。
      commit('UPDATE_TASK', { id: taskId, updates: { status: 'cancelled', cancelController: null } })
      dispatch('processQueue')
    },

    retryTask({ commit, state, dispatch }, taskId) {
      const task = state.tasks.find(t => t.id === taskId)
      if (!task || (task.status !== 'failed' && task.status !== 'cancelled')) return
      if (!task.file) {
        // 兜底：文件引用已丢失（如页面刷新后遗留的任务）
        commit('UPDATE_TASK', { id: taskId, updates: { status: 'failed', error: '原文件不可用，请重新添加后再上传' } })
        return
      }
      commit('UPDATE_TASK', { id: taskId, updates: { status: 'pending', progress: 0, error: null, uploadedChunks: [], fileHash: '', cancelController: null } })
      dispatch('processQueue')
    },

    removeTask({ commit, state, dispatch }, taskId) {
      const task = state.tasks.find(t => t.id === taskId)
      if (!task) return
      if (task.status === 'uploading') {
        dispatch('cancelTask', taskId)
      }
      commit('REMOVE_TASK', taskId)
    },

    clearCompleted({ commit, state }) {
      state.tasks.filter(t => t.status === 'completed' || t.status === 'cancelled').forEach(t => {
        commit('REMOVE_TASK', t.id)
      })
    }
  }
}
