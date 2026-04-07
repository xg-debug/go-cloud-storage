<template>
  <div class="share-link-page">
    <div class="share-card">
      <div v-if="loading" class="state-container">
        <el-icon class="is-loading" :size="40" color="#409eff"><Loading /></el-icon>
        <p>正在加载分享信息...</p>
      </div>

      <div v-else-if="error" class="state-container">
        <el-icon :size="48" color="#f56c6c"><CircleCloseFilled /></el-icon>
        <p class="error-text">{{ error }}</p>
      </div>

      <div v-else-if="needCode" class="state-container">
        <div class="icon-wrapper">
          <el-icon :size="48" color="#e6a23c"><Lock /></el-icon>
        </div>
        <h3>请输入提取码</h3>
        <p class="hint-text">此分享受提取码保护</p>
        <div class="code-input-wrapper">
          <el-input
            v-model="extractCode"
            placeholder="请输入提取码"
            maxlength="20"
            @keyup.enter="verifyCode"
            size="large"
          >
            <template #append>
              <el-button @click="verifyCode" :loading="verifying">提取文件</el-button>
            </template>
          </el-input>
        </div>
      </div>

      <div v-else-if="shareInfo" class="file-info-container">
        <div class="file-header">
          <div class="file-icon">
            <el-icon :size="56" :color="getFileIconColor(shareInfo.fileType)">
              <component :is="getFileIcon(shareInfo.fileType)" />
            </el-icon>
          </div>
          <div class="file-details">
            <h2 class="file-name" :title="shareInfo.fileName">{{ shareInfo.fileName }}</h2>
            <div class="file-meta">
              <span>{{ formatSize(shareInfo.fileSize) }}</span>
              <span class="dot">·</span>
              <span v-if="shareInfo.canPreview">支持在线预览</span>
              <span v-else>暂不支持在线预览</span>
            </div>
          </div>
        </div>

        <div v-if="shareInfo.canPreview" class="preview-box">
          <img v-if="shareInfo.previewType === 'image'" :src="shareInfo.fileUrl" alt="preview" class="preview-image" />

          <video
            v-else-if="shareInfo.previewType === 'video'"
            :src="shareInfo.fileUrl"
            controls
            class="preview-media"
          />

          <audio
            v-else-if="shareInfo.previewType === 'audio'"
            :src="shareInfo.fileUrl"
            controls
            class="preview-audio"
          />

          <iframe
            v-else-if="shareInfo.previewType === 'pdf'"
            :src="shareInfo.fileUrl"
            class="preview-frame"
            frameborder="0"
          />

          <iframe
            v-else-if="shareInfo.previewType === 'office'"
            :src="shareInfo.officePreviewUrl"
            class="preview-frame"
            frameborder="0"
          />

          <iframe
            v-else-if="shareInfo.previewType === 'text'"
            :src="shareInfo.fileUrl"
            class="preview-frame"
            frameborder="0"
          />
        </div>

        <div class="actions">
          <el-button type="primary" size="large" @click="handleDownload" :loading="downloading" class="download-btn">
            <el-icon class="el-icon--left"><Download /></el-icon>
            下载文件
          </el-button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { accessShare, downloadSharedFile } from '@/api/share'
import { ElMessage } from 'element-plus'
import {
  Loading,
  CircleCloseFilled,
  Lock,
  Download,
  Document,
  Picture,
  VideoCamera,
  Headset,
  Files
} from '@element-plus/icons-vue'

const route = useRoute()
const token = route.params.token

const loading = ref(true)
const error = ref('')
const needCode = ref(false)
const shareInfo = ref(null)
const extractCode = ref('')
const verifying = ref(false)
const downloading = ref(false)

onMounted(() => {
  if (!token) {
    error.value = '无效的分享链接'
    loading.value = false
    return
  }
  loadShareInfo()
})

