<template>
    <div class="search-page">
        <h2>搜索结果: "{{ keyword }}"</h2>
        <el-divider/>

        <el-empty v-if="loading === false && results.length === 0" description="无搜索结果"/>

        <el-skeleton v-if="loading" animated>
            <template #template>
                <el-skeleton-item variant="text" style="width: 100%; height: 30px"/>
                <el-skeleton-item variant="text" style="width: 100%; height: 30px; margin-top: 10px"/>
                <el-skeleton-item variant="text" style="width: 100%; height: 30px; margin-top: 10px"/>
            </template>
        </el-skeleton>

        <el-table v-else :data="results" style="width: 100%" v-if="results.length > 0">
            <el-table-column prop="name" label="文件名" min-width="200"/>
            <el-table-column prop="type" label="类型" width="100"/>
            <el-table-column prop="modified" label="修改日期" width="180"/>
            <el-table-column prop="size" label="大小" width="120"/>
            <el-table-column label="操作" width="150" align="center">
                <template #default="{ row }">
                    <el-button size="small" type="text" @click="handlePreview(row)">预览</el-button>
                    <el-button size="small" type="text" @click="handleDownload(row)">下载</el-button>
                </template>
            </el-table-column>
        </el-table>
    </div>
</template>

<script setup>
import {onMounted, ref, watch} from 'vue'
import {useRoute} from 'vue-router'
import {ElMessage} from 'element-plus'
import {downloadFile, previewFile, searchFiles} from '@/api/file'

const route = useRoute()
const keyword = ref(route.query.q || '')
const results = ref([])
const loading = ref(false)

const loadResults = async () => {
    if (!keyword.value.trim()) {
        results.value = []
        return
    }
    loading.value = true
    try {
        const res = await searchFiles(keyword.value.trim())
        results.value = res.data || []
    } catch (err) {
        ElMessage.error('搜索失败，请重试')
    } finally {
        loading.value = false
    }
}

onMounted(loadResults)
watch(() => route.query.q, (newQ) => {
    keyword.value = newQ || ''
    loadResults()
})

const handlePreview = (file) => {
    previewFile(file.id)
    // 这里你可以做打开预览弹窗或跳转预览页
}

const handleDownload = (file) => {
    downloadFile(file.id)
    // 这里直接调用下载接口即可
}
</script>

<style scoped>
.search-page {
    padding: 20px;
}
</style>
  