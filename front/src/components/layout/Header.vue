<template>
  <div class="topbar">
    <!-- Left: sidebar toggle + logo area handled by sidebar -->
    <div class="tb-left">
      <button class="tb-icon-btn" @click="$emit('toggle-sidebar')" title="折叠侧边栏">
        <el-icon :size="20"><Menu /></el-icon>
      </button>
    </div>

    <!-- Center: search -->
    <div class="tb-search">
      <el-icon :size="15" class="tb-search-icon"><Search /></el-icon>
      <input
        v-model="query"
        class="tb-search-input"
        placeholder="搜索文件、文件夹与分享内容..."
        @keyup.enter="doSearch"
      />
      <button v-if="query" class="tb-search-clear" @click="query = ''">
        <el-icon :size="14"><Close /></el-icon>
      </button>
      <kbd class="tb-search-kbd">⌘K</kbd>
    </div>

    <!-- Right: actions -->
    <div class="tb-right">
      <!-- Upload -->
      <button class="tb-icon-btn" @click="uploadDialogVisible = true" title="上传文件">
        <el-icon :size="20"><Upload /></el-icon>
      </button>
      <FileUploadDialog v-model="uploadDialogVisible" :parent-id="currentDirId" @success="onUploadDone" />

      <!-- Notifications -->
      <el-dropdown trigger="click" placement="bottom-end" popper-class="hd-notif-popper">
        <button class="tb-icon-btn" :class="{ 'has-unread': unreadCount > 0 }">
          <el-icon :size="20"><Bell /></el-icon>
          <span v-if="unreadCount > 0" class="tb-badge">{{ unreadCount > 99 ? '99+' : unreadCount }}</span>
        </button>
        <template #dropdown>
          <div class="hd-notif">
            <div class="hd-notif-head">
              <strong>通知</strong>
              <el-button v-if="unreadCount" type="primary" link size="small" @click="markAllRead">全部已读</el-button>
            </div>
            <div v-if="!notifications.length" class="hd-notif-empty">
              <el-icon :size="32"><Bell /></el-icon>
              <span>暂无通知</span>
            </div>
            <div v-else class="hd-notif-list">
              <div v-for="n in notifications" :key="n.id" class="hd-notif-item" :class="{ unread: !n.is_read }" @click="readOne(n)">
                <span class="hd-notif-dot" v-if="!n.is_read"></span>
                <div>
                  <div class="hd-notif-title">{{ n.title }}</div>
                  <div class="hd-notif-msg">{{ n.message }}</div>
                  <div class="hd-notif-time">{{ n.created_at }}</div>
                </div>
              </div>
            </div>
          </div>
        </template>
      </el-dropdown>

      <!-- User -->
      <el-dropdown trigger="click" placement="bottom-end" popper-class="hd-user-popper">
        <button class="tb-user-btn">
          <el-avatar :size="32" :src="user?.avatar">
            <el-icon :size="16"><User /></el-icon>
          </el-avatar>
        </button>
        <template #dropdown>
          <div class="hd-user-panel">
            <div class="hd-user-info">
              <el-avatar :size="44" :src="user?.avatar"><el-icon :size="20"><User /></el-icon></el-avatar>
              <div>
                <strong>{{ user?.username }}</strong>
                <span>{{ user?.email }}</span>
              </div>
            </div>
            <div class="hd-user-links">
              <button @click="$router.push('/user')">
                <el-icon :size="16"><User /></el-icon>个人中心
              </button>
              <button class="danger" @click="doLogout">
                <el-icon :size="16"><SwitchButton /></el-icon>退出登录
              </button>
            </div>
          </div>
        </template>
      </el-dropdown>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useStore } from 'vuex'
import {
  Bell, Close, Menu, Search, SwitchButton, Upload, User
} from '@element-plus/icons-vue'
import { logout } from '@/api/auth'
import { getNotifications, markAsRead, markAllAsRead } from '@/api/notification'
import FileUploadDialog from '@/components/FileUploadDialog.vue'

defineEmits(['toggle-sidebar'])

const router = useRouter()
const store = useStore()
const user = computed(() => store.state.userInfo)
const currentDirId = computed(() => store.state.userInfo?.rootFolderId || '')

const query = ref('')
const uploadDialogVisible = ref(false)
const notifications = ref([])
const unreadCount = ref(0)

function doSearch() {
  if (query.value.trim()) {
    router.push({ name: 'MyDrive', query: { search: query.value.trim() } })
  }
}

function onUploadDone() { store.commit('file/setNeedRefresh', true) }

async function loadNotifications() {
  try {
    const res = await getNotifications()
    notifications.value = res?.notifications || []
    unreadCount.value = res?.unread_count || notifications.value.filter(n => !n.is_read).length
  } catch {}
}

async function markAllRead() {
  try { await markAllAsRead(); notifications.value = []; unreadCount.value = 0 } catch {}
}

async function readOne(n) {
  try {
    await markAsRead(n.id)
    notifications.value = notifications.value.filter(x => x.id !== n.id)
    unreadCount.value = Math.max(0, unreadCount.value - 1)
  } catch {}
  if (n.link) router.push(n.link)
}

async function doLogout() {
  try { await logout() } catch {}
  store.commit('clearAuth')
  router.push('/login')
}

onMounted(loadNotifications)
</script>

<style scoped>
.topbar {
  width: 100%;
  display: flex;
  align-items: center;
  gap: 16px;
}

/* ── Left ── */
.tb-left { flex-shrink: 0; }

