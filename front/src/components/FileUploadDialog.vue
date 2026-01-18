<template>
    <el-dialog
            v-model="dialogVisible"
            title="文件上传"
            width="500px"
            :close-on-click-modal="false"
            :before-close="beforeClose"
    >
        <!-- 选择 / 拖拽 -->
        <div
                v-if="!pendingFile"
                class="drop-zone"
                :class="{ dragging: isDragging }"
                @dragover.prevent="isDragging = true"
                @dragleave.prevent="isDragging = false"
                @drop.prevent="onDrop"
        >
            <el-icon :size="48"><Upload/></el-icon>
            <p>将文件拖拽到此处 或</p>
            <el-button link @click="triggerSelect">选择本地文件</el-button>
            <input
                    type="file"
                    ref="uploadInputRef"
                    style="display: none"
                    @change="onSelectFile"
            />
        </div>

        <!-- 上传进度 -->
        <div v-else class="upload-progress-info">
            <h4 style="margin-bottom: 15px;">
                正在上传：{{ pendingFile.name }}
            </h4>

            <el-progress :percentage="uploadProgress" />
        </div>

        <template #footer>
            <el-button @click="handleClose">关闭</el-button>
        </template>
    </el-dialog>
</template>


<script setup>
import {useStore} from 'vuex'
import {onMounted, ref, computed, watch} from 'vue'
import {uploadFile, chunkUploadInit, chunkUploadPart, chunkUploadMerge, chunkUploadCancel} from '@/api/file'
import {ElMessage} from 'element-plus'
import {Upload} from "@element-plus/icons-vue";

const store = useStore()

const props = defineProps({
    modelValue: {
        type: Boolean,
        required: true
    },
    parentId: {
        type: String,
        required: true,
        default: '' // 默认空字符串
    }
})

const currentParentId = ref(props.parentId) // 默认上传到根目录

const emit = defineEmits(['update:modelValue', 'success'])

const dialogVisible = computed({
    get: () => props.modelValue,
    set: val => emit('update:modelValue', val)
})


// 上传相关状态
const uploadInputRef = ref(null)
const pendingFile = ref(null)            // 当前上传的文件
const uploadProgress = ref(0)            // 总进度（大文件/小文件统一）
const uploading = ref(false)             // 上传中
const isDragging = ref(false)

const CHUNK_SIZE = 10 * 1024 * 1024      // 10MB
const CHUNK_THRESHOLD = 10 * 1024 * 1024 // 判定大文件


const beforeClose = (done) => {
    if (uploading.value) {
        ElMessage.warning('文件正在上传，请稍候')
        return
    }
    done()
}

const handleClose = () => {
    dialogVisible.value = false
}

/* 打开本地选择文件 */
const triggerSelect = () => {
    uploadInputRef.value.click()
}

/* 选择文件 */
const onSelectFile = (e) => {
    const file = e.target.files[0]
    e.target.value = ''
    if (file) prepareUpload(file)
}

/* 拖拽上传 */
const onDrop = (e) => {
    const file = e.dataTransfer.files[0]
    isDragging.value = false
    if (file) prepareUpload(file)
}

/* 准备上传（区分大小文件） */
const prepareUpload = (file) => {
    pendingFile.value = file
    uploadProgress.value = 0

    if (file.size >= CHUNK_THRESHOLD) {
        uploadLargeFile(file)
    } else {
        uploadSmallFile(file)
    }
}

const uploadSmallFile = async (file) => {
    uploading.value = true

    const fileHash = await calcSHA256(file)


    const form = new FormData()
    form.append('file', file)
    form.append('parentId', currentParentId.value)
    form.append("fileHash", fileHash)

    try {
        await uploadFile(form, (e) => {
            uploadProgress.value = Math.round((e.loaded * 100) / e.total)
        })

        finishUpload(true)
    } catch (err) {
        finishUpload(false)
    }
}

