<template>
    <div class="header-container">
        <!-- 左侧：Logo -->
        <div class="header-left">
            <div class="logo-wrapper">
                <div class="logo-icon">
                    <el-icon :size="24"><Cloudy /></el-icon>
                </div>
                <div class="logo-text">
                    <span class="logo-title">CloudDisk</span>
                    <span class="logo-subtitle">云存储</span>
                </div>
            </div>
        </div>

        <!-- 右侧：通知 + 用户菜单 -->
        <div class="header-right">
            <!-- 快捷操作按钮 -->
            <div class="quick-actions">
                <el-tooltip content="上传文件" placement="bottom">
                    <div class="action-btn">
                        <el-icon :size="20"><Upload /></el-icon>
                    </div>
                </el-tooltip>
            </div>

            <!-- 通知中心 -->
            <el-dropdown trigger="click" class="notification-dropdown">
                <div class="notification-wrapper">
                    <el-badge :value="unreadCount" :hidden="unreadCount === 0" class="notification-badge">
                        <div class="action-btn">
                            <el-icon :size="20"><Bell /></el-icon>
                        </div>
                    </el-badge>
                </div>
                <template #dropdown>
                    <el-dropdown-menu class="notification-menu">
                        <div class="notification-header">
                            <span class="notification-title">通知中心</span>
                            <el-button v-if="notifications.length > 0" type="text" size="small" @click.stop="markAllAsRead">
                                全部已读
                            </el-button>
                        </div>

                        <div v-if="notifications.length === 0" class="notification-empty">
                            <el-icon :size="48" color="#cbd5e1"><Bell /></el-icon>
                            <p>暂无新通知</p>
                        </div>

                        <div v-else class="notification-list">
                            <div
                                v-for="(item, index) in notifications"
                                :key="item.id"
                                class="notification-item"
                                @click="handleNotificationClick(item)"
                            >
                                <div class="notification-dot"></div>
                                <div class="notification-content">
                                    <span class="notification-text">{{ item.message }}</span>
                                    <span class="notification-time">刚刚</span>
                                </div>
                                <el-button
                                    type="text"
                                    class="mark-read-btn"
                                    @click.stop="markAsRead(index)"
                                >
                                    <el-icon><Check /></el-icon>
                                </el-button>
                            </div>
                        </div>
                    </el-dropdown-menu>
                </template>
            </el-dropdown>

            <!-- 用户菜单 -->
            <el-dropdown class="user-dropdown" trigger="click">
                <div class="user-avatar-wrapper">
                    <el-avatar :size="38" :src="user?.avatar" class="avatar">
                        <el-icon :size="20"><User /></el-icon>
                    </el-avatar>
                    <div class="user-info" v-if="user?.username">
                        <span class="username">{{ user.username }}</span>
                        <span class="user-role">个人版</span>
                    </div>
                    <el-icon class="dropdown-icon"><ArrowDown /></el-icon>
                </div>
                <template #dropdown>
                    <el-dropdown-menu class="user-menu">
                        <div class="user-menu-header">
                            <el-avatar :size="48" :src="user?.avatar" class="menu-avatar">
                                <el-icon :size="24"><User /></el-icon>
                            </el-avatar>
                            <div class="menu-user-info">
                                <span class="menu-username">{{ user?.username || '用户' }}</span>
                                <span class="menu-email">{{ user?.email || '' }}</span>
                            </div>
                        </div>
                        <el-dropdown-item @click="goToProfile" class="menu-item">
                            <el-icon><User /></el-icon>
                            <span>个人中心</span>
                        </el-dropdown-item>
                        <el-dropdown-item @click="goToSettings" class="menu-item">
                            <el-icon><Setting /></el-icon>
                            <span>系统设置</span>
                        </el-dropdown-item>
                        <el-dropdown-item divided @click="handleLogout" class="menu-item logout">
                            <el-icon><SwitchButton /></el-icon>
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
import {Cloudy, Search, Upload, Check} from '@element-plus/icons-vue'

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
    padding: 0 8px;
    height: 100%;
    background: transparent;
}

/* Logo 区域 */
.header-left {
    display: flex;
    align-items: center;
    flex-shrink: 0;
}

.logo-wrapper {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 8px 16px;
    border-radius: var(--radius-lg);
    cursor: pointer;
    transition: all var(--transition-fast);
}

.logo-wrapper:hover {
    background: rgba(99, 102, 241, 0.08);
}

.logo-icon {
    width: 42px;
    height: 42px;
    background: var(--primary-gradient);
    border-radius: var(--radius-md);
    display: flex;
    align-items: center;
    justify-content: center;
    color: white;
    box-shadow: 0 4px 12px rgba(99, 102, 241, 0.3);
}

.logo-text {
    display: flex;
    flex-direction: column;
}

.logo-title {
    font-size: 18px;
    font-weight: 700;
    color: var(--text-primary);
    letter-spacing: -0.5px;
    line-height: 1.2;
}

.logo-subtitle {
    font-size: 11px;
    color: var(--text-tertiary);
    font-weight: 500;
}

