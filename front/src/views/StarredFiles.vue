<template>
    <div class="starred-container">
        <!-- 顶部搜索栏 -->
        <div class="header">
            <h2>⭐ 我的收藏</h2>
        </div>

        <el-divider/>

        <!-- 收藏列表 -->
        <el-table
            :data="starredItems"
            v-loading="loading"
            style="width: 100%"
            empty-text="暂无收藏内容"
            stripe
            border
        >
            <el-table-column label="名称">
                <template #default="{ row }">
                    <div class="file-name-cell">
                        <el-icon :color="row.is_dir ? '#FFB800' : '#3a86ff'">
                            <component :is="row.is_dir ? Folder : Document"/>
                        </el-icon>
                        <span>{{ row.name }}</span>
                    </div>
                </template>
            </el-table-column>

            <el-table-column prop="path" label="所在目录"/>
            <el-table-column prop="size_str" label="大小" width="120" />
            <el-table-column prop="created_at" label="收藏时间" width="180"/>
            <el-table-column label="操作" width="320">
                <template #default="{ row }">
                    <el-button size="small" type="primary" @click="openFile(row)">打开</el-button>
                    <el-button size="small" @click="downloadFile(row)" :disabled="row.is_dir">下载</el-button>
                    <el-button size="small" type="danger" @click="unfavorite(row)">取消收藏</el-button>
                    <el-button size="small" @click="locateFile(row)">定位</el-button>
                </template>
            </el-table-column>
        </el-table>

        <!-- 分页器 -->
        <div class="pagination">
            <el-pagination
                background
                layout="prev, pager, next"
                :total="totalCount"
                :page-size="pageSize"
                v-model:current-page="currentPage"
                @current-change="onPageChange"
                hide-on-single-page
            />
        </div>
    </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { Document, Folder, Search } from '@element-plus/icons-vue'
import { getFavorites } from '@/api/favorite'

// 加载状态
const loading = ref(false)

// 收藏列表
const starredItems = ref([])
const totalCount = ref(0)

// 分页参数
const currentPage = ref(1)
const pageSize = 10

// 获取收藏列表
const fetchFavorites = async () => {
    loading.value = true
    try {
        const res = await getFavorites({ page: currentPage.value, pageSize })
        starredItems.value = res.favoriteList
        totalCount.value = res.total
    } catch (err) {
        console.error('获取收藏列表失败', err)
    } finally {
        loading.value = false
    }
}

// 页码变化回调
const onPageChange = (page) => {
    currentPage.value = page
    fetchFavorites()
}

// 页面加载时获取
onMounted(fetchFavorites)

// 操作方法
const openFile = (row) => {
    if (row.is_dir) {
        // 跳转到目录页面
        router.push({ name: 'FileList', query: { parentId: row.id } })
    } else {
        // 文件预览，可以在新窗口打开或者使用内置预览组件
        window.open(row.fileURL, '_blank')
    }
}

const downloadFile = (row) => {
    if (!row.is_dir && row.fileURL) {
        const a = document.createElement('a')
        a.href = row.fileURL
        a.download = row.name
        a.click()
    }
}

const unfavorite = async (row) => {
    try {
        await cancelFavorite(row.id)
        starredItems.value = starredItems.value.filter(i => i.id !== row.id)
    } catch (err) {
        console.error('取消收藏失败', err)
    }
}

const locateFile = (row) => {
    // 跳转到所在目录，并传 fileId 用于高亮
    router.push({
        name: 'FileList',
        query: { parentId: row.parentId, highlightFileId: row.id }
    })
}
</script>

<style scoped>
.starred-container {
    padding: 24px;
    background: #fff;
    border-radius: 10px;
    box-shadow: 0 8px 20px rgb(0 0 0 / 0.06);
    height: 100%;
    display: flex;
    flex-direction: column;
}

.header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    flex-wrap: wrap;
    gap: 12px;
}

.header h2 {
    font-size: 20px;
    margin: 0;
}

.search-box {
    width: 300px;
}

.file-name-cell {
    display: flex;
    align-items: center;
    gap: 8px;
}

.el-table {
    flex: 1 1 auto;
    overflow: auto;
}

.pagination {
    position: sticky;
    bottom: 0;
    background: #fff;
    padding: 10px 0;
    display: flex;
    justify-content: center;
    border-top: 1px solid #f0f0f0;
}
</style>
