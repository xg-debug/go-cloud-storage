<template>
  <div class="rp-page">
    <div class="rp-card">
      <div class="rp-head">
        <h1>重置密码</h1>
        <p>请设置你的新密码</p>
      </div>
      <el-form ref="formRef" :model="form" :rules="rules" @keyup.enter="handleSubmit">
        <el-form-item prop="password">
          <el-input v-model="form.password" type="password" placeholder="新密码（8位以上，含大小写字母和数字）" :prefix-icon="Lock" size="large" show-password />
        </el-form-item>
        <div class="password-strength" v-if="form.password">
          <span v-for="i in 3" :key="i" :class="{ active: passwordScore >= i }"></span>
          <em>{{ passwordText }}</em>
        </div>
        <el-form-item prop="passwordConfirm">
          <el-input v-model="form.passwordConfirm" type="password" placeholder="确认新密码" :prefix-icon="Lock" size="large" show-password />
        </el-form-item>
        <el-button type="primary" class="submit-btn" :loading="loading" @click="handleSubmit" size="large">重置密码</el-button>
      </el-form>
      <div v-if="success" class="rp-success">
        <el-icon :size="48" color="#10B981"><CircleCheckFilled /></el-icon>
        <p>密码重置成功</p>
        <a href="#" @click.prevent="$router.push('/login')">返回登录</a>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, reactive, ref } from 'vue'
import { useRoute } from 'vue-router'
import { ElMessage } from 'element-plus'
import { Lock, CircleCheckFilled } from '@element-plus/icons-vue'
import { resetPassword } from '@/api/auth'

const route = useRoute()
const form = reactive({ password: '', passwordConfirm: '' })
const formRef = ref()
const loading = ref(false)
const success = ref(false)

const passwordScore = computed(() => {
  const v = form.password || ''
  let s = 0
  if (v.length >= 8) s++
  if (/[a-z]/.test(v) && /[A-Z]/.test(v)) s++
  if (/\d/.test(v)) s++
  return s
})
const passwordText = computed(() => ['弱', '中', '强', '安全'][passwordScore.value])

const rules = {
  password: [
    { required: true, message: '请输入新密码', trigger: 'blur' },
    { min: 8, message: '至少8个字符', trigger: 'blur' },
    { pattern: /(?=.*[a-z])(?=.*[A-Z])(?=.*\d)/, message: '需包含大小写字母和数字', trigger: 'blur' }
  ],
  passwordConfirm: [
    { required: true, message: '请确认密码', trigger: 'blur' },
    { validator: (_, v, cb) => v !== form.password ? cb(new Error('两次密码不一致')) : cb(), trigger: 'blur' }
  ]
}

async function handleSubmit() {
  const ok = await formRef.value?.validate().catch(() => false)
  if (!ok) return
  const token = route.query.token
  if (!token) { ElMessage.error('无效的重置链接'); return }
  loading.value = true
  try {
    await resetPassword({ token, password: form.password })
    success.value = true
    ElMessage.success('密码已重置')
  } catch { ElMessage.error('重置失败，链接可能已过期') }
  finally { loading.value = false }
}
</script>

<style scoped>
.rp-page {
  min-height: 100vh;
  display: flex; align-items: center; justify-content: center;
  background: var(--cb-bg);
  padding: 24px;
}
.rp-card {
  width: 100%; max-width: 420px;
  background: var(--cb-surface);
  border: 1px solid var(--cb-border);
  border-radius: var(--cb-radius-xl);
  padding: 40px;
}
.rp-head { margin-bottom: 28px; }
.rp-head h1 { font-size: 24px; font-weight: 800; color: var(--cb-text); margin: 0 0 8px; letter-spacing: -0.3px; }
.rp-head p { font-size: 14px; color: var(--cb-text-secondary); margin: 0; }
.submit-btn { width: 100%; border-radius: var(--cb-radius-sm) !important; font-weight: 700; height: 44px; font-size: 15px; }
.password-strength { display: flex; align-items: center; gap: 6px; margin: -8px 0 14px; }
.password-strength span { height: 4px; flex: 1; border-radius: 99px; background: #E5E7EB; }
.password-strength span.active { background: var(--cb-success); }
.password-strength em { font-size: 12px; color: var(--cb-text-secondary); font-style: normal; font-weight: 700; }
.rp-success { text-align: center; padding-top: 24px; }
.rp-success p { margin: 16px 0 12px; font-size: 16px; font-weight: 600; color: var(--cb-text); }
.rp-success a { font-size: 14px; color: var(--cb-primary); text-decoration: none; font-weight: 600; }
</style>
