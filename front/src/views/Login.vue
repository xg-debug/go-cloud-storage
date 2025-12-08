<template>
    <div class="login-container">
        <!-- 背景装饰元素 -->
        <div class="decoration-circle circle-1"></div>
        <div class="decoration-circle circle-2"></div>

        <!-- 主卡片 -->
        <el-card class="auth-card" shadow="hover">
            <div class="brand-header">
                <img src="@/assets/logo.png" alt="CloudDisk" class="logo">
                <h1>{{ activeTab === 'login' ? '欢迎回来' : '创建新账号' }}</h1>
                <p class="subtitle">{{ activeTab === 'login' ? '安全访问您的云存储空间' : '立即加入CloudDisk' }}</p>
            </div>

            <!-- 标签页切换 -->
            <el-tabs v-model="activeTab" stretch class="auth-tabs">
                <el-tab-pane label="登录" name="login">
                    <!-- 绑定 loginFormRef -->
                    <el-form ref="loginFormRef" :model="loginForm" :rules="loginRules" @keyup.enter="handleLogin">
                        <el-form-item prop="account">
                            <!-- 添加 @input 事件，清除当前字段的验证状态 -->
                            <el-input
                                    v-model="loginForm.account"
                                    placeholder="邮箱/手机号"
                                    prefix-icon="User"
                                    clearable
                                    @input="clearLoginValidate('account')"
                            />
                        </el-form-item>

                        <el-form-item prop="password">
                            <!-- 添加 @input 事件，清除当前字段的验证状态 -->
                            <el-input
                                    v-model="loginForm.password"
                                    type="password"
                                    placeholder="密码"
                                    prefix-icon="Lock"
                                    show-password
                                    @input="clearLoginValidate('password')"
                            />
                        </el-form-item>

                        <div class="flex-bar">
                            <el-checkbox v-model="rememberMe">记住我</el-checkbox>
                            <el-link type="primary" :underline="false">忘记密码?</el-link>
                        </div>

                        <el-button type="primary" class="auth-btn" :loading="loading" @click="handleLogin">
                            登录
                        </el-button>
                    </el-form>
                </el-tab-pane>

                <el-tab-pane label="注册" name="register">
                    <!-- 绑定 registerFormRef -->
                    <el-form ref="registerFormRef" :model="registerForm" :rules="registerRules"
                             @keyup.enter="handleRegister">
                        <el-form-item prop="email">
                            <!-- 添加 @input 事件，清除当前字段的验证状态 -->
                            <el-input
                                    v-model="registerForm.email"
                                    placeholder="邮箱"
                                    prefix-icon="Message"
                                    clearable
                                    @input="clearRegisterValidate('email')"
                            />
                        </el-form-item>

                        <el-form-item prop="password">
                            <!-- 添加 @input 事件，清除当前字段的验证状态 -->
                            <el-input
                                    v-model="registerForm.password"
                                    type="password"
                                    placeholder="密码"
                                    show-password
                                    prefix-icon="Lock"
                                    @input="clearRegisterValidate('password')"
                            />
                        </el-form-item>

                        <el-form-item prop="password_confirm">
                            <!-- 添加 @input 事件，清除当前字段的验证状态 -->
                            <el-input
                                    v-model="registerForm.password_confirm"
                                    type="password"
                                    placeholder="确认密码"
                                    show-password
                                    prefix-icon="Lock"
                                    @input="clearRegisterValidate('password_confirm')"
                            />
                        </el-form-item>

                        <el-button type="primary" class="auth-btn" :loading="loading" @click="handleRegister">
                            注册
                        </el-button>
                    </el-form>
                </el-tab-pane>
            </el-tabs>

            <!-- 第三方登录 -->
            <div class="third-party-login">
                <el-divider>{{ activeTab === 'login' ? '或通过以下方式登录' : '其他注册方式' }}</el-divider>
                <div class="oauth-icons">
                    <el-icon class="iconfont icon-weixindenglu"></el-icon>
                    <el-icon class="iconfont icon-icon_alipay"></el-icon>
                </div>
            </div>
        </el-card>

        <!-- 底部版权 -->
        <footer class="login-footer">
            <span>© 2025 CloudDisk 云存储服务</span>
            <el-divider direction="vertical"/>
            <el-link type="info" :underline="false">用户协议</el-link>
            <el-divider direction="vertical"/>
            <el-link type="info" :underline="false">隐私政策</el-link>
        </footer>
    </div>
</template>

<script setup>
import {reactive, ref} from 'vue'
import {useRouter} from 'vue-router'
import {ElMessage} from 'element-plus'
import {login, register, storeToken} from '@/api/auth'
import {useStore} from 'vuex'


const router = useRouter()
const activeTab = ref('login')
const loading = ref(false)
const rememberMe = ref(false)
const store = useStore()

// 用 reactive，保证 v-model 与 el-form 绑定一致
const loginForm = reactive({
    account: '',
    password: '',
})

const registerForm = reactive({
    email: '',
    password: '',
    password_confirm: ''
})

// 表单引用（可选，用于手动验证）
const loginFormRef = ref()
const registerFormRef = ref()

