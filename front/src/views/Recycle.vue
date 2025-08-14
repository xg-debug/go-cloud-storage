<template>
    <div class="trash-container">
        <div class="header">
            <h2>回收站</h2>
            <div class="actions">
                <el-button
                        type="danger"
                        @click="openClearDialog"
                        plain
                        round
                >
                    <el-icon>
                        <Delete/>
                    </el-icon>
                    清空回收站
                </el-button>
                <el-button
                    type="danger"
                    :disabled="selectedItems.length === 0"
                    @click="handleBatchDelete"
                    plain
                    round
                >
                    <el-icon>
                        <Delete/>
                    </el-icon>
                    批量删除
                </el-button>
                <el-button
                        :disabled="selectedItems.length === 0"
                        @click="handleRestore"
                        plain
                        round
                >
                    <el-icon>
                        <Refresh/>
                    </el-icon>
                    还原
                </el-button>
            </div>
        </div>

        <el-divider/>

        <el-table
                :data="trashItems"
                style="width: 100%"
                @selection-change="handleSelectionChange"
                stripe
                border
                :row-key="row => row.fileId"
                empty-text="回收站空空如也"
        >
            <el-table-column type="selection" width="55"/>

            <el-table-column label="名称" min-width="160">
                <template #default="{ row }">
                    <div class="file-name-cell" title="点击打开">
                        <el-icon
                                :color="row.is_dir === true ? '#FFB800' : '#3a86ff'"
                                class="file-icon"
                        >
                            <component :is="row.is_dir === true ? Folder : Document"/>
                        </el-icon>
                        <span class="file-name" @click="handleOpen(row)">{{ row.name }}</span>
                    </div>
                </template>
            </el-table-column>

            <el-table-column prop="size" label="大小" width="120" align="center">
                <template #default="{ row }">
                    {{ row.size_str }}
                </template>
            </el-table-column>
            <el-table-column prop="deletedDate" label="删除时间" width="160" align="center">
                <template #default="{ row }">
                    {{ row.deletedDate }}
                </template>
            </el-table-column>
            <el-table-column prop="expireDays" label="有效时间" width="160" align="center">
                <template #default="{ row }">
                    {{ row.expireDays }}天
                </template>
            </el-table-column>
            <el-table-column label="操作" fixed="right" width="180" align="center">
                <template #default="{ row }">
                    <!-- 单个还原 -->
                    <el-button
                        size="small"
                        type="primary"
                        text
                        @click="handleRestoreOne(row)"
                        title="还原"
                    >
                        还原
                    </el-button>

                    <!-- 单个彻底删除 -->
                    <el-button
                        size="small"
                        type="danger"
                        text
                        @click="openDeleteDialog(row)"
                        title="彻底删除"
                    >
                        彻底删除
                    </el-button>
                </template>
            </el-table-column>

        </el-table>

        <!-- 清空回收站弹窗 -->
        <el-dialog
            v-model="clearDialogVisible"
            title="清空回收站"
            width="400px"
            :before-close="handleClearDialogClose"
        >
            <div class="delete-confirm-text">
                <div>确认清空回收站？</div>
            </div>
            <template #footer>
                <el-button @click="clearDialogVisible = false">取消</el-button>
                <el-button type="primary" @click="confirmClear" :loading="deleting">确定</el-button>
            </template>
        </el-dialog>

        <!-- 彻底删除弹窗 -->
        <el-dialog
            v-model="deleteDialogVisible"
            title="彻底删除"
            width="400px"
            :before-close="handleDeleteDialogClose"
        >
            <div class="delete-confirm-text">
                <div>文件删除后将无法恢复，您确认要彻底删除所选文件吗？</div>
            </div>
            <template #footer>
                <el-button @click="deleteDialogVisible = false">取消</el-button>
                <el-button type="primary" @click="">确定</el-button>
            </template>
        </el-dialog>
    </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { Delete, Document, Folder, Refresh } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import {
    loadSoftDeletedFiles,
    deletePermanent,
    deleteSelected,
    clearRecycleBin,
    restore,
    restoreBatch,
} from '@/api/recycle'

const deleteDialogVisible = ref(false)
const deleteTarget = ref({})
const deleting = ref(false)

const clearDialogVisible = ref(false)

const selectedItems = ref([])
const trashItems = ref([])

