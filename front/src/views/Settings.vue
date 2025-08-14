<template>
    <div class="settings-page">
        <div class="settings-container">
            <el-card class="settings-card">
                <template #header>
                    <div class="card-header">
                        <el-icon>
                            <Setting/>
                        </el-icon>
                        <span>系统设置</span>
                    </div>
                </template>

                <el-tabs v-model="activeTab" class="settings-tabs">
                    <!-- 界面设置 -->
                    <el-tab-pane label="界面设置" name="interface">
                        <div class="tab-content">
                            <el-form :model="interfaceSettings" label-width="120px">
                                <el-form-item label="主题模式">
                                    <el-radio-group v-model="interfaceSettings.theme">
                                        <el-radio label="light">浅色主题</el-radio>
                                        <el-radio label="dark">深色主题</el-radio>
                                        <el-radio label="auto">跟随系统</el-radio>
                                    </el-radio-group>
                                </el-form-item>

                                <el-form-item label="文件显示方式">
                                    <el-radio-group v-model="interfaceSettings.fileDisplay">
                                        <el-radio label="list">列表模式</el-radio>
                                        <el-radio label="grid">网格模式</el-radio>
                                    </el-radio-group>
                                </el-form-item>

                                <el-form-item label="文件大小单位">
                                    <el-select v-model="interfaceSettings.sizeUnit">
                                        <el-option label="自动" value="auto"/>
                                        <el-option label="字节" value="B"/>
                                        <el-option label="KB" value="KB"/>
                                        <el-option label="MB" value="MB"/>
                                        <el-option label="GB" value="GB"/>
                                    </el-select>
                                </el-form-item>

                                <el-form-item label="语言设置">
                                    <el-select v-model="interfaceSettings.language">
                                        <el-option label="简体中文" value="zh-CN"/>
                                        <el-option label="English" value="en-US"/>
                                    </el-select>
                                </el-form-item>
                            </el-form>
                        </div>
                    </el-tab-pane>

                    <!-- 隐私设置 -->
                    <el-tab-pane label="隐私设置" name="privacy">
                        <div class="tab-content">
                            <el-form :model="privacySettings" label-width="120px">
                                <el-form-item label="文件可见性">
                                    <el-radio-group v-model="privacySettings.fileVisibility">
                                        <el-radio label="public">公开</el-radio>
                                        <el-radio label="private">私有</el-radio>
                                        <el-radio label="shared">仅共享</el-radio>
                                    </el-radio-group>
                                </el-form-item>

                                <el-form-item label="共享链接有效期">
                                    <el-select v-model="privacySettings.linkExpiry">
                                        <el-option label="永久有效" value="never"/>
                                        <el-option label="1天" value="1d"/>
                                        <el-option label="7天" value="7d"/>
                                        <el-option label="30天" value="30d"/>
                                        <el-option label="自定义" value="custom"/>
                                    </el-select>
                                </el-form-item>

                                <el-form-item label="访问权限控制">
                                    <el-checkbox v-model="privacySettings.requirePassword">需要密码访问</el-checkbox>
                                    <el-checkbox v-model="privacySettings.allowDownload">允许下载</el-checkbox>
                                    <el-checkbox v-model="privacySettings.allowPreview">允许预览</el-checkbox>
                                </el-form-item>

                                <el-form-item label="数据加密">
                                    <el-checkbox v-model="privacySettings.enableEncryption">启用文件加密</el-checkbox>
                                </el-form-item>
                            </el-form>
                        </div>
                    </el-tab-pane>

                    <!-- 通知设置 -->
                    <el-tab-pane label="通知设置" name="notification">
                        <div class="tab-content">
                            <el-form :model="notificationSettings" label-width="120px">
                                <el-form-item label="邮件通知">
                                    <el-checkbox v-model="notificationSettings.emailLogin">登录通知</el-checkbox>
                                    <el-checkbox v-model="notificationSettings.emailShare">文件共享通知</el-checkbox>
                                    <el-checkbox v-model="notificationSettings.emailStorage">存储空间警告</el-checkbox>
                                    <el-checkbox v-model="notificationSettings.emailUpdate">系统更新通知</el-checkbox>
                                </el-form-item>

                                <el-form-item label="应用内通知">
                                    <el-checkbox v-model="notificationSettings.appPush">消息推送</el-checkbox>
                                    <el-checkbox v-model="notificationSettings.appSound">通知声音</el-checkbox>
                                    <el-checkbox v-model="notificationSettings.appBanner">横幅通知</el-checkbox>
                                </el-form-item>

                                <el-form-item label="通知频率">
                                    <el-radio-group v-model="notificationSettings.frequency">
                                        <el-radio label="immediate">立即</el-radio>
                                        <el-radio label="hourly">每小时</el-radio>
                                        <el-radio label="daily">每天</el-radio>
                                        <el-radio label="weekly">每周</el-radio>
                                    </el-radio-group>
                                </el-form-item>
                            </el-form>
                        </div>
                    </el-tab-pane>

                    <!-- 同步设置 -->
                    <el-tab-pane label="同步设置" name="sync">
                        <div class="tab-content">
                            <el-form :model="syncSettings" label-width="120px">
                                <el-form-item label="自动同步">
                                    <el-switch v-model="syncSettings.autoSync"/>
                                </el-form-item>

                                <el-form-item v-if="syncSettings.autoSync" label="同步频率">
                                    <el-select v-model="syncSettings.syncFrequency">
                                        <el-option label="实时同步" value="realtime"/>
                                        <el-option label="每5分钟" value="5min"/>
                                        <el-option label="每15分钟" value="15min"/>
                                        <el-option label="每小时" value="1hour"/>
                                    </el-select>
                                </el-form-item>

                                <el-form-item label="冲突处理">
                                    <el-radio-group v-model="syncSettings.conflictResolution">
                                        <el-radio label="keepLocal">保留本地版本</el-radio>
                                        <el-radio label="keepRemote">保留云端版本</el-radio>
                                        <el-radio label="askUser">询问用户</el-radio>
                                    </el-radio-group>
                                </el-form-item>

                                <el-form-item label="同步内容">
                                    <el-checkbox v-model="syncSettings.syncFiles">同步文件</el-checkbox>
                                    <el-checkbox v-model="syncSettings.syncFolders">同步文件夹</el-checkbox>
                                    <el-checkbox v-model="syncSettings.syncSettings">同步设置</el-checkbox>
                                </el-form-item>
                            </el-form>
                        </div>
                    </el-tab-pane>

                    <!-- 高级设置 -->
                    <el-tab-pane label="高级设置" name="advanced">
                        <div class="tab-content">
                            <el-form :model="advancedSettings" label-width="120px">
                                <el-form-item label="缓存设置">
                                    <el-input-number
                                            v-model="advancedSettings.cacheSize"
                                            :max="10000"
                                            :min="100"
                                            :step="100"
                                    />
                                    <span class="unit">MB</span>
                                </el-form-item>

                                <el-form-item label="下载线程数">
                                    <el-input-number
                                            v-model="advancedSettings.downloadThreads"
                                            :max="10"
                                            :min="1"
                                            :step="1"
                                    />
                                </el-form-item>

                                <el-form-item label="上传限制">
                                    <el-input-number
                                            v-model="advancedSettings.uploadLimit"
                                            :max="100"
                                            :min="1"
                                            :step="1"
                                    />
                                    <span class="unit">MB</span>
                                </el-form-item>

                                <el-form-item label="调试模式">
                                    <el-switch v-model="advancedSettings.debugMode"/>
                                </el-form-item>

                                <el-form-item label="数据导出">
                                    <el-button type="primary" @click="exportData">
                                        导出用户数据
                                    </el-button>
                                </el-form-item>

                                <el-form-item label="清除缓存">
                                    <el-button type="warning" @click="clearCache">
                                        清除缓存
                                    </el-button>
                                </el-form-item>
                            </el-form>
                        </div>
                    </el-tab-pane>
                </el-tabs>

                <!-- 保存按钮 -->
                <div class="settings-actions">
                    <el-button @click="resetSettings">重置</el-button>
                    <el-button :loading="saving" type="primary" @click="saveSettings">
                        保存设置
                    </el-button>
                </div>
            </el-card>
        </div>
    </div>
