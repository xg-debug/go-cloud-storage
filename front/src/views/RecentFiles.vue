<template>
    <div class="recent-container">
        <div class="header">
            <h2>最近文件</h2>
            <el-select v-model="timeRange" placeholder="时间范围" size="small" style="width: 120px">
                <el-option label="今天" value="today"/>
                <el-option label="本周" value="week"/>
                <el-option label="本月" value="month"/>
            </el-select>
        </div>

        <el-divider/>

        <el-empty v-if="filteredFiles.length === 0" description="暂无最近文件"/>

        <el-timeline v-else>
            <el-timeline-item
                    v-for="(day, index) in filteredFiles"
                    :key="index"
                    :timestamp="day.date"
                    placement="top"
            >
                <el-table :data="day.files" style="width: 100%" :fit="true" border>
                    <el-table-column prop="name" label="文件名" min-width="240">
                        <template #default="{ row }">
                            <div class="file-name-cell">
                                <el-icon :color="getIconColor(row)">
                                    <component :is="getIconComponent(row)"/>
                                </el-icon>
                                <span>{{ row.name }}</span>
                            </div>
                        </template>
                    </el-table-column>

                    <el-table-column prop="modified" label="修改时间" min-width="100"/>
                    <el-table-column label="大小" min-width="100">
                        <template #default="{ row }">
                            {{ row.size_str }}
                        </template>
                    </el-table-column>
                    <el-table-column label="操作" min-width="120">
                        <template #default="{ row }">
                            <el-button size="small" type="text" @click="handleOpen(row)">打开</el-button>
                            <el-button size="small" type="text" @click="handleLocate(row)">定位</el-button>
                        </template>
                    </el-table-column>
                </el-table>
            </el-timeline-item>
        </el-timeline>
    </div>
</template>

<script setup>
import {computed, ref, watch, onMounted} from 'vue'
import {Document, Folder} from '@element-plus/icons-vue'
import {getRecentFiles} from "@/api/file";

// 时间范围
const timeRange = ref('week')
const allFiles = ref([])

// 图标选择逻辑
function getIconComponent(file) {
    if (file.type === 'folder') return Folder
    const ext = file.name.split('.').pop().toLowerCase()
    if (['doc', 'docx'].includes(ext)) return Document
    if (['xlsx', 'xls', 'csv'].includes(ext)) return Document
    if (['ppt', 'pptx'].includes(ext)) return Document
    if (['pdf'].includes(ext)) return Document
    return Document
}

function getIconColor(file) {
    if (file.type === 'folder') return '#FFB800'
    const ext = file.name.split('.').pop().toLowerCase()
    if (['doc', 'docx'].includes(ext)) return '#1E90FF'
    if (['xlsx', 'xls', 'csv'].includes(ext)) return '#27ae60'
    if (['ppt', 'pptx'].includes(ext)) return '#e67e22'
    if (['pdf'].includes(ext)) return '#e74c3c'
    return '#3a86ff'
}

// 拉取数据方法
async function fetchRecentFiles() {
    try {
        const res = await getRecentFiles(timeRange.value)
        allFiles.value = res
    } catch (err) {
        console.error('获取最近文件失败', err)
    }
}

// 监听时间范围变化
watch(timeRange, () => {
    fetchRecentFiles()
})

// 首次加载
onMounted(() => {
    fetchRecentFiles()
})

const filteredFiles = computed(() => allFiles.value) // 后端已经按 timeRange 过滤好

// 操作按钮方法
function handleOpen(file) {
    console.log('打开文件:', file)
    // TODO: 实现文件打开逻辑
}

function handleLocate(file) {
    console.log('定位文件:', file)
    // TODO: 实现文件定位逻辑
}
</script>

<style scoped>
.recent-container {
    padding: 24px;
    background: #fff;
    border-radius: 10px;
    box-shadow: 0 8px 20px rgb(0 0 0 / 0.06);
    height: 100%;
    overflow-y: auto;
}

.header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 16px;
}

.header h2 {
    margin: 0;
    font-size: 20px;
    font-weight: 600;
    color: #333;
}

.file-name-cell {
    display: flex;
    align-items: center;
    gap: 8px;
}

.el-icon {
    font-size: 18px;
}

.el-timeline {
    padding-left: 20px;
}

.el-table {
    margin-top: 16px;
    font-size: 14px;
    border-radius: 8px;
    overflow: hidden;
}

/* 按钮样式美化 */
:deep(.el-table__cell .cell) {
    display: flex;
    gap: 6px;
    align-items: center;
}

.el-button {
    padding: 0 6px;
    font-size: 13px;
}
</style>
  