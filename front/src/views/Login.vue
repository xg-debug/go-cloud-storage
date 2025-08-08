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
            <el-form ref="loginFormRef" :model="loginForm" :rules="loginRules" @keyup.enter="handleLogin">
              <el-form-item prop="account">
                <el-input v-model="loginForm.account" placeholder="邮箱/手机号" prefix-icon="User" clearable />
              </el-form-item>
  
              <el-form-item prop="password">
                <el-input v-model="loginForm.password" type="password" placeholder="密码" prefix-icon="Lock" show-password />
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
            <el-form ref="registerFormRef" :model="registerForm" :rules="registerRules" @keyup.enter="handleRegister">
              <el-form-item prop="email">
                <el-input v-model="registerForm.email" placeholder="邮箱" prefix-icon="Message" clearable />
              </el-form-item>
  
              <el-form-item prop="password">
                <el-input v-model="registerForm.password" type="password" placeholder="密码" show-password />
              </el-form-item>
  
              <el-form-item prop="password_confirm">
                <el-input v-model="registerForm.password_confirm" type="password" placeholder="确认密码" show-password />
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
        <el-divider direction="vertical" />
        <el-link type="info" :underline="false">用户协议</el-link>
        <el-divider direction="vertical" />
        <el-link type="info" :underline="false">隐私政策</el-link>
      </footer>
    </div>
  </template>
  
  <script setup>
  import { ref, reactive } from 'vue'
  import { useRouter } from 'vue-router'
  import { ElMessage } from 'element-plus'
  import { login, register, storeToken } from '@/api/auth'
  import { useStore } from 'vuex'
  
  
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
    if (value !== registerForm.password) {
      callback(new Error('两次输入的密码不一致'))
    } else {
      callback()
    }
  }
  
  // 登录验证规则
  const loginRules = {
    account: [
      { required: true, message: '请输入邮箱或手机号', trigger: 'blur' },
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
        trigger: 'blur'
      }
    ],
    password: [
      { required: true, message: '请输入密码', trigger: 'blur' },
      { min: 6, max: 20, message: '长度在6到20个字符', trigger: 'blur' }
    ]
  }
  
  // 注册验证规则
  const registerRules = {
    email: [
      { required: true, message: '请输入邮箱', trigger: 'blur' },
      { type: 'email', message: '邮箱格式不正确', trigger: 'blur' }
    ],
    password: [
      { required: true, message: '请输入密码', trigger: 'blur' },
      { validator: checkPasswordStrength, trigger: 'blur' }
    ],
    confirmPassword: [
      { required: true, message: '请确认密码', trigger: 'blur' },
      { validator: validateConfirmPassword, trigger: 'blur' }
    ]
  }
  
  // 登录处理
  const handleLogin = async () => {
    try {
      loading.value = true
      const res = await login(loginForm)
      storeToken(res.token, rememberMe.value)

      // 把token和userInfo存入到 Vuex
      store.commit('setToken', res.token)
      store.commit('setUserInfo', res.user_info)
      ElMessage.success({
        message: '登录成功',
        duration: 2000 // 单位：毫秒
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
    try {
      loading.value = true
      await register(registerForm)
      ElMessage.success('注册成功')
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
  /* 样式不变，保持一致 */
  .login-container {
    min-height: 100vh;
    display: flex;
    flex-direction: column;
    justify-content: center;
    align-items: center;
    background: linear-gradient(135deg, #f5f7fa 0%, #c3cfe2 100%);
    position: relative;
    overflow: hidden;
    padding: 20px;
  }
  
  .decoration-circle {
    position: absolute;
    border-radius: 50%;
    background: rgba(255, 255, 255, 0.3);
    backdrop-filter: blur(5px);
  }
  
  .decoration-circle.circle-1 {
    width: 300px;
    height: 300px;
    top: -50px;
    left: -50px;
  }
  
  .decoration-circle.circle-2 {
    width: 200px;
    height: 200px;
    bottom: -30px;
    right: -30px;
  }
  
  .auth-card {
    width: 100%;
    min-height: 560px;
    max-width: 420px;
    border-radius: 12px;
    border: none;
    z-index: 1;
    backdrop-filter: blur(8px);
    background-color: rgba(255, 255, 255, 0.85);
  }
  
  .auth-card :deep(.el-card__body) {
    padding: 40px;
  }
  
  .brand-header {
    text-align: center;
    margin-bottom: 30px;
  }
  
  .brand-header .logo {
    height: 50px;
    margin-bottom: 15px;
  }
  
  .brand-header h1 {
    font-size: 24px;
    color: #333;
    margin: 0 0 8px;
  }
  
  .brand-header .subtitle {
    font-size: 14px;
    color: #999;
    margin: 0;
  }
  
  .auth-tabs {
    margin-top: 20px;
  }
  
  .auth-tabs :deep(.el-tabs__nav-wrap)::after {
    display: none;
  }
  
  .auth-tabs :deep(.el-tabs__nav) {
    width: 100%;
    display: flex;
  }
  
  .auth-tabs :deep(.el-tabs__item) {
    flex: 1;
    text-align: center;
    padding: 0;
  }
  
  .flex-bar {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 20px;
  }
  
  .auth-btn {
    width: 100%;
    height: 48px;
    font-size: 16px;
    margin-top: 10px;
  }
  
  .third-party-login {
    margin-top: 30px;
  }
  
  .oauth-icons {
    display: flex;
    justify-content: center;
    gap: 20px;
  }
  
  .oauth-icons .el-icon {
    font-size: 24px;
    color: #666;
    cursor: pointer;
    transition: all 0.3s;
  }
  
  .oauth-icons .el-icon:hover {
    color: #409eff;
    transform: translateY(-3px);
  }
  
  .login-footer {
    margin-top: 30px;
    text-align: center;
    font-size: 12px;
    color: #999;
    z-index: 1;
  }
  
  .login-footer .el-divider--vertical {
    margin: 0 10px;
    height: 12px;
  }
  
  .icon-weixindenglu {
    color: #07C160 !important;
  }
  
  .icon-icon_alipay {
    color: #1677FF !important;
  }
  
  @media (max-width: 768px) {
    .auth-card :deep(.el-card__body) {
      padding: 30px 20px !important;
    }
  }
  </style>
  