// --- 新增：清除验证状态的函数 ---
const clearLoginValidate = (prop) => {
    if (loginFormRef.value) {
        loginFormRef.value.clearValidate(prop);
    }
}

const clearRegisterValidate = (prop) => {
    if (registerFormRef.value) {
        registerFormRef.value.clearValidate(prop);
    }
}

// 密码强度验证
const checkPasswordStrength = (_, value, callback) => {
    if (!value) return callback(new Error('请输入密码'))
    if (value.length < 6) return callback(new Error('至少6个字符'))
    const hasUpper = /[A-Z]/.test(value)
    const hasLower = /[a-z]/.test(value)
    const hasNumber = /\d/.test(value)
    if (!(hasUpper && hasLower && hasNumber)) {
        return callback(new Error('需包含大小写字母和数字'))
    }
    callback()
}

// 确认密码验证
const validateConfirmPassword = (_, value, callback) => {
    // 确保与注册表单中的 password 字段进行比对
    if (value !== registerForm.password) {
        callback(new Error('两次输入的密码不一致'))
    } else {
        callback()
    }
}

// 登录验证规则
const loginRules = {
    account: [
        {required: true, message: '请输入邮箱或手机号', trigger: 'change'},
        {
            validator: (_, value, callback) => {
                const isEmail = /^[\w-]+(\.[\w-]+)*@[\w-]+(\.[\w-]+)+$/.test(value)
                const isPhone = /^1[3-9]\d{9}$/.test(value)
                if (!isEmail && !isPhone) {
                    callback(new Error('请输入有效的邮箱或手机号'))
                } else {
                    callback()
                }
            },
            trigger: 'change'
        }
    ],
    password: [
        {required: true, message: '请输入密码', trigger: 'change'},
        {min: 6, max: 20, message: '长度在6到20个字符', trigger: 'change'}
    ]
}

// 注册验证规则
const registerRules = {
    email: [
        {required: true, message: '请输入邮箱', trigger: 'change'},
        {type: 'email', message: '邮箱格式不正确', trigger: 'change'}
    ],
    password: [
        {required: true, message: '请输入密码', trigger: 'change'},
        {validator: checkPasswordStrength, trigger: 'change'}
    ],
    // 修复：将错误的 confirmPassword 字段改为 password_confirm，与 prop="password_confirm" 保持一致
    password_confirm: [
        {required: true, message: '请确认密码', trigger: 'change'},
        {validator: validateConfirmPassword, trigger: 'change'}
    ]
}

// 登录处理
const handleLogin = async () => {
    // 首先进行表单验证
    const isValid = await loginFormRef.value.validate().catch(() => false);
    if (!isValid) return;

    try {
        loading.value = true
        const res = await login(loginForm)
        storeToken(res.token, rememberMe.value)

        // 把token和userInfo存入到 Vuex
        store.commit('setToken', res.token)
        store.commit('setUserInfo', res.user_info)
        ElMessage.success({
            message: '登录成功',
            duration: 2000
        })
        router.push('/')
    } catch (error) {
        ElMessage.error(error.message || '登录失败')
    } finally {
        loading.value = false
    }
}

// 注册处理
const handleRegister = async () => {
    // 首先进行表单验证
    const isValid = await registerFormRef.value.validate().catch(() => false);
    if (!isValid) return;

    try {
        loading.value = true
        await register(registerForm)
        ElMessage.success('注册成功，请登录')
        activeTab.value = 'login'
        loginForm.account = registerForm.email
    } catch (error) {
        ElMessage.error(error.message || '注册失败')
    } finally {
        loading.value = false
    }
}
</script>

