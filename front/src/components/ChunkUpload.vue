<template>
  <div class="chunk-upload-container">
    <!-- 上传区域 -->
    <div 
      class="upload-area"
      :class="{ 
        'drag-over': isDragOver,
        'uploading': isUploading,
        'has-file': selectedFile
      }"
      @drop="handleDrop"
      @dragover="handleDragOver"
      @dragleave="handleDragLeave"
      @click="triggerFileSelect"
    >
      <input
        ref="fileInput"
        type="file"
        style="display: none"
        @change="handleFileSelect"
        :accept="acceptedTypes"
      />
      
      <div v-if="!selectedFile" class="upload-placeholder">
        <el-icon class="upload-icon" size="48">
          <Upload />
        </el-icon>
        <div class="upload-text">
          <p class="primary-text">点击选择文件或拖拽文件到此处</p>
          <p class="secondary-text">支持大文件上传，自动分片处理</p>
        </div>
      </div>

      <div v-else class="file-info">
        <div class="file-details">
          <el-icon class="file-icon" size="32">
            <Document />
          </el-icon>
          <div class="file-meta">
            <p class="file-name">{{ selectedFile.name }}</p>
            <p class="file-size">{{ formatFileSize(selectedFile.size) }}</p>
          </div>
        </div>
        
        <div class="upload-progress" v-if="uploadProgress > 0">
          <el-progress 
            :percentage="uploadProgress" 
            :status="uploadStatus"
            :stroke-width="8"
            :show-text="false"
          />
          <div class="progress-info">
            <span class="progress-text">{{ uploadProgress }}%</span>
            <span class="speed-text" v-if="uploadSpeed">{{ uploadSpeed }}</span>
          </div>
        </div>

        <div class="upload-actions">
          <el-button 
            v-if="!isUploading && uploadProgress === 0"
            type="primary" 
            @click="startUpload"
            :loading="isInitializing"
          >
            {{ isInitializing ? '初始化中...' : '开始上传' }}
          </el-button>
          
          <el-button 
            v-if="isUploading && !isPaused"
            type="warning" 
            @click="pauseUpload"
          >
            暂停上传
          </el-button>
          
          <el-button 
            v-if="isPaused"
            type="success" 
            @click="resumeUpload"
          >
            继续上传
          </el-button>
          
          <el-button 
            v-if="isUploading || isPaused"
            type="danger" 
            @click="cancelUpload"
          >
            取消上传
          </el-button>
          
          <el-button 
            v-if="uploadProgress === 100"
            type="info" 
            @click="resetUpload"
          >
            重新上传
          </el-button>
        </div>
      </div>
    </div>

    <!-- 未完成的上传任务 -->
    <div v-if="incompleteTasks.length > 0" class="incomplete-tasks">
      <h3>未完成的上传</h3>
      <div class="task-list">
        <div 
          v-for="task in incompleteTasks" 
          :key="task.id"
          class="task-item"
        >
          <div class="task-info">
            <el-icon class="task-icon">
              <Document />
            </el-icon>
            <div class="task-details">
              <p class="task-name">{{ task.file_name }}</p>
              <p class="task-size">{{ formatFileSize(task.file_size) }}</p>
            </div>
          </div>
          <div class="task-progress">
            <el-progress 
              :percentage="calculateTaskProgress(task)" 
              :stroke-width="6"
              :show-text="false"
            />
            <span class="task-progress-text">{{ calculateTaskProgress(task) }}%</span>
          </div>
          <div class="task-actions">
            <el-button 
              type="primary" 
              size="small" 
              @click="resumeTask(task)"
            >
              继续
            </el-button>
            <el-button 
              type="danger" 
              size="small" 
              @click="deleteTask(task.id)"
            >
              删除
            </el-button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted, computed } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Upload, Document } from '@element-plus/icons-vue'
import SparkMD5 from 'spark-md5'
import { 
  createUploadTask, 
  getUploadTask,
  getChunkURL, 
  markChunkUploaded, 
  completeUpload,
  getIncompleteTasks,
  deleteUploadTask
} from '@/api/upload'

// 响应式数据
const selectedFile = ref(null)
const isDragOver = ref(false)
const isUploading = ref(false)
const isPaused = ref(false)
const isInitializing = ref(false)
const uploadProgress = ref(0)
const uploadStatus = ref('')
const fileInput = ref(null)
const incompleteTasks = ref([])

// 上传配置
const CHUNK_SIZE = 5 * 1024 * 1024 // 5MB
const CONCURRENT_UPLOADS = 3
const acceptedTypes = '*'

