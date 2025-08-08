<template>
    <div class="not-found-container">
      <el-result
        icon="error"
        title="404 页面不存在"
        sub-title="抱歉，您访问的页面不存在或已被移除"
      >
        <template #extra>
          <el-button type="primary" @click="goHome">返回首页</el-button>
        </template>
      </el-result>
    </div>
  </template>
  
  <script>
  export default {
    methods: {
      goHome() {
        this.$router.push('/')
      }
    }
  }
  </script>
  
  <style scoped>
  .not-found-container {
    display: flex;
    justify-content: center;
    align-items: center;
    height: 80vh;
  }
  </style>