const loadShareInfo = async () => {
  loading.value = true
  error.value = ''

  try {
    const data = await accessShare(token, '')
    shareInfo.value = data
    needCode.value = !!data.needCode
  } catch (err) {
    const msg = err.message || '获取分享信息失败'
    if (msg.includes('过期')) {
      error.value = '分享链接已过期'
    } else if (msg.includes('不存在') || msg.includes('失效')) {
      error.value = '分享链接不存在或已失效'
    } else {
      error.value = msg
    }
  } finally {
    loading.value = false
  }
}

const verifyCode = async () => {
  if (!extractCode.value) {
    ElMessage.warning('请输入提取码')
    return
  }

  verifying.value = true
  try {
    const data = await accessShare(token, extractCode.value)
    shareInfo.value = data
    needCode.value = false
  } catch (err) {
    ElMessage.error(err.message || '提取码错误')
  } finally {
    verifying.value = false
  }
}

const handleDownload = async () => {
  downloading.value = true
  try {
    const url = await downloadSharedFile(token, extractCode.value)
    window.open(url, '_blank')
  } catch (err) {
    ElMessage.error('下载失败：' + (err.message || '未知错误'))
  } finally {
    downloading.value = false
  }
}

const formatSize = (bytes) => {
  if (!bytes) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
}

const getFileIcon = (type) => {
  const iconMap = {
    image: Picture,
    video: VideoCamera,
    audio: Headset,
    document: Document
  }
  return iconMap[type] || Files
}

const getFileIconColor = (type) => {
  const colorMap = {
    image: '#f59e0b',
    video: '#ef4444',
    audio: '#8b5cf6',
    document: '#06b6d4'
  }
  return colorMap[type] || '#6b7280'
}
</script>

<style scoped>
.share-link-page {
  min-height: 100vh;
  display: flex;
  justify-content: center;
  align-items: center;
  background-color: #f0f2f5;
  padding: 20px;
}

.share-card {
  width: 100%;
  max-width: 920px;
  background: white;
  border-radius: 12px;
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.08);
  padding: 32px;
}

.state-container {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 16px;
}

.error-text {
  color: #f56c6c;
  margin-bottom: 16px;
}

.icon-wrapper {
  width: 80px;
  height: 80px;
  background: #fdf6ec;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  margin-bottom: 8px;
}

.hint-text {
  color: #909399;
  margin-bottom: 16px;
}

.code-input-wrapper {
  width: 100%;
  max-width: 320px;
}

.file-info-container {
  text-align: center;
}

.file-header {
  display: flex;
  align-items: center;
  gap: 16px;
  margin-bottom: 20px;
}

.file-icon {
  flex-shrink: 0;
}

.file-details {
  text-align: left;
}

.file-name {
  font-size: 20px;
  color: #303133;
  margin: 0 0 8px 0;
  word-break: break-all;
}

.file-meta {
  color: #909399;
  font-size: 14px;
}

.dot {
  margin: 0 8px;
}

.preview-box {
  border: 1px solid #ebeef5;
  border-radius: 10px;
  overflow: hidden;
  margin-bottom: 20px;
  background: #fafafa;
}

.preview-image {
  width: 100%;
  max-height: 520px;
  object-fit: contain;
  display: block;
}

.preview-media {
  width: 100%;
  max-height: 520px;
  background: #000;
}

.preview-audio {
  width: 100%;
  margin: 20px;
}

.preview-frame {
  width: 100%;
  height: 520px;
}

.actions {
  display: flex;
  justify-content: center;
}

.download-btn {
  width: 100%;
  max-width: 240px;
}

@media (max-width: 768px) {
  .share-card {
    padding: 20px;
  }

  .file-header {
    flex-direction: column;
    text-align: center;
  }

  .file-details {
    text-align: center;
  }

  .preview-frame,
  .preview-media,
  .preview-image {
    max-height: 380px;
    height: 380px;
  }
}
</style>