// 上传状态
const uploadState = reactive({
  taskId: '',
  fileHash: '',
  chunks: [],
  uploadedChunks: new Set(),
  activeUploads: new Map(),
  startTime: 0,
  uploadedBytes: 0
})

// 计算属性
const uploadSpeed = computed(() => {
  if (!uploadState.startTime || uploadState.uploadedBytes === 0) return ''
  const elapsed = (Date.now() - uploadState.startTime) / 1000
  const speed = uploadState.uploadedBytes / elapsed
  return formatSpeed(speed)
})

// 生命周期
onMounted(() => {
  loadIncompleteTasks()
})

// 文件选择和拖拽处理
function triggerFileSelect() {
  if (!isUploading.value) {
    fileInput.value.click()
  }
}

function handleFileSelect(event) {
  const file = event.target.files[0]
  if (file) {
    selectFile(file)
  }
}

function handleDrop(event) {
  event.preventDefault()
  isDragOver.value = false
  
  if (isUploading.value) return
  
  const files = event.dataTransfer.files
  if (files.length > 0) {
    selectFile(files[0])
  }
}

function handleDragOver(event) {
  event.preventDefault()
  isDragOver.value = true
}

function handleDragLeave(event) {
  event.preventDefault()
  isDragOver.value = false
}

function selectFile(file) {
  selectedFile.value = file
  uploadProgress.value = 0
  uploadStatus.value = ''
  resetUploadState()
}

// 上传逻辑
async function startUpload() {
  if (!selectedFile.value) return
  
  try {
    isInitializing.value = true
    
    // 计算文件哈希
    uploadState.fileHash = await calculateFileHash(selectedFile.value)
    
    // 准备分片
    prepareChunks(selectedFile.value)
    
    // 初始化上传任务
    const response = await createUploadTask({
      fileName: selectedFile.value.name,
      fileSize: selectedFile.value.size,
      fileHash: uploadState.fileHash,
      chunkSize: CHUNK_SIZE,
      userId: getUserId()
    })
    
    uploadState.taskId = response.id
    
    // 检查已上传的分片
    if (response.uploaded_chunks && response.uploaded_chunks.length > 0) {
      response.uploaded_chunks.forEach(chunk => {
        uploadState.uploadedChunks.add(chunk.index)
      })
      updateProgress()
    }
    
    isInitializing.value = false
    isUploading.value = true
    uploadState.startTime = Date.now()
    
    // 开始上传
    await uploadChunks()
    
  } catch (error) {
    console.error('上传初始化失败:', error)
    ElMessage.error('上传初始化失败: ' + error.message)
    isInitializing.value = false
  }
}

async function uploadChunks() {
  const pendingChunks = uploadState.chunks.filter(
    chunk => !uploadState.uploadedChunks.has(chunk.index + 1)
  )
  
  if (pendingChunks.length === 0) {
    await completeFileUpload()
    return
  }
  
  // 并发上传
  const uploadPromises = []
  let chunkIndex = 0
  
  for (let i = 0; i < CONCURRENT_UPLOADS && chunkIndex < pendingChunks.length; i++) {
    uploadPromises.push(uploadWorker(pendingChunks, chunkIndex))
    chunkIndex++
  }
  
  try {
    await Promise.all(uploadPromises)
    
    if (!isPaused.value && uploadProgress.value === 100) {
      await completeFileUpload()
    }
  } catch (error) {
    console.error('上传过程中出错:', error)
    ElMessage.error('上传失败: ' + error.message)
    isUploading.value = false
  }
}

async function uploadWorker(chunks, startIndex) {
  let index = startIndex
  
  while (index < chunks.length && isUploading.value && !isPaused.value) {
    const chunk = chunks[index]
    const partNumber = chunk.index + 1
    
    if (uploadState.uploadedChunks.has(partNumber)) {
      index += CONCURRENT_UPLOADS
      continue
    }
    
    try {
      // 获取预签名URL
      const urlResponse = await getChunkURL(uploadState.taskId, partNumber)
      const uploadUrl = urlResponse.data.url
      
      // 上传分片
      const etag = await uploadChunkToOSS(uploadUrl, chunk.data, partNumber)
      
      // 标记分片完成
      await markChunkUploaded(uploadState.taskId, partNumber, { etag })
      
      uploadState.uploadedChunks.add(partNumber)
      updateProgress()
      
    } catch (error) {
      console.error(`分片 ${partNumber} 上传失败:`, error)
      throw error
    }
    
    index += CONCURRENT_UPLOADS
  }
}

