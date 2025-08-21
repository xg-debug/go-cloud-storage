<template>
  <div class="chunk-upload">
    <el-upload
      ref="uploadRef"
      :auto-upload="false"
      :show-file-list="false"
      :on-change="handleFileChange"
      :accept="acceptTypes"
      :multiple="false"
      drag
    >
      <el-icon class="el-icon--upload"><upload-filled /></el-icon>
      <div class="el-upload__text">
        将文件拖到此处，或<em>点击上传</em>
      </div>
      <template #tip>
        <div class="el-upload__tip">
          支持大文件上传，自动分片处理
        </div>
      </template>
    </el-upload>

    <!-- 上传进度 -->
    <div v-if="uploadingFiles.length > 0" class="upload-progress">
      <div v-for="file in uploadingFiles" :key="file.uid" class="file-progress">
        <div class="file-info">
          <el-icon><document /></el-icon>
          <span class="file-name">{{ file.name }}</span>
          <span class="file-size">{{ formatFileSize(file.size) }}</span>
        </div>
        
        <div class="progress-container">
          <el-progress 
            :percentage="file.progress" 
            :status="file.status === 'error' ? 'exception' : (file.status === 'success' ? 'success' : '')"
            :stroke-width="6"
          />
          <div class="progress-info">
            <span v-if="file.status === 'uploading'">
              {{ file.uploadedChunks }}/{{ file.totalChunks }} 分片
            </span>
            <span v-else-if="file.status === 'merging'">合并中...</span>
            <span v-else-if="file.status === 'success'">上传完成</span>
            <span v-else-if="file.status === 'error'">上传失败</span>
          </div>
        </div>

        <div class="file-actions">
          <el-button 
            v-if="file.status === 'paused' || file.status === 'error'" 
            type="primary" 
            size="small" 
            @click="resumeUpload(file)"
          >
            继续
          </el-button>
          <el-button 
            v-if="file.status === 'uploading'" 
            type="warning" 
            size="small" 
            @click="pauseUpload(file)"
          >
            暂停
          </el-button>
          <el-button 
            type="danger" 
            size="small" 
            @click="cancelUpload(file)"
          >
            取消
          </el-button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed } from 'vue'
import { ElMessage } from 'element-plus'
import { UploadFilled, Document } from '@element-plus/icons-vue'
import { initUpload, uploadChunk, mergeChunks, checkFileExists, getUploadedChunks } from '@/api/chunk'
import SparkMD5 from 'spark-md5'

const props = defineProps({
  folderId: {
    type: [String, Number],
    default: ''
  },
  acceptTypes: {
    type: String,
    default: ''
  },
  chunkSize: {
    type: Number,
    default: 10 * 1024 * 1024 // 10MB
  },
  maxFileSize: {
    type: Number,
    default: 1024 * 1024 * 1024 // 1GB
  }
})

const emit = defineEmits(['upload-success', 'upload-error'])

const uploadRef = ref()
const uploadingFiles = ref([])

// 文件大小格式化
const formatFileSize = (bytes) => {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
}

// 计算文件MD5
const calculateFileMD5 = (file) => {
  return new Promise((resolve, reject) => {
    const spark = new SparkMD5.ArrayBuffer()
    const fileReader = new FileReader()
    const chunkSize = 10485760 // 10MB
    const chunks = Math.ceil(file.size / chunkSize)
    let currentChunk = 0

    fileReader.onload = (e) => {
      spark.append(e.target.result)
      currentChunk++

      if (currentChunk < chunks) {
        loadNext()
      } else {
        resolve(spark.end())
      }
    }

    fileReader.onerror = () => {
      reject(new Error('文件读取失败'))
    }

    const loadNext = () => {
      const start = currentChunk * chunkSize
      const end = Math.min(start + chunkSize, file.size)
      fileReader.readAsArrayBuffer(file.slice(start, end))
    }

    loadNext()
  })
}

// 计算分片MD5
const calculateChunkMD5 = (chunk) => {
  return new Promise((resolve, reject) => {
    const spark = new SparkMD5.ArrayBuffer()
    const fileReader = new FileReader()

    fileReader.onload = (e) => {
      spark.append(e.target.result)
      resolve(spark.end())
    }

    fileReader.onerror = () => {
      reject(new Error('分片读取失败'))
    }

    fileReader.readAsArrayBuffer(chunk)
  })
}

// 处理文件选择
const handleFileChange = async (file) => {
  if (file.size > props.maxFileSize) {
    ElMessage.error(`文件大小不能超过 ${formatFileSize(props.maxFileSize)}`)
    return
  }

  // 添加到上传队列
  const uploadFile = {
    uid: file.uid,
    name: file.name,
    size: file.size,
    file: file.raw,
    progress: 0,
    status: 'preparing', // preparing, uploading, paused, merging, success, error
    uploadedChunks: 0,
    totalChunks: 0,
    fileId: '',
    fileMD5: '',
    chunks: []
  }

  uploadingFiles.value.push(uploadFile)

  try {
    // 计算文件MD5
    uploadFile.status = 'calculating'
    uploadFile.fileMD5 = await calculateFileMD5(file.raw)
    uploadFile.fileId = uploadFile.fileMD5 // 使用MD5作为fileId

    // 检查文件是否已存在（秒传）
    console.log('开始检查文件是否存在，MD5:', uploadFile.fileMD5)
    const checkResult = await checkFileExists({
      fileHash: uploadFile.fileMD5,
      fileName: uploadFile.name,
      fileSize: uploadFile.size.toString()
    })
    
    console.log('文件存在性检查结果:', checkResult)

    if (checkResult && checkResult.exists) {
      uploadFile.progress = 100
      uploadFile.status = 'success'
      ElMessage.success('文件已存在，秒传成功！')
      emit('upload-success', checkResult.file)
      return
    }

    // 开始分片上传
    await startChunkUpload(uploadFile)
  } catch (error) {
    uploadFile.status = 'error'
    ElMessage.error('上传失败: ' + error.message)
    emit('upload-error', error)
  }
}

