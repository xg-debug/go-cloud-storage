<template>
  <div class="login-page">
    <!-- Left: brand -->
    <section class="login-brand">
      <div class="brand-top">
        <div class="brand-logo">
          <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round"><path d="M17.5 19H9a7 7 0 1 1 6.71-9h1.79a4.5 4.5 0 1 1 0 9Z"/></svg>
        </div>
        <div>
          <div class="brand-name">CloudBox</div>
          <div class="brand-sub">安全、便捷的个人云盘</div>
        </div>
      </div>
      <div class="brand-hero">
        <h1>存储你的</h1>
        <h1 class="hero-gradient">每一个精彩瞬间</h1>
        <p>上传、预览、分享和管理文件，一个工作台全搞定。</p>
      </div>
      <div class="brand-features">
        <div class="feat-item"><el-icon><Folder /></el-icon><div><strong>文件管理</strong><span>多种视图，便捷整理</span></div></div>
        <div class="feat-item"><el-icon><Share /></el-icon><div><strong>安全分享</strong><span>提取码保护，链接分享</span></div></div>
        <div class="feat-item"><el-icon><Clock /></el-icon><div><strong>回收站</strong><span>7天内可恢复已删除文件</span></div></div>
      </div>
      <footer class="brand-footer">
        <span>&copy; 2026 CloudBox</span>
        <span>用户协议</span>
        <span>隐私政策</span>
      </footer>
    </section>

    <!-- Right: form -->
    <section class="login-form-section">
      <div class="login-card">
        <div class="card-head">
          <h2>{{ tab === 'login' ? '欢迎回来' : '创建账号' }}</h2>
          <p>{{ tab === 'login' ? '登录你的 CloudBox 账户' : '开通 10GB 免费空间' }}</p>
        </div>

        <el-tabs v-model="tab" class="login-tabs" stretch>
          <el-tab-pane label="登录" name="login">
            <el-form ref="loginFormRef" :model="loginForm" :rules="loginRules" @keyup.enter="handleLogin">
              <el-form-item prop="account">
                <el-input v-model="loginForm.account" placeholder="邮箱或手机号" :prefix-icon="User" size="large" />
              </el-form-item>
              <el-form-item prop="password">
                <el-input v-model="loginForm.password" type="password" placeholder="密码" :prefix-icon="Lock" size="large" show-password />
              </el-form-item>
              <div class="form-row">
                <el-checkbox v-model="rememberMe">记住我</el-checkbox>
              </div>
              <el-button type="primary" class="submit-btn" :loading="loading" @click="handleLogin" size="large">登录</el-button>
            </el-form>
          </el-tab-pane>

          <el-tab-pane label="注册" name="register">
            <el-form ref="registerFormRef" :model="registerForm" :rules="registerRules" @keyup.enter="handleRegister">
              <el-form-item prop="email">
                <el-input v-model="registerForm.email" placeholder="邮箱" :prefix-icon="Message" size="large" />
              </el-form-item>
              <el-form-item prop="password">
                <el-input v-model="registerForm.password" type="password" placeholder="密码（大小写字母+数字，6位以上）" :prefix-icon="Lock" size="large" show-password />
              </el-form-item>
              <div class="password-strength" v-if="registerForm.password">
                <span v-for="i in 3" :key="i" :class="{ active: passwordScore >= i }"></span>
                <em>{{ passwordText }}</em>
              </div>
              <el-form-item prop="password_confirm">
                <el-input v-model="registerForm.password_confirm" type="password" placeholder="确认密码" :prefix-icon="Lock" size="large" show-password />
              </el-form-item>
              <el-checkbox v-model="termsAccepted">我已阅读并同意用户协议与隐私政策</el-checkbox>
              <el-button type="primary" class="submit-btn" :loading="loading" :disabled="!termsAccepted" @click="handleRegister" size="large">创建账号</el-button>
            </el-form>
          </el-tab-pane>
        </el-tabs>
      </div>
    </section>
  </div>
</template>

