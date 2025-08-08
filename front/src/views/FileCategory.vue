<template>
    <div class="file-category-container">
        <!-- 标题与操作栏 -->
        <div class="category-toolbar">
            <h2>文件分类</h2>
            <!-- <el-button type="primary" @click="dialogVisible = true" class="ml-auto">
          <el-icon><Plus /></el-icon>
          新建分类
        </el-button> -->
        </div>

        <!-- 文件类型分类展示 -->
        <el-row :gutter="24" class="type-grid">
            <el-col v-for="type in fileTypes" :key="type.name" :xs="12" :sm="8" :md="6" :lg="4">
                <category-card :icon="type.icon" :name="type.name" :count="type.count"
                    :active="type.name === activeType" @click="handleTypeClick(type)" class="category-card" />
            </el-col>
        </el-row>

        <!-- 新建分类对话框 -->
        <el-dialog v-model="dialogVisible" title="新建分类" width="420px" :before-close="handleDialogClose"
            destroy-on-close>
            <category-form @submit="handleSubmit" @cancel="dialogVisible = false" />
        </el-dialog>
    </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { Plus, Picture, VideoCamera, Document, Headset, Box, Folder } from '@element-plus/icons-vue'
import CategoryCard from '@/components/CategoryCard.vue'
import CategoryForm from '@/components/CategoryForm.vue'

const activeType = ref(null)
const dialogVisible = ref(false)

const fileTypes = ref([])

const loadData = () => {
    // 模拟接口数据，真实情况可改为请求接口
    fileTypes.value = [
        { name: '图片', icon: Picture, count: 248, ext: ['jpg', 'png', 'gif'] },
        { name: '文档', icon: Document, count: 156, ext: ['pdf', 'doc', 'ppt'] },
        { name: '视频', icon: VideoCamera, count: 87, ext: ['mp4', 'mov'] },
        { name: '音频', icon: Headset, count: 42, ext: ['mp3', 'wav'] },
        { name: '压缩包', icon: Box, count: 35, ext: ['zip', 'rar'] },
        { name: '种子', icon: Folder, count: 15, ext: ['torrent'] },
        { name: '其他', icon: Folder, count: 63, ext: [] }
    ]
    // 默认激活第一个类型
    activeType.value = fileTypes.value[0].name
}

onMounted(() => {
    loadData()
})

const handleTypeClick = (type) => {
    activeType.value = type.name
    console.log('选择文件类型分类:', type.name)
    // TODO: 触发文件列表筛选逻辑，根据 type.ext 或 type.name 过滤文件
}

const handleDialogClose = (done) => {
    dialogVisible.value = false
    done()
}

const handleSubmit = (formData) => {
    console.log('新建分类数据:', formData)
    dialogVisible.value = false
    // TODO: 调用接口创建分类，刷新 fileTypes 数据
}
</script>

<style scoped>
.file-category-container {
    padding: 24px 20px;
    height: calc(100vh - 60px);
    overflow-y: auto;
    background-color: #f9fafc;
    scrollbar-width: none;
    /* display: flex; */
    flex-direction: column;
}

.file-category-container::-webkit-scrollbar {
    display: none;
}

.category-toolbar {
    display: flex;
    align-items: center;
    margin-bottom: 24px;
    border-bottom: 1px solid #e4e7ed;
    padding-bottom: 12px;
    user-select: none;
}

.category-toolbar h2 {
    font-weight: 700;
    font-size: 24px;
    color: #303133;
    margin: 0;
}

.ml-auto {
    margin-left: auto !important;
}

.type-grid {
    flex-grow: 1;
}

.category-card {
    cursor: pointer;
    transition: transform 0.15s ease, box-shadow 0.2s ease;
    border-radius: 12px;
    box-shadow: 0 2px 10px rgb(0 0 0 / 0.06);
    background: #fff;
    padding: 20px 0;
    text-align: center;
    user-select: none;
}

.category-card:hover {
    transform: translateY(-4px);
    box-shadow: 0 8px 20px rgb(0 0 0 / 0.12);
}

.category-card.active {
    border: 2px solid var(--el-color-primary);
    box-shadow: 0 10px 25px var(--el-color-primary-light);
}

.category-card>>>.icon-wrapper {
    font-size: 40px;
    color: var(--el-color-primary);
    margin-bottom: 12px;
}

.category-card>>>.name {
    font-weight: 600;
    font-size: 16px;
    color: #606266;
    margin-bottom: 6px;
}

.category-card>>>.count {
    font-size: 14px;
    color: #909399;
}
</style>