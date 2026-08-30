<template>
  <div class="auth-page">
    <!-- Left: brand -->
    <section class="auth-left">
      <div class="brand-top">
        <div class="brand-logo">
          <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M17.5 19H9a7 7 0 1 1 6.71-9h1.79a4.5 4.5 0 1 1 0 9Z"/></svg>
        </div>
        <div>
          <div class="brand-name">CloudBox</div>
          <div class="brand-sub">安全、便捷的个人云盘</div>
        </div>
      </div>
      <div class="brand-hero">
        <h1>存储你的</h1>
        <h1 class="hero-accent">每一个精彩瞬间</h1>
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
    <div class="auth-right">
      <div class="auth-right-inner">
        <!-- Toggle -->
        <div class="auth-switch">
          <button class="sw-btn" :class="{ on: tab === 'login' }" @click="tab = 'login'">登录</button>
          <button class="sw-btn" :class="{ on: tab === 'register' }" @click="tab = 'register'">注册</button>
        </div>

        <!-- Form stack: both occupy same space, no layout jump when switching -->
        <div class="form-stack">
          <!-- Login -->
          <div v-show="tab === 'login'" class="auth-form">
            <div class="form-header">
              <h2>欢迎回来</h2>
              <p>登录你的 CloudBox 账户</p>
            </div>
            <el-form
              ref="loginFormRef"
              :model="loginForm"
              :rules="loginRules"
              autocomplete="off"
              @submit.prevent="handleLogin"
              @keyup.enter="handleLogin"
            >
              <el-form-item prop="account">
                <el-input
                  v-model="loginForm.account"
                  placeholder="邮箱或手机号"
                  :prefix-icon="User"
                  size="large"
                />
              </el-form-item>
              <el-form-item prop="password">
                <el-input
                  v-model="loginForm.password"
                  type="password"
                  placeholder="密码"
                  :prefix-icon="Lock"
                  size="large"
                  show-password
                />
              </el-form-item>
              <div class="form-row">
                <el-checkbox v-model="rememberMe">记住我</el-checkbox>
                <a class="form-link" @click.prevent="goForgotPassword">忘记密码？</a>
              </div>
              <button type="submit" class="submit-btn" :disabled="loading">
                <span v-if="loading" class="btn-spinner"></span>
                <span v-else>登录</span>
              </button>
            </el-form>
          </div>

          <!-- Register -->
          <div v-show="tab === 'register'" class="auth-form">
            <div class="form-header">
              <h2>创建账号</h2>
              <p>注册即获 10GB 免费空间</p>
            </div>
            <el-form
              ref="registerFormRef"
              :model="registerForm"
              :rules="registerRules"
              autocomplete="off"
              @submit.prevent="handleRegister"
              @keyup.enter="handleRegister"
            >
              <el-form-item prop="email">
                <el-input
                  v-model="registerForm.email"
                  placeholder="邮箱地址"
                  :prefix-icon="Message"
                  size="large"
                />
              </el-form-item>
              <el-form-item prop="password">
                <el-input
                  v-model="registerForm.password"
                  type="password"
                  placeholder="密码（大小写字母+数字，8位以上）"
                  :prefix-icon="Lock"
                  size="large"
                  show-password
                />
              </el-form-item>
              <div class="pwd-meter" v-if="registerForm.password">
                <div class="pwd-meter-track">
                  <span class="pwd-meter-fill" :style="{ width: passwordScore * 33.33 + '%', background: pwdColor }"></span>
                </div>
                <span class="pwd-meter-label" :style="{ color: pwdColor }">{{ passwordText }}</span>
              </div>
              <el-form-item prop="password_confirm">
                <el-input
                  v-model="registerForm.password_confirm"
                  type="password"
                  placeholder="确认密码"
                  :prefix-icon="Lock"
                  size="large"
                  show-password
                />
              </el-form-item>
              <div class="terms-row">
                <el-checkbox v-model="termsAccepted">我已阅读并同意用户协议与隐私政策</el-checkbox>
              </div>
              <button type="submit" class="submit-btn" :disabled="loading || !termsAccepted">
                <span v-if="loading" class="btn-spinner"></span>
                <span v-else>创建账号</span>
              </button>
            </el-form>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { login, register } from '@/api/auth'