async function uploadChunkToOSS(url, data, partNumber) {
  return new Promise((resolve, reject) => {
    const xhr = new XMLHttpRequest()
    let lastLoaded = 0
    
    xhr.open('PUT', url, true)
    
    xhr.upload.onprogress = (event) => {
      if (event.lengthComputable) {
        const delta = event.loaded - lastLoaded
        lastLoaded = event.loaded
        uploadState.uploadedBytes += delta
      }
    }
    
    xhr.onload = () => {
      if (xhr.status >= 200 && xhr.status < 300) {
        const etag = xhr.getResponseHeader('etag')
        resolve(etag)
      } else {
        reject(new Error(`上传失败，状态码: ${xhr.status}`))
      }
    }
    
    xhr.onerror = () => {
      reject(new Error('网络错误'))
    }
    
    uploadState.activeUploads.set(partNumber, xhr)
    xhr.send(data)
  })
}

async function completeFileUpload() {
  try {
    await completeUpload(uploadState.taskId)
    uploadProgress.value = 100
    uploadStatus.value = 'success'
    isUploading.value = false
    ElMessage.success('文件上传成功！')
    loadIncompleteTasks()
  } catch (error) {
    console.error('完成上传失败:', error)
    ElMessage.error('完成上传失败: ' + error.message)
  }
}

// 上传控制
function pauseUpload() {
  isPaused.value = true
  isUploading.value = false
  
  // 取消正在进行的上传
  uploadState.activeUploads.forEach(xhr => {
    xhr.abort()
  })
  uploadState.activeUploads.clear()
  
  ElMessage.info('上传已暂停')
}

function resumeUpload() {
  isPaused.value = false
  isUploading.value = true
  uploadState.startTime = Date.now()
  uploadChunks()
}

async function cancelUpload() {
  try {
    await ElMessageBox.confirm('确定要取消上传吗？', '确认', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning'
    })
    
    // 取消所有上传
    uploadState.activeUploads.forEach(xhr => {
      xhr.abort()
    })
    
    resetUpload()
    ElMessage.info('上传已取消')
  } catch {
    // 用户取消确认
  }
}

function resetUpload() {
  selectedFile.value = null
  uploadProgress.value = 0
  uploadStatus.value = ''
  isUploading.value = false
  isPaused.value = false
  isInitializing.value = false
  resetUploadState()
  
  // 清空文件输入
  if (fileInput.value) {
    fileInput.value.value = ''
  }
}

function resetUploadState() {
  uploadState.taskId = ''
  uploadState.fileHash = ''
  uploadState.chunks = []
  uploadState.uploadedChunks.clear()
  uploadState.activeUploads.clear()
  uploadState.startTime = 0
  uploadState.uploadedBytes = 0
}

// 工具函数
function prepareChunks(file) {
  const chunks = []
  const totalChunks = Math.ceil(file.size / CHUNK_SIZE)
  
  for (let i = 0; i < totalChunks; i++) {
    const start = i * CHUNK_SIZE
    const end = Math.min(file.size, start + CHUNK_SIZE)
    chunks.push({
      index: i,
      data: file.slice(start, end)
    })
  }
  
  uploadState.chunks = chunks
}

async function calculateFileHash(file) {
  return new Promise((resolve) => {
    const spark = new SparkMD5.ArrayBuffer()
    const reader = new FileReader()
    const chunkSize = 2 * 1024 * 1024 // 2MB for hash
    const chunks = Math.ceil(file.size / chunkSize)
    let currentChunk = 0
    
    reader.onload = (e) => {
      spark.append(e.target.result)
      currentChunk++
      
      if (currentChunk < chunks) {
        loadNext()
      } else {
        resolve(spark.end())
      }
    }
    
    const loadNext = () => {
      const start = currentChunk * chunkSize
      const end = Math.min(file.size, start + chunkSize)
      reader.readAsArrayBuffer(file.slice(start, end))
    }
    
    loadNext()
  })
}

function updateProgress() {
  const total = uploadState.chunks.length
  const uploaded = uploadState.uploadedChunks.size
  uploadProgress.value = Math.floor((uploaded / total) * 100)
}

function formatFileSize(bytes) {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
}

function formatSpeed(bytesPerSecond) {
  return formatFileSize(bytesPerSecond) + '/s'
}

function getUserId() {
  // 从store或localStorage获取用户ID
  return JSON.parse(localStorage.getItem('user'))?.id || 1
}

// 未完成任务管理
async function loadIncompleteTasks() {
  try {
    const response = await getIncompleteTasks()
    incompleteTasks.value = response || []
  } catch (error) {
    console.error('加载未完成任务失败:', error)
  }
}

