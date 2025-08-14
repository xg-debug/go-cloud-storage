<template>
    <div class="user-profile">
        <div class="profile-container">
            <!-- 用户信息头部 -->
            <div class="profile-header">
                <div class="user-info">
                    <div class="avatar-section">
                        <el-avatar :size="80" :src="user.avatar || defaultAvatar" class="user-avatar"/>
                        <div class="avatar-edit" @click="handleAvatarClick">
                            <el-icon>
                                <Camera/>
                            </el-icon>
                        </div>
                        <input ref="avatarInput" type="file" accept="image/*" style="display: none"
                               @change="handleAvatarChange"/>
                    </div>
                    <div class="user-details">
                        <h1 class="user-name">{{ user.username }}</h1>
                        <p class="user-email">{{ user.email }}</p>
                        <p class="user-meta">注册于 {{ user.registerTime }}</p>
                    </div>
                </div>
            </div>

            <!-- 内容区域 -->
            <div class="content-grid">
                <!-- 左侧：基本信息和存储统计 -->
                <div class="left-column">
                    <!-- 基本信息卡片 -->
                    <div class="info-card">
                        <div class="card-header">
                            <div class="card-icon">
                                <el-icon>
                                    <User/>
                                </el-icon>
                            </div>
                            <h3>基本信息</h3>
                        </div>
                        <div class="card-body">
                            <el-form :model="userForm" label-width="80px" class="info-form">
                                <el-form-item label="用户名">
                                    <el-input v-model="userForm.username" placeholder="请输入用户名"/>
                                </el-form-item>
                                <el-form-item label="手机号">
                                    <el-input v-model="userForm.phone" placeholder="请输入手机号"/>
                                </el-form-item>
                                <el-form-item>
                                    <el-button type="primary" @click="saveUserInfo" :loading="saving"
                                               class="action-btn">
                                        <el-icon>
                                            <Check/>
                                        </el-icon>
                                        保存修改
                                    </el-button>
                                </el-form-item>
                            </el-form>
                        </div>
                    </div>

                    <!-- 存储统计卡片 -->
                    <div class="stats-card">
                        <div class="card-header">
                            <div class="card-icon">
                                <el-icon>
                                    <DataAnalysis/>
                                </el-icon>
                            </div>
                            <h3>存储统计</h3>
                        </div>
                        <div class="card-body">
                            <div class="storage-visual">
                                <div class="storage-circle">
                                    <el-progress type="dashboard" :percentage="storageStats.percentage"
                                                 :color="storageColor"
                                                 :width="120">
                                        <template #default>
                                            <div class="circle-content">
                                                <div class="percentage">{{ storageStats.percentage }}%</div>
                                                <div class="storage-text">{{ storageStats.used }}GB /
                                                    {{ storageStats.total }}GB
                                                </div>
                                            </div>
                                        </template>
                                    </el-progress>
                                </div>
                                <div class="storage-details">
                                    <div class="detail-item">
                                        <div class="detail-icon">
                                            <el-icon>
                                                <Document/>
                                            </el-icon>
                                        </div>
                                        <div class="detail-content">
                                            <span class="detail-label">文件总数</span>
                                            <span class="detail-value">{{ fileStats.totalFiles }} 个</span>
                                        </div>
                                    </div>
                                    <div class="detail-item">
                                        <div class="detail-icon">
                                            <el-icon>
                                                <Folder/>
                                            </el-icon>
                                        </div>
                                        <div class="detail-content">
                                            <span class="detail-label">文件夹</span>
                                            <span class="detail-value">{{ fileStats.folders }} 个</span>
                                        </div>
                                    </div>
                                    <div class="detail-item">
                                        <div class="detail-icon">
                                            <el-icon>
                                                <Share/>
                                            </el-icon>
                                        </div>
                                        <div class="detail-content">
                                            <span class="detail-label">共享文件</span>
                                            <span class="detail-value">{{ fileStats.sharedFiles }} 个</span>
                                        </div>
                                    </div>
                                </div>
                            </div>
                        </div>
                    </div>
                </div>

                <!-- 右侧：文件类型和安全设置 -->
                <div class="right-column">
                    <!-- 文件类型分布卡片 -->
                    <div class="file-types-card">
                        <div class="card-header">
                            <div class="card-icon">
                                <el-icon>
                                    <FolderOpened/>
                                </el-icon>
                            </div>
                            <h3>文件类型分布</h3>
                        </div>
                        <div class="card-body">
                            <div class="file-types">
                                <div v-for="type in fileTypeStats" :key="type.type" class="file-type-item">
                                    <div class="type-header">
                                        <div class="type-icon" :style="{ backgroundColor: type.color + '20' }">
                                            <el-icon v-if="type.type === 'document'">
                                                <Document/>
                                            </el-icon>
                                            <el-icon v-else-if="type.type === 'image'">
                                                <Picture/>
                                            </el-icon>
                                            <el-icon v-else-if="type.type === 'video'">
                                                <VideoCamera/>
                                            </el-icon>
                                            <el-icon v-else-if="type.type === 'audio'">
                                                <Headset/>
                                            </el-icon>
                                            <el-icon v-else>
                                                <Files/>
                                            </el-icon>
                                        </div>
                                        <div class="type-info">
                                            <span class="type-name">{{ type.name }}</span>
                                            <span class="type-count">{{ type.count }} 个文件</span>
                                        </div>
                                        <span class="type-percentage">{{ type.percentage }}%</span>
                                    </div>
                                    <div class="type-progress">
                                        <div class="progress-bar"
                                             :style="{ width: type.percentage + '%', backgroundColor: type.color }">
                                        </div>
                                    </div>
                                </div>
                            </div>
                        </div>
                    </div>

                    <!-- 安全设置卡片 -->
                    <div class="security-card">
                        <div class="card-header">
                            <div class="card-icon">
                                <el-icon>
                                    <Lock/>
                                </el-icon>
                            </div>
                            <h3>安全设置</h3>
                        </div>
                        <div class="card-body">
                            <div class="security-item">
                                <div class="security-info">
                                    <h4>账户密码</h4>
                                    <p>定期更换密码以确保账户安全</p>
                                </div>
                                <el-button type="primary" @click="showPasswordDialog = true" class="action-btn">
                                    <el-icon>
                                        <Key/>
                                    </el-icon>
                                    修改密码
                                </el-button>
                            </div>
                        </div>
                    </div>
                </div>
            </div>
        </div>

        <!-- 修改密码对话框 -->
        <el-dialog v-model="showPasswordDialog" title="修改密码" width="400px" :close-on-click-modal="false">
            <el-form :model="passwordForm" :rules="passwordRules" ref="passwordFormRef" label-width="100px">
                <el-form-item label="当前密码" prop="oldPassword">
                    <el-input v-model="passwordForm.oldPassword" type="password" placeholder="请输入当前密码"
                              show-password/>
                </el-form-item>
                <el-form-item label="新密码" prop="newPassword">
                    <el-input v-model="passwordForm.newPassword" type="password" placeholder="请输入新密码"
                              show-password/>
                </el-form-item>
                <el-form-item label="确认密码" prop="confirmPassword">
                    <el-input v-model="passwordForm.confirmPassword" type="password" placeholder="请再次输入新密码"
                              show-password/>
                </el-form-item>
            </el-form>
            <template #footer>
                <el-button @click="showPasswordDialog = false">取消</el-button>
                <el-button type="primary" @click="changePassword" :loading="changingPassword">
                    确认修改
                </el-button>
            </template>
        </el-dialog>
    </div>
