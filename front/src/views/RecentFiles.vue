<template>
  <div class="page-wrap">
    <div class="page-hdr">
      <div class="page-hdr-title">
        <div class="page-hdr-icon" style="background:#FFFBEB;color:#F59E0B;">
          <el-icon :size="20"><Clock /></el-icon>
        </div>
        <div>
          <h1>最近访问</h1>
          <p>快速找回近期使用的文件</p>
        </div>
      </div>
      <el-select v-model="timeRange" size="default" @change="fetchData" style="width:120px;">
        <el-option label="今天" value="today" />
        <el-option label="本周" value="week" />
        <el-option label="本月" value="month" />
      </el-select>
    </div>

    <div class="page-body">
      <div v-if="!files || files.length === 0" class="cb-empty-state">
        <div class="empty-icon"><el-icon :size="36"><Clock /></el-icon></div>
        <h3>暂无最近文件</h3>
        <p>在所选时间范围内没有访问记录</p>
      </div>

      <template v-else>
        <div v-for="(day, idx) in files" :key="idx" class="day-group">
          <div class="day-label">{{ formatDay(day.date) }}</div>
          <div class="day-list">
            <div v-for="file in day.files" :key="file.id" class="recent-row" @dblclick="openFile(file)">
              <div class="rr-icon" :style="{ background: iconBg(file) }">
                <el-icon :size="20" :color="getFileIconColor(file.name, file.is_dir)">
                  <component :is="getFileIcon(file.name, file.is_dir)" />
                </el-icon>
              </div>
              <div class="rr-info">
                <div class="rr-name">{{ file.name }}</div>
                <div class="rr-meta">{{ file.size_str }} &middot; {{ file.modified || file.created_at }}</div>
              </div>
              <button class="rr-btn" @click.stop="openFile(file)">
                <el-icon :size="16"><ArrowRight /></el-icon>
              </button>
            </div>
          </div>
        </div>
      </template>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { ArrowRight, Clock } from '@element-plus/icons-vue'
import { getRecentFiles } from '@/api/file'
import { getFileIcon, getFileIconColor } from '@/utils/fileIcon'

const router = useRouter()
const timeRange = ref('week')
const files = ref([])

async function fetchData() {
  try { files.value = await getRecentFiles(timeRange.value) || [] } catch { files.value = [] }
}
function openFile(f) {
  if (f.is_dir) router.push({ name: 'MyDrive', query: { parentId: f.id } })
  else window.open(f.file_url, '_blank')
}
function iconBg(f) {
  if (f.is_dir) return '#FFFBF0'
  const e = (f.extension || '').toLowerCase()
  if (['jpg','jpeg','png','gif','webp'].includes(e)) return '#FDF2F8'
  if (['mp4','avi','mov','mkv','webm'].includes(e)) return '#FEF2F2'
  if (['mp3','wav','flac'].includes(e)) return '#F5F3FF'
  if (['pdf','doc','docx','txt'].includes(e)) return '#EEF4FF'
  return '#F8F9FB'
}
function formatDay(d) {
  if (!d) return '更早'
  const diff = Math.floor((Date.now() - new Date(d).getTime()) / 864e5)
  if (diff === 0) return '今天'
  if (diff === 1) return '昨天'
  return d
}
onMounted(fetchData)
</script>

<style scoped>
.day-group { margin-bottom: 4px; }
.day-group:first-child { margin-top: 0; }
.day-label {
  font-size: 11px; font-weight: 700; color: var(--cb-text-muted);
  text-transform: uppercase; letter-spacing: .5px; padding: 6px 4px 8px;
}
.day-list { display: grid; gap: 2px; }
.recent-row {
  display: flex; align-items: center; gap: 14px;
  padding: 10px 14px; border-radius: var(--cb-radius-sm);
  cursor: pointer; transition: background var(--cb-transition-fast);
}
.recent-row:hover { background: var(--cb-surface-muted); }
.rr-icon {
  width: 40px; height: 40px; border-radius: 10px;
  display: flex; align-items: center; justify-content: center; flex-shrink: 0;
}
.rr-info { flex: 1; min-width: 0; }
.rr-name { font-size: 14px; font-weight: 600; color: var(--cb-text); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; margin-bottom: 2px; }
.rr-meta { font-size: 12px; color: var(--cb-text-muted); }
.rr-btn {
  width: 32px; height: 32px; border: 1px solid var(--cb-border); border-radius: 50%;
  background: var(--cb-surface); color: var(--cb-text-muted); cursor: pointer;
  display: flex; align-items: center; justify-content: center; flex-shrink: 0;
  opacity: 0; transition: all var(--cb-transition-fast);
}
.recent-row:hover .rr-btn { opacity: 1; }
.rr-btn:hover { background: var(--cb-primary); color: #fff; border-color: var(--cb-primary); }
</style>