const uploadLargeFile = async (file) => {
    uploadProgress.value = 0

    try {
        const fileHash = await calcSHA256(file)

        // 初始化任务
        const initRes = await chunkUploadInit({
            fileHash,
            parentId: currentParentId.value,
            fileName: file.name,
            fileSize: file.size,
        })

        // 秒传成功
        if (initRes.finished) {
            uploadProgress.value = 100
            finishUpload(true)
            return
        }

        const uploaded = new Set(initRes.uploadedChunks)
        const totalChunks = Math.ceil(file.size / CHUNK_SIZE)
        let finished = uploaded.size

        // 上传每个分片
        for (let index = 0; index < totalChunks; index++) {
            if (uploaded.has(index)) {
                updateProgress(uploaded.size, totalChunks)
                continue
            }

            const start = index * CHUNK_SIZE
            const end = Math.min(file.size, start + CHUNK_SIZE)
            const chunk = file.slice(start, end)

            const form = new FormData()
            form.append('fileHash', fileHash)
            form.append('chunkIndex', index)
            form.append('chunk', chunk)

            await chunkUploadPart(form, () => {})

            finished++
            updateProgress(finished, totalChunks)
        }

        // 合并分片
        uploadProgress.value = 98
        await chunkUploadMerge({
            fileHash,
            fileName: file.name,
            fileSize: file.size,
            parentId: currentParentId.value
        })

        uploadProgress.value = 100
        finishUpload(true)

    } catch (err) {
        console.error(err)
        finishUpload(false)
    }
}
/* 更新进度条（95% 用于上传部分） */
const updateProgress = (finished, total) => {
    uploadProgress.value = Math.round((finished / total) * 95)
}
/* -----------------------------------------------------
   计算 SHA-256（浏览器原生）
------------------------------------------------------ */
const calcSHA256 = async (file) => {
    const buffer = await file.arrayBuffer()
    const hashBuffer = await crypto.subtle.digest('SHA-256', buffer)
    return Array.from(new Uint8Array(hashBuffer))
        .map(b => b.toString(16).padStart(2, '0'))
        .join('')
}

/* ======================
   上传结果统一出口
====================== */
const finishUpload = (success) => {
    uploading.value = false

    if (success) {
        ElMessage.success('上传成功')
        emit('success')
        dialogVisible.value = false
        resetState()
    } else {
        ElMessage.error('上传失败,请重新上传')
        resetState()
    }
}

/* -----------------------------------------------------
   上传对话框关闭时重置
------------------------------------------------------ */
const resetState = () => {
    pendingFile.value = null
    uploadProgress.value = 0
    isDragging.value = false
}
</script>


<style scoped>
.drop-zone {
    border: 2px dashed var(--border-medium);
    padding: 48px 24px;
    text-align: center;
    cursor: pointer;
    transition: all var(--transition-normal);
    border-radius: var(--radius-xl);
    background: linear-gradient(135deg, rgba(248, 250, 252, 0.8) 0%, rgba(241, 245, 249, 0.8) 100%);
}

.drop-zone:hover {
    border-color: var(--primary-color);
    background: linear-gradient(135deg, rgba(238, 242, 255, 0.9) 0%, rgba(250, 245, 255, 0.9) 100%);
}

.drop-zone.dragging {
    border-color: var(--primary-color);
    background: linear-gradient(135deg, rgba(238, 242, 255, 0.95) 0%, rgba(250, 245, 255, 0.95) 100%);
    transform: scale(1.02);
    box-shadow: 0 0 0 4px rgba(99, 102, 241, 0.1);
}

.drop-zone .el-icon {
    color: var(--primary-color);
}

.drop-zone p {
    margin: 16px 0;
    color: var(--text-secondary);
    font-size: 15px;
}
.upload-progress-info {
    text-align: center;
    padding: 20px 0;
}

.upload-progress-info h4 {
    color: var(--text-primary);
    font-weight: 600;
}

.upload-progress-info p {
    color: var(--text-secondary);
    margin-top: 12px;
    font-size: 14px;
}

</style>