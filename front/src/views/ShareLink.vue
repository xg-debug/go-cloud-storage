<template>
  <div class="share-link-page">
    <div class="share-card">
      <!-- Loading State -->
      <div v-if="loading" class="state-container">
        <el-icon class="is-loading" :size="40" color="#409eff"><Loading /></el-icon>
        <p>正在加载分享信息...</p>
      </div>

      <!-- Error State -->
      <div v-else-if="error" class="state-container">
        <el-icon :size="48" color="#f56c6c"><CircleCloseFilled /></el-icon>
        <p class="error-text">{{ error }}</p>
        <el-button type="primary" @click="$router.push('/')">返回首页</el-button>
      </div>

      <!-- Need Code State -->
      <div v-else-if="needCode" class="state-container">
        <div class="icon-wrapper">
          <el-icon :size="48" color="#e6a23c"><Lock /></el-icon>
        </div>
        <h3>请输入提取码</h3>
        <p class="hint-text">此文件受密码保护，请输入提取码继续访问</p>
        <div class="code-input-wrapper">
          <el-input 
            v-model="extractCode" 
            placeholder="请输入4位提取码" 
            maxlength="4"
            @keyup.enter="verifyCode"
            size="large"
          >
            <template #append>
              <el-button @click="verifyCode" :loading="verifying">
                提取文件
              </el-button>
            </template>
          </el-input>
        </div>
      </div>

      <!-- File Info State -->
      <div v-else-if="shareInfo" class="file-info-container">
        <div class="file-header">
          <div class="file-icon">
            <el-icon :size="64" :color="getFileIconColor(shareInfo.fileType)">
              <component :is="getFileIcon(shareInfo.fileType)" />
            </el-icon>
          </div>
          <div class="file-details">
            <h2 class="file-name" :title="shareInfo.fileName">{{ shareInfo.fileName }}</h2>
            <div class="file-meta">
              <span>{{ formatSize(shareInfo.fileSize) }}</span>
            </div>
          </div>
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

// Initialize
onMounted(() => {
  if (!token) {
    error.value = '无效的分享链接'
    loading.value = false
    return
  }
  loadShareInfo()
})

// Load share info (try without code first)
const loadShareInfo = async () => {
  loading.value = true
  error.value = ''
  
  try {
    const res = await accessShare(token, '')
    // Success - show file info
    if (res.code === 200) {
      shareInfo.value = res.data
    } else {
        // Should not happen if interceptor handles it, but just in case
        throw new Error(res.msg || '获取分享信息失败')
    }
  } catch (err) {
    console.error(err)
    if (err.response) {
        // If 403 or specific error message indicates code required
        // Based on backend implementation, it returns 500 or 400 with "提取码错误"
        // Let's assume the API might return a specific code or message
        // Since the backend returns `errors.New("提取码错误")`, it likely results in 500 or 400.
        // We need to check if the error message contains "提取码"
        const msg = err.response.data?.msg || ''
        if (msg.includes('提取码')) {
            needCode.value = true
        } else if (msg.includes('过期')) {
            error.value = '分享链接已过期'
        } else if (msg.includes('不存在')) {
            error.value = '分享链接不存在'
        } else {
            error.value = msg || '获取分享信息失败'
        }
    } else {
        error.value = '网络错误，请稍后重试'
    }
  } finally {
    loading.value = false
  }
}

// Verify code
const verifyCode = async () => {
  if (!extractCode.value) {
    ElMessage.warning('请输入提取码')
    return
  }
  
  verifying.value = true
  try {
    const res = await accessShare(token, extractCode.value)
    if (res.code === 200) {
      shareInfo.value = res.data
      needCode.value = false
    }
  } catch (err) {
    const msg = err.response?.data?.msg || '提取码错误'
    ElMessage.error(msg)
  } finally {
    verifying.value = false
  }
}

// Download file
const handleDownload = async () => {
  downloading.value = true
  try {
    // We already have the download URL from shareInfo, but it might be internal or require auth
    // The backend `DownloadSharedFile` returns a download URL.
    // Actually shareInfo.downloadUrl is already available from AccessShare response.
    // However, if we need to track download count, calling downloadSharedFile API is better if the backend does that.
    // But currently backend `DownloadSharedFile` just calls `AccessShare` then returns url.
    
    // Let's use the API to get the download URL again (maybe signed or temporary)
    const res = await downloadSharedFile(token, extractCode.value)
    if (res.code === 200) {
        window.location.href = res.data
    }
  } catch (err) {
    ElMessage.error('下载失败：' + (err.response?.data?.msg || err.message))
  } finally {
    downloading.value = false
  }
}

// Helpers
const formatSize = (bytes) => {
    if (bytes === 0) return '0 B'
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
  max-width: 480px;
  background: white;
  border-radius: 12px;
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.08);
  padding: 40px;
  text-align: center;
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
  margin-bottom: 16px;
}

.hint-text {
  color: #909399;
  margin-bottom: 24px;
}

.code-input-wrapper {
  width: 100%;
  max-width: 300px;
}

.file-info-container {
  text-align: center;
}

.file-header {
  margin-bottom: 32px;
}

.file-icon {
  margin-bottom: 16px;
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

.download-btn {
  width: 100%;
  max-width: 240px;
}
</style>