function calculateTaskProgress(task) {
  if (!task.uploaded_chunks || task.chunk_count === 0) return 0
  return Math.floor((task.uploaded_chunks.length / task.chunk_count) * 100)
}

async function resumeTask(task) {
  try {
    // 提示用户选择相同文件
    ElMessage.info('请选择相同的文件以继续上传')
    
    // 设置任务信息
    uploadState.taskId = task.id
    uploadState.fileHash = task.file_hash
    
    // 标记已上传的分片
    if (task.uploaded_chunks) {
      task.uploaded_chunks.forEach(chunk => {
        uploadState.uploadedChunks.add(chunk.index)
      })
    }
    
    // 触发文件选择
    fileInput.value.click()
  } catch (error) {
    console.error('恢复任务失败:', error)
    ElMessage.error('恢复任务失败')
  }
}

async function deleteTask(taskId) {
  try {
    await ElMessageBox.confirm('确定要删除这个上传任务吗？', '确认', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning'
    })
    
    await deleteUploadTask(taskId)
    
    incompleteTasks.value = incompleteTasks.value.filter(task => task.id !== taskId)
    ElMessage.success('任务已删除')
  } catch {
    // 用户取消确认
  }
}
</script>

<style scoped>
.chunk-upload-container {
  max-width: 800px;
  margin: 0 auto;
  padding: 20px;
}

.upload-area {
  border: 2px dashed #dcdfe6;
  border-radius: 8px;
  padding: 40px 20px;
  text-align: center;
  cursor: pointer;
  transition: all 0.3s ease;
  background-color: #fafafa;
}

.upload-area:hover {
  border-color: #409eff;
  background-color: #f0f9ff;
}

.upload-area.drag-over {
  border-color: #409eff;
  background-color: #e6f7ff;
  transform: scale(1.02);
}

.upload-area.uploading {
  cursor: not-allowed;
  opacity: 0.8;
}

.upload-area.has-file {
  border-color: #67c23a;
  background-color: #f0f9ff;
}

.upload-placeholder {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 16px;
}

.upload-icon {
  color: #909399;
}

.upload-text .primary-text {
  font-size: 16px;
  color: #303133;
  margin: 0 0 8px 0;
  font-weight: 500;
}

.upload-text .secondary-text {
  font-size: 14px;
  color: #909399;
  margin: 0;
}

.file-info {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.file-details {
  display: flex;
  align-items: center;
  gap: 12px;
  justify-content: center;
}

.file-icon {
  color: #409eff;
}

.file-meta {
  text-align: left;
}

.file-name {
  font-size: 16px;
  font-weight: 500;
  color: #303133;
  margin: 0 0 4px 0;
  word-break: break-all;
}

.file-size {
  font-size: 14px;
  color: #909399;
  margin: 0;
}

.upload-progress {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.progress-info {
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-size: 14px;
}

.progress-text {
  color: #303133;
  font-weight: 500;
}

.speed-text {
  color: #909399;
}

.upload-actions {
  display: flex;
  gap: 12px;
  justify-content: center;
  flex-wrap: wrap;
}

.incomplete-tasks {
  margin-top: 40px;
  padding: 20px;
  background-color: #f8f9fa;
  border-radius: 8px;
}

.incomplete-tasks h3 {
  margin: 0 0 20px 0;
  color: #303133;
  font-size: 18px;
}

.task-list {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.task-item {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 16px;
  background-color: white;
  border-radius: 8px;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
}

.task-info {
  display: flex;
  align-items: center;
  gap: 12px;
  flex: 1;
  min-width: 0;
}

.task-icon {
  color: #409eff;
  flex-shrink: 0;
}

.task-details {
  min-width: 0;
}

.task-name {
  font-size: 14px;
  font-weight: 500;
  color: #303133;
  margin: 0 0 4px 0;
  word-break: break-all;
}

.task-size {
  font-size: 12px;
  color: #909399;
  margin: 0;
}

.task-progress {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 120px;
}

.task-progress-text {
  font-size: 12px;
  color: #303133;
  min-width: 30px;
}

.task-actions {
  display: flex;
  gap: 8px;
  flex-shrink: 0;
}

@media (max-width: 768px) {
  .chunk-upload-container {
    padding: 16px;
  }
  
  .upload-area {
    padding: 30px 16px;
  }
  
  .task-item {
    flex-direction: column;
    align-items: stretch;
    gap: 12px;
  }
  
  .task-progress {
    min-width: auto;
  }
  
  .upload-actions {
    justify-content: center;
  }
}
</style>