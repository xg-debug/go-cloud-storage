<template>
  <div id="app">
    <router-view />
  </div>
</template>



<style>
/* #app {
  min-height: 100vh;
} */
:root {
  /* 主色调 - 科技蓝 */
  --primary-color: #3a86ff;
  --primary-light: #ebf3ff;
  --primary-dark: #2667cc;

  /* 辅助色 */
  --secondary-color: #8338ec;
  --success-color: #06d6a0;
  --warning-color: #ffbe0b;
  --danger-color: #ef476f;

  /* 背景色 */
  --bg-color: #f8fafc;
  --card-bg: #ffffff;

  /* 文字色 */
  --text-primary: #1e293b;
  --text-secondary: #64748b;
}

#app {
  height: 100vh;
  margin: 0;
  padding: 0;
  background-color: var(--bg-color);
  color: var(--text-primary);
  font-family: 'Inter', -apple-system, BlinkMacSystemFont, sans-serif;
}
</style>