</template>

<script setup>
import {computed, onMounted, ref} from 'vue'
import {ElMessage} from 'element-plus'
import {useStore} from 'vuex'
import {updatePassword, updateProfile, uploadAvatar} from '@/api/user'

const store = useStore()
const userInfo = ref(store.state.userInfo)
const user = computed(() => store.state.userInfo)

const avatarInput = ref(null)
const passwordFormRef = ref(null)

// 响应式数据
const userForm = ref({
    username: '',
    phone: '',
})
const saving = ref(false)
const changingPassword = ref(false)
const showPasswordDialog = ref(false)

// 存储统计
const storageStats = ref({
    used: 0,
    total: 100,
    percentage: 0
})

const fileStats = ref({
    totalFiles: 0,
    folders: 0,
    sharedFiles: 0
})

const fileTypeStats = ref([])

// 修改密码表单
const passwordForm = ref({
    oldPassword: '',
    newPassword: '',
    confirmPassword: ''
})

// 默认头像
const defaultAvatar = 'https://cube.elemecdn.com/3/7c/3ea6beec64369c2642b92c5926d6.png'

// 计算属性
const storageColor = computed(() => {
    const percent = storageStats.value.percentage
    return percent > 90 ? '#f56c6c' : percent > 70 ? '#e6a23c' : '#409eff'
})