import { useStore } from 'vuex'
import { Clock, Folder, Lock, Message, Share, User } from '@element-plus/icons-vue'

const router = useRouter()
const store = useStore()
const tab = ref('login')
const loading = ref(false)
const rememberMe = ref(false)
const termsAccepted = ref(false)

const loginForm = reactive({ account: '', password: '' })
const registerForm = reactive({ email: '', password: '', password_confirm: '' })
const loginFormRef = ref()
const registerFormRef = ref()

const passwordScore = computed(() => {
  const v = registerForm.password || ''
  let s = 0
  if (v.length >= 8) s++
  if (/[a-z]/.test(v) && /[A-Z]/.test(v)) s++
  if (/\d/.test(v)) s++
  return s
})
const passwordText = computed(() => ['弱', '中', '强', '安全'][passwordScore.value])
const pwdColor = computed(() => ['#EF4444', '#F59E0B', '#10B981', '#10B981'][passwordScore.value])

const loginRules = {
  account: [{ required: true, message: '请输入邮箱或手机号', trigger: 'blur' }],
  password: [{ required: true, message: '请输入密码', trigger: 'blur' }]
}

const validatePassword = (_, v, cb) => {
  if (!v) return cb(new Error('请输入密码'))
  if (v.length < 8) return cb(new Error('至少8个字符'))
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
    const res = await login({ ...loginForm, remember: rememberMe.value })
    store.commit('setUserInfo', res.user_info)
    store.commit('setAuthChecked', true)
    ElMessage.success('登录成功')
    router.push('/')
  } catch { ElMessage.error('登录失败') } finally { loading.value = false }
}

function goForgotPassword() {
  router.push({ name: 'ForgotPassword' })
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
  } catch { ElMessage.error('注册失败') } finally { loading.value = false }
}
</script>

<style scoped>
.auth-page {
  min-height: 100vh;
  display: flex;
  background: var(--cb-bg);
}

/* ═══════ Left brand panel ═══════ */
.auth-left {
  flex: 1;
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
  background: var(--cb-primary);
  color: #fff;
}
.brand-name { font-size: 18px; font-weight: 800; color: var(--cb-text); }
.brand-sub { font-size: 12px; color: var(--cb-text-muted); font-weight: 600; margin-top: 2px; }
.brand-hero { margin-top: 80px; }
.brand-hero h1 { font-size: 42px; font-weight: 800; line-height: 1.15; color: var(--cb-text); margin: 0; }
.brand-hero .hero-accent { color: var(--cb-primary); }
.brand-hero p { margin-top: 16px; font-size: 16px; color: var(--cb-text-secondary); line-height: 1.7; }
.brand-features { margin-top: 48px; display: grid; gap: 18px; }
.feat-item {
  display: grid; grid-template-columns: 38px 1fr; gap: 12px; align-items: center;
}
.feat-item > :deep(.el-icon) {
  width: 38px; height: 38px; border-radius: var(--cb-radius-sm);
  background: var(--cb-primary-light); color: var(--cb-primary);
  display: flex; align-items: center; justify-content: center;
}
.feat-item:nth-child(2) > :deep(.el-icon) { background: var(--cb-success-light); color: var(--cb-success); }
.feat-item:nth-child(3) > :deep(.el-icon) { background: var(--cb-warning-light); color: var(--cb-warning); }
.feat-item strong { display: block; font-size: 14px; font-weight: 700; color: var(--cb-text); }
.feat-item span { display: block; margin-top: 2px; font-size: 12px; color: var(--cb-text-muted); }
.brand-footer {
  margin-top: auto; padding-top: 24px;
  display: flex; gap: 20px;
  font-size: 12px; color: var(--cb-text-muted);
}

/* ═══════ Right form panel ═══════ */
.auth-right {
  width: 540px;
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 48px;
  background: var(--cb-surface);
}
.auth-right-inner {
  width: 100%;
  max-width: 400px;
}