/* 右侧区域 */
.header-right {
    display: flex;
    align-items: center;
    gap: 8px;
    flex-shrink: 0;
}

.quick-actions {
    display: flex;
    align-items: center;
    gap: 4px;
}

.action-btn {
    width: 40px;
    height: 40px;
    border-radius: var(--radius-md);
    display: flex;
    align-items: center;
    justify-content: center;
    cursor: pointer;
    color: var(--text-secondary);
    transition: all var(--transition-fast);
}

.action-btn:hover {
    background: rgba(99, 102, 241, 0.1);
    color: var(--primary-color);
}

/* 通知 */
.notification-wrapper {
    display: flex;
    align-items: center;
}

.notification-badge :deep(.el-badge__content) {
    background: var(--danger-color);
    border: 2px solid white;
}

.notification-menu {
    width: 360px;
    padding: 0 !important;
    overflow: hidden;
}

.notification-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 16px 20px;
    border-bottom: 1px solid var(--border-light);
    background: var(--bg-secondary);
}

.notification-title {
    font-size: 15px;
    font-weight: 600;
    color: var(--text-primary);
}

.notification-empty {
    padding: 40px 20px;
    text-align: center;
}

.notification-empty p {
    margin-top: 12px;
    color: var(--text-tertiary);
    font-size: 14px;
}

.notification-list {
    max-height: 320px;
    overflow-y: auto;
}

.notification-item {
    display: flex;
    align-items: flex-start;
    gap: 12px;
    padding: 14px 20px;
    cursor: pointer;
    transition: background var(--transition-fast);
    border-bottom: 1px solid var(--border-light);
}

.notification-item:last-child {
    border-bottom: none;
}

.notification-item:hover {
    background: var(--bg-secondary);
}

.notification-dot {
    width: 8px;
    height: 8px;
    background: var(--primary-color);
    border-radius: 50%;
    margin-top: 6px;
    flex-shrink: 0;
}

.notification-content {
    flex: 1;
    min-width: 0;
}

.notification-text {
    display: block;
    font-size: 14px;
    color: var(--text-primary);
    line-height: 1.5;
    margin-bottom: 4px;
}

.notification-time {
    font-size: 12px;
    color: var(--text-tertiary);
}

.mark-read-btn {
    padding: 4px;
    color: var(--text-tertiary);
    opacity: 0;
    transition: all var(--transition-fast);
}

.notification-item:hover .mark-read-btn {
    opacity: 1;
}

.mark-read-btn:hover {
    color: var(--success-color);
}

/* 用户菜单 */
.user-avatar-wrapper {
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 6px 12px 6px 6px;
    border-radius: var(--radius-full);
    background: rgba(241, 245, 249, 0.6);
    border: 1px solid transparent;
    cursor: pointer;
    transition: all var(--transition-fast);
}

.user-avatar-wrapper:hover {
    background: rgba(241, 245, 249, 1);
    border-color: var(--border-light);
    box-shadow: var(--shadow-sm);
}

.avatar {
    border: 2px solid white;
    box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
}

.user-info {
    display: flex;
    flex-direction: column;
    line-height: 1.3;
}

.username {
    font-size: 13px;
    font-weight: 600;
    color: var(--text-primary);
}

.user-role {
    font-size: 11px;
    color: var(--text-tertiary);
}

.dropdown-icon {
    font-size: 12px;
    color: var(--text-tertiary);
    transition: transform var(--transition-fast);
    margin-left: 4px;
}

.user-dropdown:hover .dropdown-icon {
    transform: rotate(180deg);
}

/* 用户菜单下拉 */
.user-menu {
    width: 240px;
    padding: 0 !important;
    overflow: hidden;
}

.user-menu-header {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 20px;
    background: linear-gradient(135deg, var(--primary-light) 0%, #faf5ff 100%);
    border-bottom: 1px solid var(--border-light);
}

.menu-avatar {
    border: 3px solid white;
    box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
}

.menu-user-info {
    display: flex;
    flex-direction: column;
}

.menu-username {
    font-size: 15px;
    font-weight: 600;
    color: var(--text-primary);
}

.menu-email {
    font-size: 12px;
    color: var(--text-secondary);
    margin-top: 2px;
}

.user-menu .menu-item {
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 12px 20px !important;
    margin: 4px 8px !important;
    border-radius: var(--radius-md) !important;
    font-size: 14px;
    color: var(--text-secondary);
    transition: all var(--transition-fast);
}

.user-menu .menu-item:hover {
    background: var(--primary-light) !important;
    color: var(--primary-color) !important;
}

.user-menu .menu-item.logout {
    color: var(--danger-color);
}

.user-menu .menu-item.logout:hover {
    background: var(--danger-light) !important;
    color: var(--danger-color) !important;
}

/* 响应式 */
@media (max-width: 1024px) {
    .header-center {
        padding: 0 20px;
    }
}

@media (max-width: 768px) {
    .header-center {
        display: none;
    }

    .logo-text {
        display: none;
    }

    .user-info {
        display: none;
    }

    .quick-actions {
        display: none;
    }
}
</style>