// 密码验证规则
const passwordRules = {
    oldPassword: [
        {required: true, message: '请输入当前密码', trigger: 'blur'}
    ],
    newPassword: [
        {required: true, message: '请输入新密码', trigger: 'blur'},
        {min: 6, message: '密码长度至少6位', trigger: 'blur'}
    ],
    confirmPassword: [
        {required: true, message: '请确认新密码', trigger: 'blur'},
        {
            validator: (rule, value, callback) => {
                if (value !== passwordForm.value.newPassword) {
                    callback(new Error('两次输入的密码不一致'))
                } else {
                    callback()
                }
            },
            trigger: 'blur'
        }
    ]
}

const loadStats = async () => {
    try {
        // 模拟统计数据，实际项目中应该调用API
        const mockStatsData = {
            usedStorage: 45.8,
            totalStorage: 100,
            totalFiles: 156,
            folders: 23,
            sharedFiles: 12
        }

        const mockFileTypeData = [
            {type: 'document', name: '文档', count: 45, percentage: 30},
            {type: 'image', name: '图片', count: 38, percentage: 25},
            {type: 'video', name: '视频', count: 25, percentage: 17},
            {type: 'audio', name: '音频', count: 18, percentage: 12},
            {type: 'other', name: '其他', count: 30, percentage: 20}
        ]

        // 如果有真实API，使用下面的代码
        // const [statsData, fileTypeData] = await Promise.all([
        //   getUserStats(),
        //   getFileTypeStats()
        // ])

        // 更新存储统计
        storageStats.value = {
            used: mockStatsData.usedStorage || 0,
            total: mockStatsData.totalStorage || 100,
            percentage: Math.round(((mockStatsData.usedStorage || 0) / (mockStatsData.totalStorage || 100)) * 100)
        }

        // 更新文件统计
        fileStats.value = {
            totalFiles: mockStatsData.totalFiles || 0,
            folders: mockStatsData.folders || 0,
            sharedFiles: mockStatsData.sharedFiles || 0
        }

        // 更新文件类型统计
        fileTypeStats.value = mockFileTypeData.map((type, index) => ({
            ...type,
            color: ['#409eff', '#67c23a', '#e6a23c', '#f56c6c', '#909399'][index % 5]
        }))
    } catch (error) {
        ElMessage.error('获取统计信息失败')
    }
}

const handleAvatarClick = () => {
    avatarInput.value.click()
}

const handleAvatarChange = async (event) => {
    const file = event.target.files[0]
    if (!file) return

    // 验证文件类型和大小
    if (!file.type.startsWith('image/')) {
        ElMessage.error('请选择图片文件')
        return
    }

    if (file.size > 5 * 1024 * 1024) {
        ElMessage.error('图片大小不能超过5MB')
        return
    }

    try {
        const formData = new FormData()
        formData.append('avatar', file)
        const res = await uploadAvatar(formData)
        const avatarUrl = res.avatar
        // 更新本地用户信息
        userInfo.value.avatar = avatarUrl
        store.commit('setUserInfo', { ...store.state.userInfo, avatar: avatarUrl })
        ElMessage.success('头像上传成功')
    } catch (error) {
        ElMessage.error('头像上传失败')
    } finally {
        // 清空input
        event.target.value = ''
    }
}

