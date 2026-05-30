<template>
  <el-dialog
    v-model="visible"
    title="创建分享"
    width="500px"
    :close-on-click-modal="false"
    @close="handleClose"
  >
    <div class="create-share-form">
      <!-- 文件信息 -->
      <div class="file-info">
        <div class="file-icon">
          <el-icon :size="32" :color="getFileIconColor(fileInfo.fileType)">
            <component :is="getFileIcon(fileInfo.fileType)" />
          </el-icon>
        </div>
        <div class="file-details">
          <div class="file-name">{{ fileInfo.name }}</div>
          <div class="file-size">{{ formatSize(fileInfo.size) }}</div>
        </div>
      </div>

      <!-- 分享设置 -->
      <el-form :model="shareForm" :rules="rules" ref="formRef" label-width="100px">
        <el-form-item label="提取码" prop="extractionCode">
          <el-input
            v-model="shareForm.extractionCode"
            placeholder="留空则无需提取码"
            maxlength="20"
            clearable
          />
          <div class="form-tip">设置提取码可以增加分享的安全性</div>
        </el-form-item>

        <el-form-item label="有效期" prop="expireDays">
          <el-select v-model="shareForm.expireDays" placeholder="选择有效期">
            <el-option label="永久有效" :value="0" />
            <el-option label="1天" :value="1" />
            <el-option label="7天" :value="7" />
            <el-option label="30天" :value="30" />
          </el-select>
        </el-form-item>
      </el-form>
    </div>

    <template #footer>
      <el-button @click="handleClose">取消</el-button>
      <el-button type="primary" @click="handleCreateShare" :loading="creating">
        创建分享
      </el-button>
    </template>
  </el-dialog>
</template>

<script setup>
import { ref, reactive, watch } from 'vue'
import { ElMessage } from 'element-plus'
import {
  Picture,
  VideoCamera,
  Headset,
  Document,
  Files
} from '@element-plus/icons-vue'
import { createShare } from '@/api/share'

const props = defineProps({
  modelValue: {
    type: Boolean,
    default: false
  },
  fileInfo: {
    type: Object,
    default: () => ({})
  }
})

const emit = defineEmits(['update:modelValue', 'success'])

const visible = ref(false)
const creating = ref(false)
const formRef = ref()

const shareForm = reactive({
  extractionCode: '',
  expireDays: 7
})

const rules = {
  extractionCode: [
    { min: 0, max: 20, message: '提取码长度不能超过20个字符', trigger: 'blur' }
  ]
}

watch(() => props.modelValue, (val) => {
  visible.value = val
  if (val) {
    // 重置表单
    shareForm.extractionCode = ''
    shareForm.expireDays = 7
  }
})

watch(visible, (val) => {
  emit('update:modelValue', val)
})

// 获取文件图标
const getFileIcon = (type) => {
  const iconMap = {
    image: Picture,
    video: VideoCamera,
    audio: Headset,
    document: Document
  }
  return iconMap[type] || Files
}

// 获取文件图标颜色
const getFileIconColor = (type) => {
  const colorMap = {
    image: '#f59e0b',
    video: '#ef4444',
    audio: '#8b5cf6',
    document: '#06b6d4'
  }
  return colorMap[type] || '#6b7280'
}

// 格式化文件大小
const formatSize = (bytes) => {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
}

// 创建分享
const handleCreateShare = async () => {
  if (!formRef.value) return
  
  try {
    await formRef.value.validate()
    
    creating.value = true
    
    const data = await createShare({
      file_id: props.fileInfo.id,
      extraction_code: shareForm.extractionCode,
      expire_days: shareForm.expireDays
    })
    
    if (data) {
      ElMessage.success('分享成功')
      emit('success', data)// 通知其上层组件去处理后续的逻辑
      handleClose()
    } else {
      ElMessage.error('创建分享失败')
    }
  } catch (error) {
    console.error('创建分享失败:', error)
    ElMessage.error('创建分享失败')
  } finally {
    creating.value = false
  }
}

// 关闭对话框
const handleClose = () => {
  visible.value = false
}
</script>

<style scoped>
.create-share-form {
  padding: 20px 0;
}

.file-info {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 16px;
  background: #f9fafb;
  border-radius: 8px;
  margin-bottom: 24px;
}

.file-details {
  flex: 1;
}

.file-name {
  font-size: 16px;
  font-weight: 500;
  color: #1f2937;
  margin-bottom: 4px;
}

.file-size {
  font-size: 14px;
  color: #6b7280;
}

.form-tip {
  font-size: 12px;
  color: #6b7280;
  margin-top: 4px;
}
</style>