</template>

<script setup>
import {onMounted, reactive, ref} from 'vue'
import {ElMessage, ElMessageBox} from 'element-plus'

// 响应式数据
const activeTab = ref('interface')
const saving = ref(false)

// 界面设置
const interfaceSettings = reactive({
    theme: 'light',
    fileDisplay: 'list',
    sizeUnit: 'auto',
    language: 'zh-CN'
})

// 隐私设置
const privacySettings = reactive({
    fileVisibility: 'private',
    linkExpiry: 'never',
    requirePassword: false,
    allowDownload: true,
    allowPreview: true,
    enableEncryption: false
})

// 通知设置
const notificationSettings = reactive({
    emailLogin: true,
    emailShare: true,
    emailStorage: true,
    emailUpdate: false,
    appPush: true,
    appSound: true,
    appBanner: true,
    frequency: 'immediate'
})

// 同步设置
const syncSettings = reactive({
    autoSync: true,
    syncFrequency: 'realtime',
    conflictResolution: 'askUser',
    syncFiles: true,
    syncFolders: true,
    syncSettings: true
})

// 高级设置
const advancedSettings = reactive({
    cacheSize: 500,
    downloadThreads: 3,
    uploadLimit: 10,
    debugMode: false
})

// 方法
const loadSettings = () => {
    // 从本地存储加载设置
    const savedSettings = localStorage.getItem('userSettings')
    if (savedSettings) {
        try {
            const settings = JSON.parse(savedSettings)
            Object.assign(interfaceSettings, settings.interface || {})
            Object.assign(privacySettings, settings.privacy || {})
            Object.assign(notificationSettings, settings.notification || {})
            Object.assign(syncSettings, settings.sync || {})
            Object.assign(advancedSettings, settings.advanced || {})
        } catch (error) {
            console.error('加载设置失败:', error)
        }
    }
}

