<template>
    <div class="header-container">
        <!-- 左侧：Logo -->
        <div class="header-left">
            <div class="logo">CloudLite</div>
        </div>

        <!-- 右侧：搜索框 + 通知 + 用户菜单 -->
        <div class="header-right">

            <el-input
                v-model="searchQuery"
                placeholder="搜索文件/文件夹"
                class="search-input"
                clearable
                @keyup.enter="handleSearch"
            >
                <template #prefix>
                    <el-icon class="search-icon">
                        <Search/>
                    </el-icon>
                </template>
            </el-input>

            <!-- 通知中心 -->
            <el-dropdown trigger="click" class="notification-dropdown">
                <el-badge :value="unreadCount" class="notification-badge">
                    <el-icon class="notification-icon">
                        <Bell/>
                    </el-icon>
                </el-badge>
                <template #dropdown>
                    <el-dropdown-menu class="notification-menu">
                        <!-- 全部标记为已读按钮 -->
                        <div v-if="notifications.length > 0" class="mark-all-read">
                            <el-button type="text" size="small" @click.stop="markAllAsRead">
                                全部标记为已读
                            </el-button>
                        </div>

                        <el-dropdown-item v-if="notifications.length === 0" disabled>
                            暂无通知
                        </el-dropdown-item>

                        <div
                            v-for="(item, index) in notifications"
                            :key="item.id"
                            class="notification-item"
                        >
              <span @click="handleNotificationClick(item)">
                {{ item.message }}
              </span>
                            <el-button
                                type="text"
                                icon="el-icon-check"
                                class="mark-read-btn"
                                @click.stop="markAsRead(index)"
                            />
                        </div>
                    </el-dropdown-menu>
                </template>
            </el-dropdown>

            <!-- 用户菜单 -->
            <el-dropdown class="user-dropdown">
                <div class="user-avatar">
                    <el-avatar :size="40" :src="user?.avatar" class="avatar"/>
                    <div class="user-info" v-if="user?.username">
                        <span class="username">{{ user.username }}</span>
                    </div>
                    <el-icon class="dropdown-icon">
                        <ArrowDown/>
                    </el-icon>
                </div>
                <template #dropdown>
                    <el-dropdown-menu class="user-menu">
                        <el-dropdown-item @click="goToProfile" class="menu-item">
                            <el-icon>
                                <User/>
                            </el-icon>
                            <span>个人中心</span>
                        </el-dropdown-item>
                        <el-dropdown-item @click="goToSettings" class="menu-item">
                            <el-icon>
                                <Setting/>
                            </el-icon>
                            <span>设置</span>
                        </el-dropdown-item>
                        <el-dropdown-item divided @click="handleLogout" class="menu-item logout">
                            <el-icon>
                                <SwitchButton/>
                            </el-icon>
                            <span>退出登录</span>
                        </el-dropdown-item>
                    </el-dropdown-menu>
                </template>
            </el-dropdown>
        </div>
    </div>
</template>

<script setup>
import {ref, computed} from 'vue'
import {useRouter} from 'vue-router'
import {useStore} from 'vuex'
import {logout} from '@/api/auth'

const router = useRouter()
const store = useStore()

const searchQuery = ref('')

// 用户信息
const user = computed(() => store.state.userInfo)

// 搜索
const handleSearch = () => {
    if (!searchQuery.value.trim()) return
    router.push({name: 'Search', query: {q: searchQuery.value.trim()}})
}

// 用户菜单
const goToProfile = () => router.push({name: 'UserProfile'})
const goToSettings = () => router.push({name: 'Settings'})
const handleLogout = async () => {
    try {
        await logout()
        store.commit('clearAuth')
        ElMessage.success('退出成功')
        router.push('/login')
    } catch (error) {
        console.error('退出失败:', error)
        store.commit('clearAuth')
        ElMessage.error('退出异常')
        router.push('/login')
    }
}

// 通知中心示例数据
const notifications = ref([
    {id: 1, message: '共享文件 "工作文档.pdf" 已更新', link: '/file/123'},
    {id: 2, message: '系统维护将在今晚 12 点开始', link: '/system'}
])
const unreadCount = computed(() => notifications.value.length)

// 点击通知
const handleNotificationClick = (item) => {
    if (item.link) {
        router.push(item.link)
    }
}

// 单条标记已读
const markAsRead = (index) => {
    notifications.value.splice(index, 1)
}

// 全部标记已读
const markAllAsRead = () => {
    notifications.value = []
}
</script>

<style scoped>
.header-container {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 0 32px;
    height: 64px;
    background: transparent;
}

.header-left {
    display: flex;
    align-items: center;
    gap: 24px;
}

.logo {
    font-weight: bold;
    font-size: 18px;
    color: #1f2937;
}

.search-input {
    width: 320px;
}

.search-input :deep(.el-input__wrapper) {
    border-radius: 12px;
    background: #f1f5f9;
}

.search-icon {
    color: #94a3b8;
}

.header-right {
    display: flex;
    align-items: center;
    gap: 16px;
}

/* 通知 */
.notification-badge {
    cursor: pointer;
}

.notification-icon {
    font-size: 20px;
    color: #64748b;
}

.notification-menu {
    width: 260px;
}

.mark-all-read {
    padding: 6px 16px;
    text-align: right;
    border-bottom: 1px solid #e2e8f0;
}

.notification-item {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 8px 16px;
    cursor: pointer;
    font-size: 14px;
}

.notification-item:hover {
    background: #f1f5f9;
}

.mark-read-btn {
    font-size: 14px;
    color: #94a3b8;
}

/* 用户菜单 */
.user-avatar {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 6px 12px;
    border-radius: 12px;
    background: #f8fafc;
    border: 1px solid #e2e8f0;
    cursor: pointer;
    transition: all 0.3s;
}

.user-avatar:hover {
    background: #f1f5f9;
    border-color: #cbd5e1;
    transform: translateY(-1px);
}

.avatar {
    border: 2px solid #fff;
}

.username {
    font-size: 14px;
    font-weight: 600;
    color: #1e293b;
}
.dropdown-icon {
    font-size: 12px;
    color: #94a3b8;
    transition: transform 0.3s;
}
.user-dropdown:hover .dropdown-icon {
    transform: rotate(180deg);
}

/* 响应式 */
@media (max-width: 768px) {
    .search-input {
        width: 200px;
    }

    .user-info {
        display: none;
    }
}
</style>