const saveUserInfo = async () => {
    try {
        saving.value = true
        await updateProfile(userForm.value)
        // 更新store中的用户信息
        store.commit('setUserInfo', {
            ...user.value,
            username: userForm.value.username,
            phone: userForm.value.phone
        })

        ElMessage.success('更新成功')
    } catch (error) {
        ElMessage.error('更新失败，请重试')
    } finally {
        saving.value = false
    }
}

const changePassword = async () => {
    try {
        await passwordFormRef.value.validate()
        changingPassword.value = true
        await updatePassword({
            oldPassword: passwordForm.value.oldPassword,
            newPassword: passwordForm.value.newPassword
        })

        ElMessage.success('密码修改成功')
        showPasswordDialog.value = false
        passwordForm.value = {
            oldPassword: '',
            newPassword: '',
            confirmPassword: ''
        }
    } catch (error) {
        if (error.message) {
            ElMessage.error(error.message)
        }
    } finally {
        changingPassword.value = false
    }
}

// 初始化用户表单数据
const initUserForm = () => {
    if (user.value) {
        userForm.value = {
            username: user.value.username || '',
            phone: user.value.phone || '',
        }
    }
}

// 生命周期
onMounted(() => {
    initUserForm()
    loadStats()
})
</script>

<style scoped>
.user-profile {
    background: #f8fafc;
    min-height: 100vh;
    padding: 32px;
}

.profile-container {
    max-width: 1200px;
    margin: 0 auto;
}

.profile-header {
    background: #fff;
    border-radius: 16px;
    padding: 32px;
    margin-bottom: 32px;
    box-shadow: 0 4px 20px rgba(0, 0, 0, 0.08);
    border: 1px solid #e2e8f0;
}

.user-info {
    display: flex;
    align-items: center;
    gap: 24px;
}

.avatar-section {
    position: relative;
}

.user-avatar {
    border: 4px solid #fff;
    box-shadow: 0 4px 16px rgba(0, 0, 0, 0.15);
}