<style scoped>
.login-container {
    min-height: 100vh;
    display: flex;
    flex-direction: column;
    justify-content: center;
    align-items: center;
    background: linear-gradient(135deg, #667eea 0%, #764ba2 50%, #f093fb 100%);
    position: relative;
    overflow: hidden;
    padding: 20px;
}

/* 动态背景装饰 */
.decoration-circle {
    position: absolute;
    border-radius: 50%;
    background: rgba(255, 255, 255, 0.1);
    backdrop-filter: blur(10px);
    animation: float 15s ease-in-out infinite;
}

.decoration-circle.circle-1 {
    width: 400px;
    height: 400px;
    top: -100px;
    left: -100px;
    animation-delay: 0s;
}

.decoration-circle.circle-2 {
    width: 300px;
    height: 300px;
    bottom: -80px;
    right: -80px;
    animation-delay: -5s;
}

@keyframes float {
    0%, 100% {
        transform: translate(0, 0) scale(1);
    }
    33% {
        transform: translate(20px, -20px) scale(1.05);
    }
    66% {
        transform: translate(-15px, 15px) scale(0.95);
    }
}

/* 登录卡片 */
.auth-card {
    width: 100%;
    min-height: 520px;
    max-width: 440px;
    border-radius: var(--radius-xl);
    border: 1px solid rgba(255, 255, 255, 0.3);
    z-index: 1;
    backdrop-filter: blur(20px);
    -webkit-backdrop-filter: blur(20px);
    background: rgba(255, 255, 255, 0.9);
    box-shadow: 0 25px 50px -12px rgba(0, 0, 0, 0.25);
}

.auth-card :deep(.el-card__body) {
    padding: 48px 40px;
}

/* 品牌头部 */
.brand-header {
    text-align: center;
    margin-bottom: 36px;
}

.brand-header .logo {
    height: 56px;
    margin-bottom: 20px;
    filter: drop-shadow(0 4px 12px rgba(99, 102, 241, 0.3));
}

.brand-header h1 {
    font-size: 26px;
    font-weight: 700;
    background: var(--primary-gradient);
    -webkit-background-clip: text;
    -webkit-text-fill-color: transparent;
    background-clip: text;
    margin: 0 0 8px;
}

.brand-header .subtitle {
    font-size: 14px;
    color: var(--text-tertiary);
    margin: 0;
}

/* 标签页 */
.auth-tabs {
    margin-top: 24px;
}

.auth-tabs :deep(.el-tabs__nav-wrap)::after {
    display: none;
}

.auth-tabs :deep(.el-tabs__nav) {
    width: 100%;
    display: flex;
    background: var(--bg-secondary);
    border-radius: var(--radius-md);
    padding: 0;
}

.auth-tabs :deep(.el-tabs__item) {
    flex: 1;
    text-align: center;
    padding: 0;
    height: 40px;
    line-height: 40px;
    border-radius: var(--radius-md);
    font-weight: 500;
    color: var(--text-secondary);
    transition: all var(--transition-fast);
}

.auth-tabs :deep(.el-tabs__item.is-active) {
    background: white;
    color: var(--primary-color);
    box-shadow: var(--shadow-sm);
}

.auth-tabs :deep(.el-tabs__active-bar) {
    display: none;
}

/* 表单样式 */
.auth-tabs :deep(.el-form-item) {
    margin-bottom: 20px;
}

.auth-tabs :deep(.el-input__wrapper) {
    border-radius: var(--radius-lg);
    padding: 4px 16px;
    height: 48px;
    background: var(--bg-secondary);
    box-shadow: none;
    border: 1px solid transparent;
    transition: all var(--transition-fast);
}

.auth-tabs :deep(.el-input__wrapper:hover) {
    background: var(--bg-tertiary);
}

.auth-tabs :deep(.el-input__wrapper.is-focus) {
    background: white;
//border-color: var(--primary-color); //box-shadow: 0 0 0 3px rgba(99, 102, 241, 0.1);
}

.flex-bar {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 24px;
}

.flex-bar :deep(.el-checkbox__label) {
    color: var(--text-secondary);
    font-size: 13px;
}

.flex-bar :deep(.el-link) {
    font-size: 13px;
}

/* 登录按钮 */
.auth-btn {
    width: 100%;
    height: 50px;
    font-size: 16px;
    font-weight: 600;
    margin-top: 8px;
    border-radius: var(--radius-lg);
    background: var(--primary-gradient) !important;
    border: none !important;
    transition: all var(--transition-normal);
}

.auth-btn:hover {
    transform: translateY(-2px);
}

.auth-btn:active {
    transform: translateY(0);
}

/* 第三方登录 */
.third-party-login {
    margin-top: 32px;
}

.third-party-login :deep(.el-divider__text) {
    color: var(--text-tertiary);
    font-size: 13px;
    background: rgba(255, 255, 255, 0.9);
}

.oauth-icons {
    display: flex;
    justify-content: center;
    gap: 16px;
    margin-top: 20px;
}

.oauth-icons .el-icon {
    width: 48px;
    height: 48px;
    font-size: 22px;
    color: white;
    cursor: pointer;
    transition: all var(--transition-normal);
    border-radius: var(--radius-lg);
    display: flex;
    align-items: center;
    justify-content: center;
}

.oauth-icons .el-icon:hover {
    transform: translateY(-3px);
    box-shadow: 0 8px 16px -4px rgba(0, 0, 0, 0.2);
}

.icon-weixindenglu {
    background: linear-gradient(135deg, #07C160 0%, #00a854 100%) !important;
}

.icon-icon_alipay {
    background: linear-gradient(135deg, #1677FF 0%, #0958d9 100%) !important;
}

/* 页脚 */
.login-footer {
    margin-top: 32px;
    text-align: center;
    font-size: 12px;
    color: rgba(255, 255, 255, 0.7);
    z-index: 1;
}

.login-footer :deep(.el-divider--vertical) {
    margin: 0 12px;
    height: 12px;
    background: rgba(255, 255, 255, 0.3);
}

.login-footer :deep(.el-link) {
    color: rgba(255, 255, 255, 0.8);
    font-size: 12px;
}

.login-footer :deep(.el-link:hover) {
    color: white;
}

/* 响应式 */
@media (max-width: 768px) {
    .auth-card :deep(.el-card__body) {
        padding: 32px 24px !important;
    }

    .brand-header h1 {
        font-size: 22px;
    }

    .decoration-circle.circle-1 {
        width: 250px;
        height: 250px;
    }

    .decoration-circle.circle-2 {
        width: 180px;
        height: 180px;
    }
}
</style>