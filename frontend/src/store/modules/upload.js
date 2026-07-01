import { uploadFile, chunkUploadInit, chunkUploadPart, chunkUploadMerge, chunkUploadCancel } from '@/api/file'

const CHUNK_SIZE = 10 * 1024 * 1024
const CHUNK_THRESHOLD = 10 * 1024 * 1024
const MAX_CONCURRENT_CHUNKS = 3

let taskIdCounter = 0

function genId() {
  return `upload_${Date.now()}_${++taskIdCounter}`
}

async function calcSHA256(file) {
  const buffer = await file.arrayBuffer()
  const hashBuffer = await crypto.subtle.digest('SHA-256', buffer)
  return Array.from(new Uint8Array(hashBuffer))
    .map(b => b.toString(16).padStart(2, '0'))
    .join('')
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
        commit('UPDATE_TASK', { id: taskId, updates: { status: 'completed', progress: 100, file: null } })
      } catch (err) {
        if (err?.name === 'AbortError' || err?.message === 'cancelled') {
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

      const fileHash = await calcSHA256(task.file)
      commit('UPDATE_TASK', { id: taskId, updates: { fileHash } })

      await new Promise((resolve, reject) => {
        const form = new FormData()
        form.append('file', task.file)
        form.append('parentId', task.parentId)
        form.append('fileHash', fileHash)

        uploadFile(form, (e) => {
          const t = state.tasks.find(t => t.id === taskId)
          if (!t || t.status === 'paused' || t.status === 'cancelled') return
          const pct = Math.round((e.loaded * 100) / e.total)
          commit('UPDATE_TASK', { id: taskId, updates: { progress: pct } })
        }).then(() => resolve()).catch(reject)
      })
    },

    async uploadChunked({ commit, state, dispatch }, taskId) {
      let task = state.tasks.find(t => t.id === taskId)
      if (!task) return

      const fileHash = await calcSHA256(task.file)
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
        fileSize: task.fileSize
      })

      checkState()

      if (initRes.finished) {
        commit('UPDATE_TASK', { id: taskId, updates: { progress: 100 } })
        return
      }

      const uploaded = new Set(initRes.uploadedChunks || [])
      const totalChunks = Math.ceil(task.fileSize / CHUNK_SIZE)
      commit('UPDATE_TASK', { id: taskId, updates: { totalChunks, uploadedChunks: [...uploaded] } })

      // Build chunk list: skip already-uploaded
      const pendingChunks = []
      for (let i = 0; i < totalChunks; i++) {
        if (!uploaded.has(i)) pendingChunks.push(i)
      }

      let finishedCount = uploaded.size
      const updateProgress = () => {
        commit('UPDATE_TASK', { id: taskId, updates: { progress: Math.round((finishedCount / totalChunks) * 95) } })
      }
      updateProgress()

      // Upload with concurrency limit
      let activeCount = 0
      let chunkIdx = 0

      await new Promise((resolve, reject) => {
        const uploadNext = async () => {
          while (chunkIdx < pendingChunks.length && activeCount < MAX_CONCURRENT_CHUNKS) {
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

                await chunkUploadPart(form, () => {})

                checkState()
                finishedCount++
                updateProgress()
                commit('UPDATE_TASK', { id: taskId, updates: { uploadedChunks: [...state.tasks.find(t => t.id === taskId).uploadedChunks, idx] } })
              } catch (e) {
                reject(e)
                return
              } finally {
                activeCount--
                if (chunkIdx < pendingChunks.length) uploadNext()
                else if (activeCount === 0) resolve()
              }
            }
            doChunk()
          }
        }
        uploadNext()
      })

      checkState()

      // Merge
      commit('UPDATE_TASK', { id: taskId, updates: { progress: 98 } })
      await chunkUploadMerge({
        fileHash,
        fileName: task.fileName,
        fileSize: task.fileSize,
        parentId: task.parentId
      })

      checkState()
      commit('UPDATE_TASK', { id: taskId, updates: { progress: 100 } })
    },

    pauseTask({ commit, state }, taskId) {
      const task = state.tasks.find(t => t.id === taskId)
      if (!task || task.status !== 'uploading') return
      commit('UPDATE_TASK', { id: taskId, updates: { status: 'paused' } })
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

      commit('UPDATE_TASK', { id: taskId, updates: { status: 'cancelled', file: null, cancelController: null } })
      dispatch('processQueue')
    },

    retryTask({ commit, state, dispatch }, taskId) {
      const task = state.tasks.find(t => t.id === taskId)
      if (!task || (task.status !== 'failed' && task.status !== 'cancelled')) return
      commit('UPDATE_TASK', { id: taskId, updates: { status: 'pending', progress: 0, error: null, uploadedChunks: [], fileHash: '' } })
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
