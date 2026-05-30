<template>
  <aside class="rpanel">
    <!-- Storage ring -->
    <section class="rp-section">
      <h3 class="rp-title">存储空间</h3>
      <div class="rp-ring-wrap">
        <svg class="rp-ring" viewBox="0 0 140 140">
          <defs>
            <linearGradient id="ringGrad" x1="0%" y1="0%" x2="100%" y2="0%">
              <stop offset="0%" stop-color="#4D8DFF" />
              <stop offset="100%" stop-color="#8B5CFF" />
            </linearGradient>
          </defs>
          <circle class="rp-ring-bg" cx="70" cy="70" r="56" />
          <circle
            class="rp-ring-fill"
            cx="70" cy="70" r="56"
            :style="{ strokeDashoffset: 351.86 - (351.86 * percent / 100) }"
            stroke="url(#ringGrad)"
          />
        </svg>
        <div class="rp-ring-center">
          <strong>{{ usedGB }}</strong>
          <span>GB 已用</span>
          <small>共 {{ totalGB }} GB</small>
        </div>
      </div>

      <!-- Category breakdown -->
      <div class="rp-types">
        <div v-for="t in fileTypes" :key="t.label" class="rp-type-row">
          <span class="rp-type-dot" :style="{ background: t.color }"></span>
          <span class="rp-type-label">{{ t.label }}</span>
          <span class="rp-type-count">{{ t.count }} 个</span>
          <span class="rp-type-val">{{ t.size || '-' }}</span>
        </div>
        <div v-if="fileTypes.length === 0" class="rp-types-empty">暂无文件</div>
      </div>
    </section>

    <!-- Recent shares -->
    <section class="rp-section">
      <h3 class="rp-title">最近分享</h3>
      <div v-if="recentShares.length > 0" class="rp-shares">
        <div v-for="s in recentShares" :key="s.id" class="rp-share-item" @click="$router.push('/shared')">
          <div class="rp-share-fileicon">
            <el-icon :size="16"><Document /></el-icon>
          </div>
          <div class="rp-share-info">
            <span class="rp-share-name">{{ s.fileName }}</span>
            <span class="rp-share-time">{{ s.timeAgo }}</span>
          </div>
          <span class="rp-share-status" :class="s.status">{{ s.status === 'active' ? '有效' : '已过期' }}</span>
        </div>
      </div>
      <div v-else class="rp-empty">
        <span>暂无分享记录</span>
        <router-link to="/shared" class="rp-empty-link">去分享文件</router-link>
      </div>
    </section>

    <!-- Upgrade CTA -->
    <section class="rp-section rp-section-last">
      <div class="rp-upgrade-cta" :class="{ warning: percent > 80 }">
        <div class="rp-upgrade-icon">
          <el-icon :size="18"><Present v-if="percent <= 80" /><WarningFilled v-else /></el-icon>
        </div>
        <div class="rp-upgrade-text">
          <span v-if="percent <= 80">需要更多空间？</span>
          <span v-else>存储空间即将用尽</span>
        </div>
        <el-button type="primary" size="small" round @click="$router.push('/user')">查看详情</el-button>
      </div>
    </section>
  </aside>
</template>

<script setup>
import { ref, onMounted, watch } from 'vue'
import { useStore } from 'vuex'
import { Document, Present, WarningFilled } from '@element-plus/icons-vue'
import { getUserStats } from '@/api/user'
import { getUserShares } from '@/api/share'

const store = useStore()

const usedGB = ref('0')
const totalGB = ref('0')
const percent = ref(0)
const fileTypes = ref([])
const recentShares = ref([])

function timeAgo(dateStr) {
  if (!dateStr) return ''
  const diff = Date.now() - new Date(dateStr).getTime()
  const mins = Math.floor(diff / 60000)
  if (mins < 1) return '刚刚'
  if (mins < 60) return `${mins} 分钟前`
  const hours = Math.floor(mins / 60)
  if (hours < 24) return `${hours} 小时前`
  const days = Math.floor(hours / 24)
  if (days < 30) return `${days} 天前`
  return `${Math.floor(days / 30)} 月前`
}

async function loadStorage() {
  try {
    const stats = await getUserStats()
    if (stats) {
      const q = stats.storage_quota || {}
      usedGB.value = q.used_gb?.toFixed(1) || '0'
      totalGB.value = q.total_gb?.toFixed(0) || '0'
      percent.value = Math.min(q.used_percent || 0, 100)
      if (stats.file_type_stats?.length) {
        fileTypes.value = stats.file_type_stats.map((t, i) => ({
          label: t.name,
          count: t.count || 0,
          size: t.size_str || (t.size_gb != null ? t.size_gb + ' GB' : '-'),
          color: ['#EC4899','#EF4444','#2F6BFF','#F59E0B','#8B5CF6','#6B7280'][i % 6]
        }))
      }
    }
  } catch {}
}