<script setup>
import { computed, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { login, register, storeToken } from '@/api/auth'
import { useStore } from 'vuex'

const router = useRouter()
const store = useStore()
const tab = ref('login')
const loading = ref(false)
const rememberMe = ref(false)
const termsAccepted = ref(true)

const loginForm = reactive({ account: '', password: '' })
const registerForm = reactive({ email: '', password: '', password_confirm: '' })
const loginFormRef = ref()
const registerFormRef = ref()

const passwordScore = computed(() => {
  const v = registerForm.password || ''
  let s = 0
  if (v.length >= 6) s++
  if (/[a-z]/.test(v) && /[A-Z]/.test(v)) s++
  if (/\d/.test(v)) s++
  return s
})
const passwordText = computed(() => ['弱', '中', '强', '安全'][passwordScore.value])

const loginRules = {
  account: [{ required: true, message: '请输入邮箱或手机号', trigger: 'blur' }],
  password: [{ required: true, message: '请输入密码', trigger: 'blur' }]
}

const validatePassword = (_, v, cb) => {
  if (!v) return cb(new Error('请输入密码'))
  if (v.length < 6) return cb(new Error('至少6个字符'))
  if (!/[a-z]/.test(v) || !/[A-Z]/.test(v) || !/\d/.test(v)) return cb(new Error('需包含大小写字母和数字'))
  cb()
}

const registerRules = {
  email: [{ required: true, message: '请输入邮箱', trigger: 'blur' }, { type: 'email', message: '格式不正确', trigger: 'blur' }],
  password: [{ required: true, validator: validatePassword, trigger: 'blur' }],
  password_confirm: [
    { required: true, message: '请确认密码', trigger: 'blur' },
    { validator: (_, v, cb) => v !== registerForm.password ? cb(new Error('两次密码不一致')) : cb(), trigger: 'blur' }
  ]
}

async function handleLogin() {
  const ok = await loginFormRef.value.validate().catch(() => false)
  if (!ok) return
  loading.value = true
  try {
    const res = await login(loginForm)
    storeToken(res.token, rememberMe.value)
    store.commit('setToken', res.token)
    store.commit('setUserInfo', res.user_info)
    ElMessage.success('登录成功')
    router.push('/')
  } catch {} finally { loading.value = false }
}

async function handleRegister() {
  if (!termsAccepted.value) { ElMessage.warning('请同意用户协议'); return }
  const ok = await registerFormRef.value.validate().catch(() => false)
  if (!ok) return
  loading.value = true
  try {
    await register(registerForm)
    ElMessage.success('注册成功，请登录')
    tab.value = 'login'
    loginForm.account = registerForm.email
  } catch {} finally { loading.value = false }
}
</script>

<style scoped>
.login-page {
  min-height: 100vh;
  display: grid;
  grid-template-columns: 1fr 480px;
}

/* Brand side */
.login-brand {
  display: flex; flex-direction: column;
  padding: 48px 56px;
  background: var(--cb-bg);
  border-right: 1px solid var(--cb-border);
}
.brand-top { display: flex; align-items: center; gap: 12px; }
.brand-logo {
  width: 40px; height: 40px;
  display: flex; align-items: center; justify-content: center;
  border-radius: var(--cb-radius-sm);
  background: var(--cb-primary-gradient);
  color: #fff;
}
.brand-name { font-size: 18px; font-weight: 800; color: var(--cb-text); }
.brand-sub { font-size: 12px; color: var(--cb-text-muted); font-weight: 600; margin-top: 2px; }
.brand-hero { margin-top: 80px; }
.brand-hero h1 { font-size: 42px; font-weight: 800; line-height: 1.15; color: var(--cb-text); margin: 0; }
.brand-hero .hero-gradient {
  background: var(--cb-primary-gradient);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
}
.brand-hero p { margin-top: 16px; font-size: 16px; color: var(--cb-text-secondary); line-height: 1.7; }
.brand-features { margin-top: 48px; display: grid; gap: 18px; }
.feat-item {
  display: grid; grid-template-columns: 38px 1fr; gap: 12px; align-items: center;
}
.feat-item > .el-icon {
  width: 38px; height: 38px; border-radius: var(--cb-radius-sm);
  background: var(--cb-primary-light); color: var(--cb-primary);
}
.feat-item:nth-child(2) > .el-icon { background: var(--cb-success-light); color: var(--cb-success); }
.feat-item:nth-child(3) > .el-icon { background: var(--cb-warning-light); color: var(--cb-warning); }
.feat-item strong { display: block; font-size: 14px; font-weight: 700; color: var(--cb-text); }
.feat-item span { display: block; margin-top: 2px; font-size: 12px; color: var(--cb-text-muted); }
.brand-footer {
  margin-top: auto; padding-top: 24px;
  display: flex; gap: 20px;
  font-size: 12px; color: var(--cb-text-muted);
}

/* Form side */
.login-form-section {
  display: flex; align-items: center; justify-content: center;
  background: var(--cb-surface);
  padding: 40px;
}
.login-card { width: 100%; max-width: 380px; }
.card-head { margin-bottom: 24px; }
.card-head h2 { font-size: 22px; font-weight: 800; color: var(--cb-text); margin: 0; }
.card-head p { margin-top: 6px; font-size: 14px; color: var(--cb-text-secondary); }

.login-tabs :deep(.el-tabs__nav-wrap::after) { display: none; }
.login-tabs :deep(.el-tabs__nav) {
  background: var(--cb-bg); border-radius: var(--cb-radius-sm); padding: 3px;
}
.login-tabs :deep(.el-tabs__item) {
  height: 36px; line-height: 36px; border-radius: 5px; font-weight: 700; font-size: 13px;
}
.login-tabs :deep(.el-tabs__item.is-active) { background: var(--cb-surface); color: var(--cb-primary); box-shadow: var(--cb-shadow-sm); }
.login-tabs :deep(.el-tabs__active-bar) { display: none; }

.login-tabs :deep(.el-form-item) { margin-bottom: 18px; }

.form-row { display: flex; justify-content: space-between; align-items: center; margin-bottom: 18px; }
.submit-btn { width: 100%; border-radius: var(--cb-radius-sm) !important; font-weight: 700; height: 44px; }

.password-strength { display: flex; align-items: center; gap: 6px; margin: -8px 0 14px; }
.password-strength span { height: 4px; flex: 1; border-radius: 99px; background: #E5E7EB; }
.password-strength span.active { background: var(--cb-success); }
.password-strength em { font-size: 12px; color: var(--cb-text-secondary); font-style: normal; font-weight: 700; }

@media (max-width: 860px) {
  .login-page { grid-template-columns: 1fr; overflow: auto; }
  .login-brand { min-height: auto; padding: 28px 24px; }
  .brand-hero { margin-top: 32px; }
  .brand-hero h1 { font-size: 28px; }
  .login-form-section { padding: 24px; }
}
</style>
