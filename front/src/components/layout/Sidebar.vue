<template>
  <nav class="sidenav">
    <!-- Logo -->
    <div class="sn-brand" @click="$router.push('/')">
      <div class="sn-logo">
        <svg width="24" height="24" viewBox="0 0 32 32" fill="none">
          <rect width="32" height="32" rx="8" fill="url(#lg)" />
          <path d="M10 22V12l6-4 6 4v10H10z" fill="#fff" opacity=".95" />
          <path d="M16 8l-6 4 6 4 6-4-6-4z" fill="#fff" opacity=".7" />
          <defs>
            <linearGradient id="lg" x1="0" y1="0" x2="32" y2="32">
              <stop stop-color="#4D8DFF"/><stop offset="1" stop-color="#8B5CFF"/>
            </linearGradient>
          </defs>
        </svg>
      </div>
      <span v-show="!collapsed" class="sn-name">CloudBox</span>
    </div>

    <!-- Nav items -->
    <div class="sn-nav">
      <router-link to="/" class="sn-item" exact-active-class="active" :title="collapsed ? '我的文件' : ''">
        <span class="sn-icon"><el-icon :size="20"><Folder /></el-icon></span>
        <span v-show="!collapsed" class="sn-label">我的文件</span>
      </router-link>

      <router-link to="/recent" class="sn-item" active-class="active" :title="collapsed ? '最近访问' : ''">
        <span class="sn-icon"><el-icon :size="20"><Clock /></el-icon></span>
        <span v-show="!collapsed" class="sn-label">最近访问</span>
      </router-link>

      <router-link to="/shared" class="sn-item" active-class="active" :title="collapsed ? '我的分享' : ''">
        <span class="sn-icon"><el-icon :size="20"><Share /></el-icon></span>
        <span v-show="!collapsed" class="sn-label">我的分享</span>
      </router-link>

      <router-link to="/starred" class="sn-item" active-class="active" :title="collapsed ? '收藏夹' : ''">
        <span class="sn-icon"><el-icon :size="20"><Star /></el-icon></span>
        <span v-show="!collapsed" class="sn-label">收藏夹</span>
      </router-link>

      <router-link to="/recycle" class="sn-item" active-class="active" :title="collapsed ? '回收站' : ''">
        <span class="sn-icon"><el-icon :size="20"><Delete /></el-icon></span>
        <span v-show="!collapsed" class="sn-label">回收站</span>
      </router-link>
    </div>

    <!-- Storage card -->
    <div v-show="!collapsed" class="sn-storage">
      <div class="sn-storage-header">
        <span class="sn-storage-label">存储空间</span>
        <span class="sn-storage-pct">{{ percent }}%</span>
      </div>
      <div class="sn-storage-bar">
        <div class="sn-storage-fill" :class="{ warn: percent > 80 }" :style="{ width: percent + '%' }"></div>
      </div>
      <div class="sn-storage-nums">
        <span>{{ usedGB }} GB 已用</span>
        <span>共 {{ totalGB }} GB</span>
      </div>
      <button class="sn-upgrade-btn" @click="$router.push('/user')">
        <el-icon :size="14"><Pointer /></el-icon>
        <span>升级存储空间</span>
      </button>
    </div>

    <!-- Collapse toggle -->
    <button class="sn-toggle" @click="$emit('toggle')">
      <el-icon :size="18"><DArrowLeft v-if="!collapsed" /><DArrowRight v-else /></el-icon>
    </button>
  </nav>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { Clock, DArrowLeft, DArrowRight, Delete, Folder, Pointer, Share, Star } from '@element-plus/icons-vue'
import { getUserStats } from '@/api/user'

defineProps({ collapsed: Boolean })
defineEmits(['toggle'])

const percent = ref(0)
const usedGB = ref('0')
const totalGB = ref('200')

onMounted(async () => {
  try {
    const stats = await getUserStats()
    if (stats?.storage_quota) {
      const q = stats.storage_quota
      usedGB.value = (q.used_gb || 0).toFixed(1)
      totalGB.value = (q.total_gb || 200).toFixed(0)
      percent.value = Math.min(q.used_percent || 0, 100)
    }
  } catch {}
})
</script>

<style scoped>
.sidenav {
  height: 100%;
  display: flex;
  flex-direction: column;
  background: var(--cb-surface);
  border-right: 1px solid var(--cb-border);
  padding: 0 12px;
  position: relative;
}

/* Brand */
.sn-brand {
  height: var(--cb-header-h);
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 0 4px;
  cursor: pointer;
  flex-shrink: 0;
}
.sn-logo { flex-shrink: 0; }
.sn-name {
  font-size: 18px;
  font-weight: 800;
  color: var(--cb-text);
  letter-spacing: -0.3px;
}

/* Nav */
.sn-nav {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 2px;
  padding: 8px 0;
  overflow-y: auto;
}
.sn-item {
  display: flex;
  align-items: center;
  gap: 12px;
  height: 42px;
  padding: 0 12px;
  border-radius: var(--cb-radius-sm);
  color: var(--cb-text-secondary);
  text-decoration: none;
  font-size: 14px;
  font-weight: 500;
  transition: all var(--cb-transition-fast);
  white-space: nowrap;
}
.sn-item:hover { background: var(--cb-bg-alt); color: var(--cb-text); }
.sn-item.active {
  background: var(--cb-primary-light);
  color: var(--cb-primary);
  font-weight: 600;
}
.sn-icon {
  width: 22px;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}
.sn-label { overflow: hidden; text-overflow: ellipsis; }

/* Storage */
.sn-storage {
  margin: 0 4px 8px;
  padding: 14px;
  border-radius: var(--cb-radius);
  background: var(--cb-bg);
  border: 1px solid var(--cb-border-light);
  flex-shrink: 0;
}
.sn-storage-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 10px;
}
.sn-storage-label {
  font-size: 12px;
  font-weight: 700;
  color: var(--cb-text);
}
.sn-storage-pct {
  font-size: 12px;
  font-weight: 700;
  color: var(--cb-primary);
}
.sn-storage-bar {
  height: 5px;
  background: #E5E7EB;
  border-radius: 99px;
  overflow: hidden;
  margin-bottom: 8px;
}
.sn-storage-fill {
  height: 100%;
  border-radius: 99px;
  background: var(--cb-primary-gradient);
  transition: width .6s var(--cb-ease);
}
.sn-storage-fill.warn { background: linear-gradient(90deg, #F59E0B, #EF4444); }
.sn-storage-nums {
  display: flex;
  justify-content: space-between;
  font-size: 11px;
  color: var(--cb-text-muted);
  font-weight: 500;
  margin-bottom: 12px;
}
.sn-upgrade-btn {
  width: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  padding: 8px 0;
  border: 1px solid var(--cb-border);
  border-radius: var(--cb-radius-xs);
  background: var(--cb-surface);
  color: var(--cb-primary);
  font-size: 12px;
  font-weight: 600;
  cursor: pointer;
  transition: all var(--cb-transition-fast);
}
.sn-upgrade-btn:hover {
  background: var(--cb-primary);
  color: #fff;
  border-color: var(--cb-primary);
}

/* Toggle */
.sn-toggle {
  position: absolute;
  right: -12px;
  bottom: 80px;
  width: 24px; height: 24px;
  border: 1px solid var(--cb-border);
  border-radius: 50%;
  background: var(--cb-surface);
  color: var(--cb-text-muted);
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  box-shadow: var(--cb-shadow-xs);
  transition: all var(--cb-transition-fast);
}
.sn-toggle:hover { color: var(--cb-text); border-color: var(--cb-border-strong); }
</style>