onMounted(async () => {
  loadStorage()

  try {
    const shares = await getUserShares()
    if (Array.isArray(shares)) {
      recentShares.value = shares.slice(0, 4).map(s => ({
        id: s.id,
        fileName: s.fileName,
        status: s.status,
        timeAgo: timeAgo(s.createdAt)
      }))
    }
  } catch {}
})

watch(() => store.state.file.needRefreshStorage, val => {
  if (val) { loadStorage(); store.commit('file/setNeedRefreshStorage', false) }
})
</script>

<style scoped>
.rpanel {
  padding: 20px 20px 32px;
}

.rp-section {
  padding-bottom: 24px;
  margin-bottom: 24px;
  border-bottom: 1px solid var(--cb-border-light);
}
.rp-section-last { border-bottom: 0; margin-bottom: 0; padding-bottom: 0; }

.rp-title {
  font-size: 13px;
  font-weight: 700;
  color: var(--cb-text);
  letter-spacing: -0.1px;
  margin: 0 0 16px;
}

/* ── Ring ── */
.rp-ring-wrap {
  position: relative;
  display: flex;
  justify-content: center;
  margin-bottom: 20px;
}
.rp-ring {
  width: 150px;
  height: 150px;
  transform: rotate(-90deg);
}
.rp-ring-bg {
  fill: none;
  stroke: #F2F3F5;
  stroke-width: 10;
}
.rp-ring-fill {
  fill: none;
  stroke-width: 10;
  stroke-linecap: round;
  stroke-dasharray: 351.86;
  transition: stroke-dashoffset .8s var(--cb-ease);
}
.rp-ring-center {
  position: absolute;
  inset: 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  text-align: center;
}
.rp-ring-center strong {
  font-size: 28px;
  font-weight: 800;
  color: var(--cb-text);
  letter-spacing: -1px;
  line-height: 1;
}
.rp-ring-center span {
  font-size: 12px;
  font-weight: 600;
  color: var(--cb-text-secondary);
}
.rp-ring-center small {
  font-size: 11px;
  color: var(--cb-text-muted);
  font-weight: 500;
}

/* ── Types ── */
.rp-types { display: grid; gap: 8px; }
.rp-type-row {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 13px;
}
.rp-type-dot {
  width: 8px; height: 8px;
  border-radius: 50%;
  flex-shrink: 0;
}
.rp-type-label { flex: 1; color: var(--cb-text-secondary); font-weight: 500; }
.rp-type-val { color: var(--cb-text); font-weight: 600; font-size: 12px; }

/* ── Shares ── */
.rp-shares { display: grid; gap: 4px; }
.rp-share-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 12px;
  border-radius: var(--cb-radius-sm);
  transition: background var(--cb-transition-fast);
  cursor: pointer;
}
.rp-share-item:hover { background: var(--cb-bg-alt); }
.rp-share-fileicon {
  width: 34px; height: 34px;
  border-radius: var(--cb-radius-xs);
  background: var(--cb-primary-light);
  color: var(--cb-primary);
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}
.rp-share-info {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 1px;
}
.rp-share-name {
  font-size: 13px;
  color: var(--cb-text);
  font-weight: 600;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.rp-share-time {
  font-size: 11px;
  color: var(--cb-text-muted);
  font-weight: 500;
}
.rp-share-status {
  font-size: 11px;
  font-weight: 600;
  padding: 2px 8px;
  border-radius: 99px;
  flex-shrink: 0;
}
.rp-share-status.active { color: #059669; background: #ECFDF5; }
.rp-share-status.expired { color: #9CA3AF; background: #F3F4F6; }

/* ── Empty shares ── */
.rp-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  padding: 20px 12px;
  text-align: center;
  font-size: 13px;
  color: var(--cb-text-muted);
  background: var(--cb-bg);
  border-radius: var(--cb-radius);
}
.rp-empty-link {
  font-size: 12px;
  font-weight: 600;
  color: var(--cb-primary);
  text-decoration: none;
}
.rp-empty-link:hover { text-decoration: underline; }

/* ── Upgrade CTA ── */
.rp-upgrade-cta {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 14px 16px;
  border-radius: var(--cb-radius);
  background: var(--cb-primary-light);
  transition: background var(--cb-transition-fast);
}
.rp-upgrade-cta.warning { background: var(--cb-warning-light); }

.rp-upgrade-icon {
  width: 36px; height: 36px;
  border-radius: var(--cb-radius-xs);
  background: rgba(47,107,255,.12);
  color: var(--cb-primary);
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}
.rp-upgrade-cta.warning .rp-upgrade-icon {
  background: rgba(245,158,11,.15);
  color: #D97706;
}

.rp-upgrade-text {
  flex: 1;
  font-size: 13px;
  font-weight: 600;
  color: var(--cb-text);
}
.rp-upgrade-cta.warning .rp-upgrade-text { color: #92400E; }
</style>