.avatar-edit {
    position: absolute;
    right: -8px;
    bottom: -8px;
    background: linear-gradient(135deg, #3b82f6 0%, #1d4ed8 100%);
    color: #fff;
    border-radius: 50%;
    width: 32px;
    height: 32px;
    display: flex;
    align-items: center;
    justify-content: center;
    cursor: pointer;
    border: 2px solid #fff;
    box-shadow: 0 2px 8px rgba(0, 0, 0, 0.15);
    transition: all 0.3s;
}

.avatar-edit:hover {
    transform: scale(1.1);
    box-shadow: 0 4px 12px rgba(0, 0, 0, 0.2);
}

.user-details {
    flex: 1;
}

.user-name {
    margin: 0 0 8px 0;
    font-size: 28px;
    font-weight: 700;
    color: #1e293b;
    background: linear-gradient(135deg, #3b82f6 0%, #1d4ed8 100%);
    -webkit-background-clip: text;
    -webkit-text-fill-color: transparent;
    background-clip: text;
}

.user-email {
    margin: 0 0 4px 0;
    color: #64748b;
    font-size: 16px;
}

.user-meta {
    margin: 0;
    color: #94a3b8;
    font-size: 14px;
}

.content-grid {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 24px;
}

.left-column,
.right-column {
    display: flex;
    flex-direction: column;
    gap: 24px;
}

.info-card,
.stats-card,
.file-types-card,
.security-card {
    background: #fff;
    border-radius: 16px;
    box-shadow: 0 4px 20px rgba(0, 0, 0, 0.08);
    border: 1px solid #e2e8f0;
    transition: all 0.3s ease;
    overflow: hidden;
}

.info-card:hover,
.stats-card:hover,
.file-types-card:hover,
.security-card:hover {
    transform: translateY(-4px);
    box-shadow: 0 8px 32px rgba(0, 0, 0, 0.12);
}

.card-header {
    background: linear-gradient(135deg, #f7fafc 0%, #edf2f7 100%);
    padding: 20px 24px;
    border-bottom: 1px solid #e2e8f0;
    display: flex;
    align-items: center;
    gap: 12px;
}

.card-icon {
    width: 40px;
    height: 40px;
    background: linear-gradient(135deg, #3b82f6 0%, #1d4ed8 100%);
    border-radius: 10px;
    display: flex;
    align-items: center;
    justify-content: center;
    color: #fff;
    font-size: 18px;
}

.card-header h3 {
    margin: 0;
    font-size: 18px;
    font-weight: 600;
    color: #1e293b;
}

.card-body {
    padding: 24px;
}

.info-form {
    margin-top: 0;
}

.action-btn {
    background: linear-gradient(135deg, #3b82f6 0%, #1d4ed8 100%);
    border: none;
    border-radius: 8px;
    font-weight: 500;
    display: flex;
    align-items: center;
    gap: 6px;
    transition: all 0.3s;
}

.action-btn:hover {
    transform: translateY(-1px);
    box-shadow: 0 4px 12px rgba(59, 130, 246, 0.3);
}

.storage-visual {
    display: flex;
    align-items: center;
    gap: 32px;
}

.storage-circle {
    flex-shrink: 0;
}

.circle-content {
    text-align: center;
}

.percentage {
    font-size: 24px;
    font-weight: 700;
    color: #1e293b;
    line-height: 1;
}

.storage-text {
    font-size: 12px;
    color: #64748b;
    margin-top: 4px;
}

.storage-details {
    flex: 1;
    display: flex;
    flex-direction: column;
    gap: 16px;
}

.detail-item {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 12px;
    background: #f8fafc;
    border-radius: 8px;
    transition: all 0.3s;
}

.detail-item:hover {
    background: #f1f5f9;
    transform: translateX(4px);
}

.detail-icon {
    width: 32px;
    height: 32px;
    background: linear-gradient(135deg, #3b82f6 0%, #1d4ed8 100%);
    border-radius: 8px;
    display: flex;
    align-items: center;
    justify-content: center;
    color: #fff;
}

.detail-content {
    flex: 1;
    display: flex;
    flex-direction: column;
    gap: 2px;
}

.detail-label {
    font-size: 12px;
    color: #64748b;
}

.detail-value {
    font-size: 16px;
    font-weight: 600;
    color: #1e293b;
}

.file-types {
    display: flex;
    flex-direction: column;
    gap: 20px;
}

.file-type-item {
    display: flex;
    flex-direction: column;
    gap: 8px;
}

.type-header {
    display: flex;
    align-items: center;
    gap: 12px;
}

.type-icon {
    width: 36px;
    height: 36px;
    border-radius: 8px;
    display: flex;
    align-items: center;
    justify-content: center;
    color: #1e293b;
    font-size: 16px;
}

.type-info {
    flex: 1;
    display: flex;
    flex-direction: column;
    gap: 2px;
}

.type-name {
    font-size: 14px;
    font-weight: 600;
    color: #1e293b;
}

.type-count {
    font-size: 12px;
    color: #64748b;
}

.type-percentage {
    font-size: 16px;
    font-weight: 700;
    color: #3b82f6;
}

.type-progress {
    height: 8px;
    background: #e2e8f0;
    border-radius: 4px;
    overflow: hidden;
}

.progress-bar {
    height: 100%;
    border-radius: 4px;
    transition: width 0.6s ease;
}

.security-item {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 16px;
}

.security-info h4 {
    margin: 0 0 4px 0;
    font-size: 16px;
    font-weight: 600;
    color: #1e293b;
}

.security-info p {
    margin: 0;
    font-size: 14px;
    color: #64748b;
}

/* 响应式设计 */
@media (max-width: 1024px) {
    .content-grid {
        grid-template-columns: 1fr;
        gap: 20px;
    }

    .storage-visual {
        flex-direction: column;
        gap: 20px;
        text-align: center;
    }
}

@media (max-width: 768px) {
    .user-profile {
        padding: 16px;
    }

    .profile-header {
        padding: 24px;
        margin-bottom: 24px;
    }

    .user-info {
        flex-direction: column;
        text-align: center;
        gap: 16px;
    }

    .user-name {
        font-size: 24px;
    }

    .content-grid {
        gap: 16px;
    }

    .card-body {
        padding: 20px;
    }

    .security-item {
        flex-direction: column;
        align-items: flex-start;
        gap: 12px;
    }
}
</style>