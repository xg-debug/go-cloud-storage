<template>
  <div class="fp-page">
    <div class="fp-card">
      <div class="fp-back">
        <a href="#" @click.prevent="$router.push('/login')" class="fp-back-link">
          <el-icon :size="16"><ArrowLeft /></el-icon>返回登录
        </a>
      </div>
      <div class="fp-head">
        <h1>忘记密码</h1>
        <p>输入你的注册邮箱，我们将发送密码重置链接</p>
      </div>
      <el-form ref="formRef" :model="form" :rules="rules" @keyup.enter="handleSubmit">
        <el-form-item prop="email">
          <el-input v-model="form.email" placeholder="请输入注册邮箱" :prefix-icon="Message" size="large" />
        </el-form-item>
        <el-button type="primary" class="submit-btn" :loading="loading" @click="handleSubmit" size="large">发送重置链接</el-button>
      </el-form>
      <div v-if="sent" class="fp-success">
        <el-icon :size="48" color="#10B981"><CircleCheckFilled /></el-icon>
        <p>密码重置邮件已发送至 <strong>{{ form.email }}</strong></p>
        <span>请检查邮箱并点击链接重置密码（30分钟内有效）</span>
      </div>
    </div>
  </div>
</template>

<script setup>
import { reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { ArrowLeft, Message, CircleCheckFilled } from '@element-plus/icons-vue'
import { forgotPassword } from '@/api/auth'

const form = reactive({ email: '' })
const formRef = ref()
const loading = ref(false)
const sent = ref(false)

const rules = {
  email: [
    { required: true, message: '请输入邮箱', trigger: 'blur' },
    { type: 'email', message: '格式不正确', trigger: 'blur' }
  ]
}

async function handleSubmit() {
  const ok = await formRef.value?.validate().catch(() => false)
  if (!ok) return
  loading.value = true
  try {
    await forgotPassword(form)
    sent.value = true
  } catch { ElMessage.error('发送失败，请检查邮箱是否正确') }
  finally { loading.value = false }
}
</script>

<style scoped>
.fp-page {
  min-height: 100vh;
  display: flex; align-items: center; justify-content: center;
  background: var(--cb-bg);
  padding: 24px;
}
.fp-card {
  width: 100%; max-width: 420px;
  background: var(--cb-surface);
  border: 1px solid var(--cb-border);
  border-radius: var(--cb-radius-xl);
  padding: 40px;
}
.fp-back { margin-bottom: 24px; }
.fp-back-link {
  display: inline-flex; align-items: center; gap: 6px;
  font-size: 13px; font-weight: 600; color: var(--cb-text-secondary); text-decoration: none;
}
.fp-back-link:hover { color: var(--cb-primary); }
.fp-head { margin-bottom: 28px; }
.fp-head h1 { font-size: 24px; font-weight: 800; color: var(--cb-text); margin: 0 0 8px; letter-spacing: -0.3px; }
.fp-head p { font-size: 14px; color: var(--cb-text-secondary); margin: 0; }
.submit-btn { width: 100%; border-radius: var(--cb-radius-sm) !important; font-weight: 700; height: 44px; font-size: 15px; }
.fp-success { text-align: center; padding-top: 24px; }
.fp-success p { margin: 16px 0 8px; font-size: 15px; color: var(--cb-text); }
.fp-success span { font-size: 13px; color: var(--cb-text-muted); }
</style>
