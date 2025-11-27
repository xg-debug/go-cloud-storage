<template>
  <div id="app">
    <router-view />
  </div>
</template>

<style>
@import url('https://fonts.googleapis.com/css2?family=Inter:wght@300;400;500;600;700;800&display=swap');

:root {
  /* 主色调 - 现代渐变蓝紫 */
  --primary-color: #6366f1;
  --primary-light: #eef2ff;
  --primary-dark: #4f46e5;
  --primary-gradient: linear-gradient(135deg, #6366f1 0%, #8b5cf6 100%);
  
  /* 辅助色 */
  --secondary-color: #8b5cf6;
  --success-color: #10b981;
  --success-light: #d1fae5;
  --warning-color: #f59e0b;
  --warning-light: #fef3c7;
  --danger-color: #ef4444;
  --danger-light: #fee2e2;
  --info-color: #06b6d4;
  --info-light: #cffafe;

  /* 背景色 */
  --bg-primary: #f8fafc;
  --bg-secondary: #f1f5f9;
  --bg-tertiary: #e2e8f0;
  --card-bg: #ffffff;
  --glass-bg: rgba(255, 255, 255, 0.8);
  --glass-border: rgba(255, 255, 255, 0.3);

  /* 文字色 */
  --text-primary: #0f172a;
  --text-secondary: #475569;
  --text-tertiary: #94a3b8;
  --text-inverse: #ffffff;

  /* 边框色 */
  --border-light: #e2e8f0;
  --border-medium: #cbd5e1;
  --border-dark: #94a3b8;

  /* 阴影 */
  --shadow-sm: 0 1px 2px 0 rgba(0, 0, 0, 0.05);
  --shadow-md: 0 4px 6px -1px rgba(0, 0, 0, 0.1), 0 2px 4px -2px rgba(0, 0, 0, 0.1);
  --shadow-lg: 0 10px 15px -3px rgba(0, 0, 0, 0.1), 0 4px 6px -4px rgba(0, 0, 0, 0.1);
  --shadow-xl: 0 20px 25px -5px rgba(0, 0, 0, 0.1), 0 8px 10px -6px rgba(0, 0, 0, 0.1);
  --shadow-glow: 0 0 20px rgba(99, 102, 241, 0.3);
  --shadow-card: 0 1px 3px rgba(0, 0, 0, 0.08), 0 1px 2px rgba(0, 0, 0, 0.06);

  /* 圆角 */
  --radius-sm: 6px;
  --radius-md: 10px;
  --radius-lg: 16px;
  --radius-xl: 24px;
  --radius-full: 9999px;

  /* 过渡 */
  --transition-fast: 150ms cubic-bezier(0.4, 0, 0.2, 1);
  --transition-normal: 250ms cubic-bezier(0.4, 0, 0.2, 1);
  --transition-slow: 350ms cubic-bezier(0.4, 0, 0.2, 1);

  /* 间距 */
  --spacing-xs: 4px;
  --spacing-sm: 8px;
  --spacing-md: 16px;
  --spacing-lg: 24px;
  --spacing-xl: 32px;
  --spacing-2xl: 48px;
}

* {
  margin: 0;
  padding: 0;
  box-sizing: border-box;
}

html, body {
  height: 100%;
  overflow: hidden;
}

#app {
  height: 100vh;
  margin: 0;
  padding: 0;
  background: #f1f5f9;
  color: var(--text-primary);
  font-family: 'Inter', -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
  font-feature-settings: 'cv02', 'cv03', 'cv04', 'cv11';
  -webkit-font-smoothing: antialiased;
  -moz-osx-font-smoothing: grayscale;
}

/* 全局滚动条样式 */
::-webkit-scrollbar {
  width: 8px;
  height: 8px;
}

::-webkit-scrollbar-track {
  background: transparent;
  border-radius: var(--radius-full);
}

::-webkit-scrollbar-thumb {
  background: var(--border-medium);
  border-radius: var(--radius-full);
  transition: background var(--transition-fast);
}

::-webkit-scrollbar-thumb:hover {
  background: var(--border-dark);
}

.el-dialog__header {
  padding: var(--spacing-lg) var(--spacing-xl) !important;
  border-bottom: 1px solid var(--border-light);
}

.el-dialog__body {
  padding: var(--spacing-xl) !important;
}

.el-dialog__footer {
  padding: var(--spacing-lg) var(--spacing-xl) !important;
  border-top: 1px solid var(--border-light);
}

.el-message-box {
  border-radius: var(--radius-lg) !important;
}

.el-dropdown-menu {
  border-radius: 4px !important;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.1) !important;
  border: 1px solid var(--border-light) !important;
  padding: 4px !important;
}

.el-dropdown-menu__item {
  border-radius: 4px !important;
  padding: 8px 16px !important;
  margin: 2px 0 !important;
}

.el-dropdown-menu__item:hover {
  background: #f5f5f5 !important;
  color: var(--text-primary) !important;
}

/* 全局动画 */
@keyframes fadeIn {
  from { opacity: 0; transform: translateY(10px); }
  to { opacity: 1; transform: translateY(0); }
}

@keyframes slideIn {
  from { opacity: 0; transform: translateX(-20px); }
  to { opacity: 1; transform: translateX(0); }
}

@keyframes pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.5; }
}

@keyframes shimmer {
  0% { background-position: -200% 0; }
  100% { background-position: 200% 0; }
}

.animate-fade-in {
  animation: fadeIn 0.3s ease-out;
}

.animate-slide-in {
  animation: slideIn 0.3s ease-out;
}
</style>