/* ── Search ── */
.tb-search {
  flex: 1;
  max-width: 520px;
  position: relative;
  margin: 0 auto;
}
.tb-search-icon {
  position: absolute;
  left: 14px;
  top: 50%;
  transform: translateY(-50%);
  color: var(--cb-text-muted);
  pointer-events: none;
}
.tb-search-input {
  width: 100%;
  height: 40px;
  padding: 0 80px 0 40px;
  border: 1px solid var(--cb-border);
  border-radius: 10px;
  background: var(--cb-bg-alt);
  font-size: 14px;
  color: var(--cb-text);
  outline: none;
  transition: all var(--cb-transition-fast);
}
.tb-search-input::placeholder { color: var(--cb-text-placeholder); font-weight: 500; }
.tb-search-input:focus {
  border-color: var(--cb-border-focus);
  background: var(--cb-surface);
  box-shadow: var(--cb-focus-ring);
}
.tb-search-clear {
  position: absolute;
  right: 44px;
  top: 50%;
  transform: translateY(-50%);
  width: 22px; height: 22px;
  border: 0; border-radius: 50%;
  background: var(--cb-border);
  color: var(--cb-text-muted);
  cursor: pointer;
  display: flex; align-items: center; justify-content: center;
  transition: all var(--cb-transition-fast);
}
.tb-search-clear:hover { background: var(--cb-border-strong); color: var(--cb-text); }
.tb-search-kbd {
  position: absolute;
  right: 10px;
  top: 50%;
  transform: translateY(-50%);
  padding: 2px 6px;
  border-radius: 4px;
  background: var(--cb-bg);
  border: 1px solid var(--cb-border);
  font-size: 11px;
  font-weight: 600;
  color: var(--cb-text-muted);
  font-family: inherit;
  pointer-events: none;
}

/* ── Buttons ── */
.tb-right {
  display: flex;
  align-items: center;
  gap: 4px;
  flex-shrink: 0;
}
.tb-icon-btn {
  width: 38px; height: 38px;
  border: 0;
  border-radius: 10px;
  background: transparent;
  color: var(--cb-text-secondary);
  cursor: pointer;
  display: flex; align-items: center; justify-content: center;
  position: relative;
  transition: all var(--cb-transition-fast);
}
.tb-icon-btn:hover { background: var(--cb-bg-alt); color: var(--cb-text); }
.tb-badge {
  position: absolute;
  top: 4px; right: 4px;
  min-width: 16px; height: 16px;
  padding: 0 4px;
  border-radius: 99px;
  background: var(--cb-danger);
  color: #fff;
  font-size: 10px;
  font-weight: 700;
  display: flex; align-items: center; justify-content: center;
  border: 2px solid var(--cb-surface);
}

/* ── User ── */
.tb-user-btn {
  width: 38px; height: 38px;
  border: 0; border-radius: 50%;
  background: transparent;
  cursor: pointer;
  padding: 2px;
  display: flex; align-items: center; justify-content: center;
  margin-left: 6px;
}

@media (max-width: 768px) {
  .tb-search { max-width: 240px; }
  .tb-search-kbd { display: none; }
}
</style>

<style>
.hd-notif-popper {
  border-radius: var(--cb-radius) !important;
  border: 1px solid var(--cb-border) !important;
  box-shadow: var(--cb-shadow-md) !important;
  padding: 0 !important; min-width: 320px !important;
  margin-top: 6px !important; overflow: hidden;
}
.hd-notif-head {
  display: flex; justify-content: space-between; align-items: center;
  padding: 14px 16px; border-bottom: 1px solid var(--cb-border-light);
}
.hd-notif-head strong { font-size: 14px; color: var(--cb-text); }
.hd-notif-empty {
  display: flex; flex-direction: column; align-items: center; gap: 8px;
  padding: 40px 16px; color: var(--cb-text-muted); font-size: 13px;
}
.hd-notif-list { max-height: 320px; overflow-y: auto; }
.hd-notif-item {
  display: flex; gap: 10px; padding: 12px 16px; cursor: pointer;
  border-bottom: 1px solid var(--cb-border-light);
  transition: background var(--cb-transition-fast);
}
.hd-notif-item:hover { background: var(--cb-bg-alt); }
.hd-notif-item.unread { background: #FAFBFC; }
.hd-notif-dot { width: 8px; height: 8px; border-radius: 50%; background: var(--cb-primary); margin-top: 4px; flex-shrink: 0; }
.hd-notif-title { font-size: 13px; font-weight: 600; color: var(--cb-text); margin-bottom: 2px; }
.hd-notif-msg { font-size: 12px; color: var(--cb-text-secondary); line-height: 1.4; }
.hd-notif-time { font-size: 11px; color: var(--cb-text-muted); margin-top: 4px; }

.hd-user-popper {
  border-radius: var(--cb-radius) !important;
  border: 1px solid var(--cb-border) !important;
  box-shadow: var(--cb-shadow-md) !important;
  padding: 0 !important; min-width: 220px !important;
  margin-top: 6px !important; overflow: hidden;
}
.hd-user-info {
  display: flex; align-items: center; gap: 12px;
  padding: 16px; border-bottom: 1px solid var(--cb-border-light);
}
.hd-user-info strong { display: block; font-size: 14px; color: var(--cb-text); }
.hd-user-info span { font-size: 12px; color: var(--cb-text-muted); }
.hd-user-links { padding: 6px; }
.hd-user-links button {
  width: 100%; display: flex; align-items: center; gap: 10px;
  padding: 10px 12px; border: 0; border-radius: 8px;
  background: transparent; font-size: 13px; font-weight: 500; color: var(--cb-text-secondary);
  cursor: pointer; transition: all var(--cb-transition-fast);
}
.hd-user-links button:hover { background: var(--cb-bg-alt); color: var(--cb-text); }
.hd-user-links button.danger { color: var(--cb-danger); }
.hd-user-links button.danger:hover { background: var(--cb-danger-light); }
</style>