const saveSettings = async () => {
    try {
        saving.value = true

        // 保存到本地存储
        const settings = {
            interface: interfaceSettings,
            privacy: privacySettings,
            notification: notificationSettings,
            sync: syncSettings,
            advanced: advancedSettings
        }
        localStorage.setItem('userSettings', JSON.stringify(settings))

        // 这里可以添加保存到服务器的逻辑
        // await updateSettings(settings)

        ElMessage.success('设置保存成功')
    } catch (error) {
        ElMessage.error('设置保存失败')
    } finally {
        saving.value = false
    }
}

const resetSettings = async () => {
    try {
        await ElMessageBox.confirm(
            '确定要重置所有设置吗？此操作不可撤销。',
            '确认重置',
            {
                confirmButtonText: '确定',
                cancelButtonText: '取消',
                type: 'warning'
            }
        )

        // 重置所有设置
        Object.assign(interfaceSettings, {
            theme: 'light',
            fileDisplay: 'list',
            sizeUnit: 'auto',
            language: 'zh-CN'
        })

        Object.assign(privacySettings, {
            fileVisibility: 'private',
            linkExpiry: 'never',
            requirePassword: false,
            allowDownload: true,
            allowPreview: true,
            enableEncryption: false
        })

        Object.assign(notificationSettings, {
            emailLogin: true,
            emailShare: true,
            emailStorage: true,
            emailUpdate: false,
            appPush: true,
            appSound: true,
            appBanner: true,
            frequency: 'immediate'
        })

        Object.assign(syncSettings, {
            autoSync: true,
            syncFrequency: 'realtime',
            conflictResolution: 'askUser',
            syncFiles: true,
            syncFolders: true,
            syncSettings: true
        })

        Object.assign(advancedSettings, {
            cacheSize: 500,
            downloadThreads: 3,
            uploadLimit: 10,
            debugMode: false
        })

        localStorage.removeItem('userSettings')
        ElMessage.success('设置已重置')
    } catch (error) {
        // 用户取消操作
    }
}

const exportData = () => {
    ElMessage.info('数据导出功能开发中...')
}

const clearCache = async () => {
    try {
        await ElMessageBox.confirm(
            '确定要清除缓存吗？这可能会影响应用性能。',
            '确认清除',
            {
                confirmButtonText: '确定',
                cancelButtonText: '取消',
                type: 'warning'
            }
        )

        // 清除缓存的逻辑
        localStorage.removeItem('fileCache')
        sessionStorage.clear()

        ElMessage.success('缓存清除成功')
    } catch (error) {
        // 用户取消操作
    }
}

// 生命周期
onMounted(() => {
    loadSettings()
})
</script>

<style scoped>
.settings-page {
    padding: 20px;
    background: var(--bg-color);
    min-height: 100vh;
}

.settings-container {
    max-width: 1000px;
    margin: 0 auto;
}

.settings-card {
    background: var(--card-bg);
}

.card-header {
    display: flex;
    align-items: center;
    gap: 8px;
    font-weight: 600;
}

.settings-tabs {
    margin-top: 20px;
}

.tab-content {
    padding: 20px 0;
}

.unit {
    margin-left: 8px;
    color: var(--text-secondary);
}

.settings-actions {
    display: flex;
    justify-content: flex-end;
    gap: 12px;
    margin-top: 30px;
    padding-top: 20px;
    border-top: 1px solid var(--border-color);
}

/* 响应式设计 */
@media (max-width: 768px) {
    .settings-container {
        max-width: 100%;
    }

    .tab-content {
        padding: 10px 0;
    }
}
</style>
  