// 页面初始化加载回收站数据
const fetchTrashItems = async () => {
    try {
        const res = await loadSoftDeletedFiles()
        trashItems.value = res.data || []
    } catch (error) {
        ElMessage.error('加载回收站数据失败')
    }
}

onMounted(() => {
    fetchTrashItems()
})

// 多选变化
const handleSelectionChange = (selection) => {
    selectedItems.value = selection
}

// 打开文件/文件夹
const handleOpen = (item) => {
    ElMessage.info(`打开文件: ${item.name}`)
}

const openClearDialog = () => {
    clearDialogVisible.value = true
}

// 清空回收站
const confirmClear = async () => {
    try {
        await clearRecycleBin()
        trashItems.value = []
        selectedItems.value = []
        ElMessage.success('回收站已清空')
        clearDialogVisible.value = false
    } catch (error) {
        if (error !== 'cancel') {
            ElMessage.error('清空失败')
        }
    }
}

const handleClearDialogClose = () => {
    clearDialogVisible.value = false
}

// 批量删除
const handleBatchDelete = async () => {
    if (selectedItems.value.length === 0) return
    try {
        const deleteIds = selectedItems.value.map((item) => item.fileId)
        await deleteSelected(deleteIds)
        trashItems.value = trashItems.value.filter((item) => !deleteIds.includes(item.fileId))
        selectedItems.value = []
        ElMessage.success('选中文件已彻底删除')
    } catch (error) {
        ElMessage.error('删除失败')
    }
}

// 批量还原
const handleRestore = async () => {
    if (selectedItems.value.length === 0) return
    try {
        const restoredIds = selectedItems.value.map((item) => item.fileId)
        await restoreBatch(restoredIds)
        trashItems.value = trashItems.value.filter((item) => !restoredIds.includes(item.fileId))
        selectedItems.value = []
        ElMessage.success('选中文件已还原')
    } catch (error) {
        ElMessage.error('还原失败')
    }
}

// 单个还原
const handleRestoreOne = async (row) => {
    try {
        await restore(row.fileId)
        trashItems.value = trashItems.value.filter((item) => item.fileId !== row.fileId)
        ElMessage.success(`已还原 ${row.name}`)
    } catch (error) {
        ElMessage.error('还原失败')
    }
}

// 彻底删除 - 打开确认弹窗
const openDeleteDialog = (row) => {
    deleteTarget.value = row
    deleteDialogVisible.value = true
}

// 确认彻底删除
const confirmDelete = async () => {
    deleting.value = true
    try {
        await deletePermanent(deleteTarget.value.fileId)
        ElMessage.success('删除成功')
        deleteDialogVisible.value = false
        fetchTrashItems()
    } catch (error) {
        ElMessage.error('删除失败')
    } finally {
        deleting.value = false
        deleteTarget.value = {}
    }
}

// 关闭弹窗
const handleDeleteDialogClose = () => {
    deleteDialogVisible.value = false
    deleteTarget.value = {}
    deleting.value = false
}
</script>

<style scoped>
.trash-container {
    padding: 24px;
    background: #fff;
    border-radius: 10px;
    box-shadow: 0 8px 20px rgb(0 0 0 / 0.06);
    height: 100%;
    overflow-y: auto;
}

.header {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 20px;
    gap: 12px;
}

.header h2 {
    font-size: 26px;
    font-weight: 700;
    color: #2c3e50;
    margin: 0;
}

.actions {
    display: flex;
    gap: 12px;
    flex-wrap: wrap;
}

.file-name-cell {
    display: flex;
    align-items: center;
    gap: 10px;
    cursor: pointer;
    user-select: none;
}

.file-icon {
    font-size: 20px;
}

.file-name {
    font-weight: 600;
    color: #409eff;
    transition: color 0.25s ease;
}

.file-name:hover {
    color: #1f63d6;
    text-decoration: underline;
}

.el-table th,
.el-table td {
    vertical-align: middle !important;
}

@media (max-width: 768px) {
    .header {
        flex-direction: column;
        align-items: flex-start;
    }

    .actions {
        width: 100%;
        justify-content: flex-start;
    }
}

.delete-confirm-text {
    text-align: center; /* 文字居中 */
    font-size: 14px;
    line-height: 1.8; /* 行高，保证两行间距合适 */
    user-select: none;
}
</style>