// 开始分片上传
const startChunkUpload = async (uploadFile) => {
  const totalChunks = Math.ceil(uploadFile.size / props.chunkSize)
  uploadFile.totalChunks = totalChunks

  // 初始化分片上传
  try {
    const initResult = await initUpload({
      fileName: uploadFile.name,
      fileId: uploadFile.fileId
    })
    
    uploadFile.uploadId = initResult.uploadId
    uploadFile.objectKey = initResult.objectKey
  } catch (error) {
    throw new Error('初始化分片上传失败: ' + error.message)
  }

  // 检查已上传的分片
  let uploadedChunkIndexes = []
  try {
    const result = await getUploadedChunks(uploadFile.fileId)
    uploadedChunkIndexes = result.uploadedChunks || []
    uploadFile.uploadedChunks = uploadedChunkIndexes.length
  } catch (error) {
    // 如果获取失败，说明还没有上传过
    uploadFile.uploadedChunks = 0
  }

  // 创建分片
  for (let i = 0; i < totalChunks; i++) {
    const start = i * props.chunkSize
    const end = Math.min(start + props.chunkSize, uploadFile.size)
    const chunk = uploadFile.file.slice(start, end)
    
    uploadFile.chunks.push({
      index: i,
      chunk: chunk,
      uploaded: uploadedChunkIndexes.includes(i)
    })
  }

  uploadFile.status = 'uploading'
  await uploadChunks(uploadFile)
}

// 上传分片
const uploadChunks = async (uploadFile) => {
  const concurrency = 3 // 并发上传数量
  const uploadQueue = uploadFile.chunks.filter(chunk => !chunk.uploaded)
  
  const uploadPromises = []
  let index = 0

  const uploadNext = async () => {
    if (index >= uploadQueue.length || uploadFile.status === 'paused') {
      return
    }

    const chunkInfo = uploadQueue[index++]
    
    try {
      // 计算分片MD5
      const chunkHash = await calculateChunkMD5(chunkInfo.chunk)
      
      const formData = new FormData()
      formData.append('fileId', uploadFile.fileId)
      formData.append('chunkIndex', chunkInfo.index)
      formData.append('chunkHash', chunkHash)
      formData.append('uploadId', uploadFile.uploadId)
      formData.append('objectKey', uploadFile.objectKey)
      formData.append('chunk', chunkInfo.chunk)

      await uploadChunk(formData, (progressEvent) => {
        // 可以在这里处理单个分片的上传进度
      })

      chunkInfo.uploaded = true
      uploadFile.uploadedChunks++
      uploadFile.progress = Math.floor((uploadFile.uploadedChunks / uploadFile.totalChunks) * 90) // 90%用于上传，10%用于合并

      // 继续上传下一个分片
      return uploadNext()
    } catch (error) {
      if (uploadFile.status !== 'paused') {
        throw error
      }
    }
  }

  // 启动并发上传
  for (let i = 0; i < Math.min(concurrency, uploadQueue.length); i++) {
    uploadPromises.push(uploadNext())
  }

  await Promise.all(uploadPromises)

  // 所有分片上传完成，开始合并
  if (uploadFile.status === 'uploading') {
    await mergeFileChunks(uploadFile)
  }
}

// 合并分片
const mergeFileChunks = async (uploadFile) => {
  uploadFile.status = 'merging'
  uploadFile.progress = 95

  try {
    const result = await mergeChunks({
      fileId: uploadFile.fileId,
      fileName: uploadFile.name,
      totalChunks: uploadFile.totalChunks,
      fileSize: uploadFile.size,
      fileHash: uploadFile.fileMD5,
      parentId: props.folderId || '',
      objectKey: uploadFile.objectKey
    })

    uploadFile.progress = 100
    uploadFile.status = 'success'
    ElMessage.success('文件上传成功！')
    emit('upload-success', result.file)
  } catch (error) {
    uploadFile.status = 'error'
    throw error
  }
}

// 暂停上传
const pauseUpload = (uploadFile) => {
  uploadFile.status = 'paused'
}

// 继续上传
const resumeUpload = async (uploadFile) => {
  uploadFile.status = 'uploading'
  await uploadChunks(uploadFile)
}

// 取消上传
const cancelUpload = (uploadFile) => {
  const index = uploadingFiles.value.findIndex(f => f.uid === uploadFile.uid)
  if (index > -1) {
    uploadingFiles.value.splice(index, 1)
  }
}
</script>

<style scoped>
.chunk-upload {
  width: 100%;
}

.upload-progress {
  margin-top: 20px;
}

.file-progress {
  border: 1px solid #e4e7ed;
  border-radius: 6px;
  padding: 16px;
  margin-bottom: 12px;
  background: #fafafa;
}

.file-info {
  display: flex;
  align-items: center;
  margin-bottom: 12px;
}

.file-info .el-icon {
  margin-right: 8px;
  color: #606266;
}

.file-name {
  flex: 1;
  font-weight: 500;
  margin-right: 12px;
}

.file-size {
  color: #909399;
  font-size: 12px;
}

.progress-container {
  margin-bottom: 12px;
}

.progress-info {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-top: 4px;
  font-size: 12px;
  color: #909399;
}

.file-actions {
  display: flex;
  gap: 8px;
}
</style>