/* ── Toggle switch ── */
.auth-switch {
  display: flex;
  gap: 4px;
  background: var(--cb-surface);
  border: 1px solid var(--cb-border);
  border-radius: 10px;
  padding: 3px;
  margin-bottom: 36px;
}
.sw-btn {
  flex: 1;
  height: 40px;
  border: none;
  border-radius: 8px;
  background: transparent;
  font-size: 14px;
  font-weight: 600;
  color: var(--cb-text-muted);
  cursor: pointer;
  transition: all .2s var(--cb-ease);
}
.sw-btn.on {
  background: #1e293b;
  color: #fff;
  box-shadow: var(--cb-shadow-sm);
}

/* ── Form header ── */
.form-header {
  margin-bottom: 28px;
}
.form-header h2 {
  font-size: 24px;
  font-weight: 800;
  color: var(--cb-text);
  margin: 0 0 6px;
  letter-spacing: -.3px;
}
.form-header p {
  font-size: 14px;
  color: var(--cb-text-secondary);
  margin: 0;
}

/* ── Form stack (prevents height jump when switching) ── */
.form-stack {
  min-height: 430px;
}
.auth-form {
  width: 100%;
}

/* ── Input overrides ── */
:deep(.el-form-item) {
  margin-bottom: 18px;
}
:deep(.el-input__wrapper) {
  border-radius: 10px;
  background: var(--cb-surface);
  box-shadow: none !important;
  border: 1px solid var(--cb-border);
  padding: 2px 14px;
}
:deep(.el-input__wrapper:hover) {
  border-color: var(--cb-border-strong);
}
:deep(.el-input__wrapper.is-focus) {
  border-color: var(--cb-primary);
  background: var(--cb-surface);
}
:deep(.el-input__inner) {
  font-size: 14px;
  color: var(--cb-text);
}
:deep(.el-input__prefix) {
  color: var(--cb-text-muted);
}

/* ── Form row ── */
.form-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 24px;
}
.form-link {
  font-size: 13px;
  color: var(--cb-primary);
  text-decoration: none;
  font-weight: 600;
  cursor: pointer;
}
.form-link:hover { text-decoration: underline; }

:deep(.el-checkbox__label) {
  font-size: 13px;
  color: var(--cb-text-secondary);
}

/* ── Password meter ── */
.pwd-meter {
  display: flex;
  align-items: center;
  gap: 12px;
  margin: -8px 0 16px;
}
.pwd-meter-track {
  flex: 1;
  height: 4px;
  border-radius: 99px;
  background: #E5E7EB;
  overflow: hidden;
}
.pwd-meter-fill {
  height: 100%;
  border-radius: 99px;
  transition: width .35s var(--cb-ease), background .35s var(--cb-ease);
}
.pwd-meter-label {
  font-size: 12px;
  font-weight: 700;
  min-width: 24px;
}

/* ── Terms ── */
.terms-row {
  margin-bottom: 24px;
}

/* ── Submit button ── */
.submit-btn {
  width: 100%;
  height: 46px;
  border: none;
  border-radius: 10px;
  background: #1e293b;
  color: #fff;
  font-size: 15px;
  font-weight: 700;
  cursor: pointer;
  transition: background .2s var(--cb-ease), opacity .2s;
  display: flex;
  align-items: center;
  justify-content: center;
}
.submit-btn:hover { background: #0f172a; }
.submit-btn:disabled { opacity: .55; cursor: not-allowed; }

.btn-spinner {
  width: 18px; height: 18px;
  border: 2px solid rgba(255,255,255,.3);
  border-top-color: #fff;
  border-radius: 50%;
  animation: spin .6s linear infinite;
}
@keyframes spin { to { transform: rotate(360deg); } }

/* ── Responsive ── */
@media (max-width: 860px) {
  .auth-page { flex-direction: column; overflow: auto; }
  .auth-left { min-height: auto; padding: 28px 24px; }
  .brand-hero { margin-top: 32px; }
  .brand-hero h1 { font-size: 28px; }
  .auth-right { width: 100%; padding: 24px; }
  .auth-right-inner { max-width: 100%; }
}
</style>
