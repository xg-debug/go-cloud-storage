<template>
  <div class="nf-root">
    <div class="nf-card">
      <div class="nf-code">404</div>
      <div class="nf-divider"></div>
      <h1>页面不存在</h1>
      <p>您访问的页面不存在或已被移除</p>
      <el-button type="primary" size="large" @click="goHome" round>
        <el-icon><HomeFilled /></el-icon>返回首页
      </el-button>
    </div>
  </div>
</template>

<script setup>
import { useRouter } from 'vue-router'
import { HomeFilled } from '@element-plus/icons-vue'
const router = useRouter()
function goHome() { router.push('/') }
</script>

<style scoped>
.nf-root {
  height: 100vh;
  display: flex; align-items: center; justify-content: center;
  background: var(--cb-bg);
}
.nf-card {
  text-align: center;
  padding: 48px;
}
.nf-code {
  font-size: 120px;
  font-weight: 900;
  letter-spacing: -4px;
  line-height: 1;
  margin-bottom: 0;
  background: var(--cb-primary-gradient);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
}
.nf-divider {
  width: 48px; height: 3px;
  border-radius: 99px;
  background: var(--cb-primary-gradient);
  margin: 16px auto 24px;
}
.nf-card h1 {
  font-size: 24px; font-weight: 800;
  color: var(--cb-text);
  margin: 0 0 8px;
  letter-spacing: -0.3px;
}
.nf-card p {
  font-size: 15px;
  color: var(--cb-text-muted);
  margin: 0 0 28px;
}
</style>
