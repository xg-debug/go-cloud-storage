<template>
  <el-upload class="upload-demo" drag multiple :action="uploadUrl" :headers="headers" :data="uploadData"
    :on-success="handleSuccess" :on-error="handleError" :before-upload="beforeUpload" :file-list="fileList"
    :auto-upload="false" :on-change="handleChange">
    <el-icon class="el-icon--upload"><i-ep-UploadFilled /></el-icon>
    <div class="el-upload__text">
      拖拽文件到此处或<em>点击上传</em>
    </div>
    <template #tip>
      <div class="el-upload__tip">
        支持单个或批量上传，单文件不超过10GB
      </div>
    </template>
  </el-upload>

  <div class="upload-actions" v-if="fileList.length > 0">
    <el-button type="primary" @click="submitUpload">
      开始上传
    </el-button>
    <el-button @click="clearFiles">
      清空列表
    </el-button>
    <el-progress v-if="uploading" :percentage="totalProgress" :stroke-width="6" style="margin-top: 10px; width: 100%" />
  </div>
</template>
  
<script setup>
import { ref, computed } from 'vue'
import { useFileStore } from '@/stores/files'
import { getToken } from '@/utils/auth'

const fileStore = useFileStore()
const uploadRef = ref()
const fileList = ref([])
const uploadProgress = ref({})
const uploadUrl = import.meta.env.VITE_API_BASE_URL + '/upload'

const headers = computed(() => ({
  Authorization: `Bearer ${getToken()}`
}))

const uploadData = computed(() => ({
  path: fileStore.currentPath
}))

const totalProgress = computed(() => {
  if (fileList.value.length === 0) return 0
  const total = Object.values(uploadProgress.value).reduce((sum, p) => sum + p, 0)
  return Math.round(total / fileList.value.length)
})

function beforeUpload(file) {
  const isLt10G = file.size / 1024 / 1024 / 1024 < 10
  if (!isLt10G) {
    ElMessage.error('文件大小不能超过10GB!')
    return false
  }
  return true
}

function handleChange(file, files) {
  fileList.value = files
}

function submitUpload() {
  uploadRef.value?.submit()
}

function clearFiles() {
  fileList.value = []
}

function handleSuccess(response, file) {
  ElMessage.success(`${file.name} 上传成功`)
  fileStore.fetchFiles()
}

function handleError(err, file) {
  ElMessage.error(`${file.name} 上传失败: ${err.message}`)
}
</script>