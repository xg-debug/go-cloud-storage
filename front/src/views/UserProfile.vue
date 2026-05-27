<template>
  <div class="page-wrap">
    <div class="page-hdr">
      <div class="page-hdr-title">
        <div class="page-hdr-icon" style="background:var(--cb-primary-light);color:var(--cb-primary);">
          <el-icon :size="20"><User /></el-icon>
        </div>
        <div><h1>个人中心</h1><p>管理你的账户和存储空间</p></div>
      </div>
    </div>
    <div class="page-body">
      <!-- Profile info card -->
      <div class="info-card">
        <div class="info-bg"></div>
        <div class="info-body">
          <div class="avatar-stack" @click="avatarInput?.click()">
            <div class="avatar-ring">
              <el-avatar :size="88" :src="user?.avatar" class="hero-avatar">
                <el-icon :size="34"><User /></el-icon>
              </el-avatar>
            </div>
            <div class="avatar-badge"><el-icon :size="13"><Camera /></el-icon></div>
            <input ref="avatarInput" type="file" accept="image/*" hidden @change="onAvatarChange" />
          </div>
          <div class="info-main">
            <h2 class="info-name">{{ user?.username || '用户' }}</h2>
            <div class="info-tags">
              <span class="info-tag"><el-icon :size="14"><Message /></el-icon>{{ user?.email }}</span>
              <span v-if="user?.phone" class="info-tag"><el-icon :size="14"><Phone /></el-icon>{{ user?.phone }}</span>
              <span class="info-tag"><el-icon :size="14"><Clock /></el-icon>{{ user?.registerTime || '未知' }}</span>
            </div>
          </div>
          <div class="info-stats">
            <div class="is-item">
              <span class="is-val">{{ fileStats.totalFiles }}</span>
              <span class="is-lbl">文件总数</span>
            </div>
            <div class="is-item">
              <span class="is-val">{{ storage.used }}<small> GB</small></span>
              <span class="is-lbl">已用空间</span>
            </div>
            <div class="is-item">
              <span class="is-val">{{ fileStats.sharedFiles }}</span>
              <span class="is-lbl">分享中</span>
            </div>
            <div class="is-item">
              <span class="is-val">{{ storage.total }}<small> GB</small></span>
              <span class="is-lbl">总容量</span>
            </div>
          </div>
        </div>
      </div>

      <!-- Edit profile form -->
      <div class="card">
        <div class="card-title"><el-icon :size="17"><User /></el-icon>编辑个人资料</div>
        <el-form :model="profileForm" label-width="80px" class="edit-form">
          <div class="form-grid">
            <el-form-item label="用户名">
              <el-input v-model="profileForm.username" placeholder="设置用户名" />
            </el-form-item>
            <el-form-item label="邮箱">
              <el-input :model-value="user?.email" readonly disabled />
            </el-form-item>
            <el-form-item label="手机号">
              <el-input v-model="profileForm.phone" placeholder="绑定手机号" />
            </el-form-item>
            <el-form-item label="注册时间">
              <el-input :model-value="user?.registerTime || '-'" readonly disabled />
            </el-form-item>
          </div>
          <div class="form-actions">
            <el-button type="primary" size="default" @click="saveProfile" :loading="saving" round>
              <el-icon :size="15"><Check /></el-icon>保存更改
            </el-button>
          </div>
        </el-form>
      </div>

      <!-- Cards row -->
      <div class="cards-row">
        <div class="card">
          <div class="card-title"><el-icon :size="17"><PieChart /></el-icon>存储空间</div>
          <div class="storage-inline">
            <div class="ring-wrap">
              <svg class="ring-svg" viewBox="0 0 140 140">
                <circle class="ring-bg" cx="70" cy="70" r="58" />
                <circle class="ring-fill" cx="70" cy="70" r="58"
                  :style="{ strokeDashoffset: 364.42 - (364.42 * storage.pct / 100), stroke: storageColor(storage.pct) }" />
              </svg>
              <div class="ring-center">
                <strong>{{ storage.pct }}%</strong>
                <small>{{ storage.used }} / {{ storage.total }} GB</small>
              </div>
            </div>
            <div class="type-bars">
              <div v-for="t in fileTypes" :key="t.type" class="tb-row">
                <span class="tb-dot" :style="{ background: t.color }"></span>
                <span class="tb-label">{{ t.name }}</span>
                <span class="tb-size">{{ t.size || '-' }}</span>
                <div class="tb-track"><div class="tb-fill" :style="{ width: Math.max(t.percentage, 2) + '%', background: t.color }"></div></div>
              </div>
            </div>
          </div>
        </div>

        <div class="card">
          <div class="card-title"><el-icon :size="17"><Lock /></el-icon>修改密码</div>
          <el-form ref="pwdFormRef" :model="pwdForm" :rules="pwdRules" label-width="80px" class="styled-form">
            <el-form-item label="当前密码" prop="oldPassword">
              <el-input v-model="pwdForm.oldPassword" type="password" show-password placeholder="输入当前密码" />
            </el-form-item>
            <el-form-item label="新密码" prop="newPassword">
              <el-input v-model="pwdForm.newPassword" type="password" show-password placeholder="至少6位，含大小写字母和数字" />
            </el-form-item>
            <el-form-item label="确认密码" prop="confirmPassword">
              <el-input v-model="pwdForm.confirmPassword" type="password" show-password placeholder="再次输入新密码" />
            </el-form-item>
            <el-form-item>
              <el-button type="primary" @click="changePassword" :loading="changingPwd" round>
                <el-icon :size="15"><Key /></el-icon>更新密码
              </el-button>
            </el-form-item>
          </el-form>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, onMounted, reactive, ref } from 'vue'
import { useStore } from 'vuex'
import { ElMessage } from 'element-plus'
import { Camera, Check, Clock, Key, Lock, Message, Phone, PieChart, User } from '@element-plus/icons-vue'
import { updateProfile, uploadAvatar, getUserStats, updatePassword } from '@/api/user'

const store = useStore()
const user = computed(() => store.state.userInfo)

const profileForm = reactive({ username: '', phone: '' })
const saving = ref(false)
const avatarInput = ref(null)
const storage = reactive({ used: '0', total: '10', pct: 0 })
const fileStats = reactive({ totalFiles: 0, folders: 0, sharedFiles: 0 })
const fileTypes = ref([])
const typeColors = ['#2F6BFF', '#10B981', '#F59E0B', '#EF4444', '#8B5CF6']

function storageColor(pct) { if (pct > 90) return 'var(--cb-danger)'; if (pct > 70) return 'var(--cb-warning)'; return 'var(--cb-primary)' }

const pwdForm = reactive({ oldPassword: '', newPassword: '', confirmPassword: '' })
const pwdFormRef = ref(null)
const changingPwd = ref(false)

const pwdRules = {
  oldPassword: [{ required: true, message: '请输入当前密码', trigger: 'blur' }],
  newPassword: [
    { required: true, min: 6, message: '至少6位', trigger: 'blur' },
    { pattern: /(?=.*[a-z])(?=.*[A-Z])(?=.*\d)/, message: '需包含大小写字母和数字', trigger: 'blur' }
  ],
  confirmPassword: [
    { required: true, message: '请确认密码', trigger: 'blur' },
    { validator: (_, v, cb) => v !== pwdForm.newPassword ? cb(new Error('两次密码不一致')) : cb(), trigger: 'blur' }
  ]
}

async function loadStats() {
  try {
    const d = await getUserStats()
    if (d) {
      const q = d.storage_quota || {}
      storage.used = (q.used_gb || 0).toFixed(1)
      storage.total = (q.total_gb || 10).toFixed(1)
      storage.pct = q.used_percent || 0
      const fs = d.file_stats || {}
      fileStats.totalFiles = fs.total_files || 0
      fileStats.folders = fs.folders || 0
      fileStats.sharedFiles = fs.shared_files || 0
      fileTypes.value = (d.file_type_stats || []).map((t, i) => ({ ...t, color: typeColors[i % typeColors.length] }))
    }
  } catch {}
}

function initForm() {
  if (user.value) {
    profileForm.username = user.value.username || ''
    profileForm.phone = user.value.phone || ''
  }
}

async function saveProfile() {
  saving.value = true
  try {
    await updateProfile(profileForm)
    store.commit('setUserInfo', { ...user.value, username: profileForm.username, phone: profileForm.phone })
    ElMessage.success('资料已更新')
  } catch {} finally { saving.value = false }
}

async function onAvatarChange(e) {
  const f = e.target.files[0]
  if (!f) return
  if (f.size > 5 * 1024 * 1024) { ElMessage.error('图片不超过5MB'); return }
  try {
    const fd = new FormData(); fd.append('avatar', f)
    const res = await uploadAvatar(fd)
    store.commit('setUserInfo', { ...store.state.userInfo, avatar: res.avatar })
    ElMessage.success('头像已更新')
  } catch {} finally { e.target.value = '' }
}

async function changePassword() {
  const ok = await pwdFormRef.value?.validate().catch(() => false)
  if (!ok) return
  changingPwd.value = true
  try {
    await updatePassword({ oldPassword: pwdForm.oldPassword, newPassword: pwdForm.newPassword })
    ElMessage.success('密码已修改')
    Object.assign(pwdForm, { oldPassword: '', newPassword: '', confirmPassword: '' })
  } catch {} finally { changingPwd.value = false }
}

onMounted(() => { initForm(); loadStats() })
</script>

<style scoped>
/* Info card */
.info-card {
  position: relative;
  overflow: hidden;
  background: var(--cb-surface);
  border: 1px solid var(--cb-border);
  border-radius: var(--cb-radius-xl);
  box-shadow: var(--cb-shadow-xs);
  margin-bottom: 20px;
}
.info-bg {
  position: absolute; inset: 0;
  background: radial-gradient(ellipse 80% 140% at 30% -20%, rgba(47,107,255,.04), transparent),
              radial-gradient(ellipse 50% 100% at 85% 120%, rgba(139,92,255,.03), transparent);
  pointer-events: none;
}
.info-body {
  position: relative;
  display: flex;
  align-items: center;
  gap: 32px;
  padding: 36px 40px;
}

.avatar-stack { position: relative; cursor: pointer; flex-shrink: 0; }
.avatar-ring { padding: 4px; border-radius: 50%; background: var(--cb-primary-gradient); }
.hero-avatar { border: 3px solid #fff; }
.avatar-badge {
  position: absolute; bottom: 0; right: -2px;
  width: 28px; height: 28px;
  border-radius: 50%;
  background: var(--cb-text);
  color: #fff;
  display: flex; align-items: center; justify-content: center;
  border: 3px solid #fff;
  transition: background .2s;
}
.avatar-stack:hover .avatar-badge { background: var(--cb-primary); }

.info-main { flex: 1; min-width: 0; }
.info-name { font-size: 22px; font-weight: 800; color: var(--cb-text); margin: 0 0 12px; letter-spacing: -0.4px; }
.info-tags { display: flex; flex-wrap: wrap; gap: 8px; }
.info-tag {
  display: inline-flex; align-items: center; gap: 6px;
  font-size: 13px; color: var(--cb-text-secondary);
  background: var(--cb-bg); padding: 5px 14px; border-radius: 99px; font-weight: 500;
}
.info-tag .el-icon { color: var(--cb-text-muted); }

.info-stats {
  display: flex;
  gap: 32px;
  padding-left: 32px;
  border-left: 1px solid var(--cb-border);
  flex-shrink: 0;
}
.is-item { text-align: right; }
.is-val {
  display: block;
  font-size: 24px;
  font-weight: 800;
  color: var(--cb-text);
  letter-spacing: -0.5px;
  line-height: 1.2;
}
.is-val small { font-size: 14px; font-weight: 600; color: var(--cb-text-muted); }
.is-lbl {
  display: block;
  font-size: 12px;
  color: var(--cb-text-muted);
  font-weight: 500;
  margin-top: 2px;
}

/* Card */
.card {
  background: var(--cb-surface);
  border: 1px solid var(--cb-border);
  border-radius: var(--cb-radius-lg);
  padding: 28px 32px;
  box-shadow: var(--cb-shadow-xs);
  margin-bottom: 20px;
}
.card-title {
  display: flex; align-items: center; gap: 8px;
  font-size: 15px; font-weight: 700; color: var(--cb-text); margin: 0 0 24px;
}
.card-title .el-icon { color: var(--cb-primary); }

/* Edit form */
.edit-form :deep(.el-form-item) { margin-bottom: 20px; }
.edit-form :deep(.el-form-item__label) { font-weight: 600; color: var(--cb-text-secondary); font-size: 13px; }
.form-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 0 32px;
}
.form-actions { margin-top: 24px; padding-top: 20px; border-top: 1px solid var(--cb-border-light); }

.edit-form :deep(.el-input__wrapper) {
  border-radius: var(--cb-radius-sm); background: var(--cb-bg);
  box-shadow: none !important; border: 1px solid var(--cb-border);
}
.edit-form :deep(.el-input__wrapper:hover) { border-color: var(--cb-border-strong); }
.edit-form :deep(.el-input__wrapper.is-focus) {
  border-color: var(--cb-primary); background: var(--cb-surface);
  box-shadow: var(--cb-focus-ring) !important;
}
.edit-form :deep(.el-input.is-disabled .el-input__wrapper) { background: var(--cb-bg-alt); opacity: .7; }

/* Cards row */
.cards-row {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 20px;
}
.cards-row .card { margin-bottom: 0; }

/* Storage */
.storage-inline { display: flex; gap: 36px; align-items: center; }
.ring-wrap { position: relative; flex-shrink: 0; }
.ring-svg { width: 150px; height: 150px; transform: rotate(-90deg); }
.ring-bg { fill: none; stroke: var(--cb-bg-alt); stroke-width: 9; }
.ring-fill { fill: none; stroke-width: 9; stroke-linecap: round; stroke-dasharray: 364.42; transition: stroke-dashoffset .8s var(--cb-ease), stroke .4s; }
.ring-center { position: absolute; inset: 0; display: flex; flex-direction: column; align-items: center; justify-content: center; }
.ring-center strong { font-size: 26px; font-weight: 800; color: var(--cb-text); letter-spacing: -1px; line-height: 1; }
.ring-center small { font-size: 11px; color: var(--cb-text-muted); font-weight: 500; margin-top: 2px; }

.type-bars { flex: 1; display: grid; gap: 12px; }
.tb-row { display: grid; grid-template-columns: 8px 36px 44px 1fr; align-items: center; gap: 10px; }
.tb-dot { width: 8px; height: 8px; border-radius: 50%; }
.tb-label { font-size: 12px; font-weight: 600; color: var(--cb-text-secondary); }
.tb-size { font-size: 12px; font-weight: 600; color: var(--cb-text); }
.tb-track { height: 5px; background: var(--cb-bg-alt); border-radius: 99px; overflow: hidden; }
.tb-fill { height: 100%; border-radius: 99px; transition: width .6s var(--cb-ease); }

/* Password form */
.styled-form :deep(.el-form-item) { margin-bottom: 18px; }
.styled-form :deep(.el-form-item:last-child) { margin-bottom: 0; margin-top: 6px; }
.styled-form :deep(.el-input__wrapper) {
  border-radius: var(--cb-radius-sm); background: var(--cb-bg);
  box-shadow: none !important; border: 1px solid var(--cb-border);
}
.styled-form :deep(.el-input__wrapper:hover) { border-color: var(--cb-border-strong); }
.styled-form :deep(.el-input__wrapper.is-focus) {
  border-color: var(--cb-primary); background: var(--cb-surface);
  box-shadow: var(--cb-focus-ring) !important;
}
.styled-form :deep(.el-form-item__label) { font-weight: 600; color: var(--cb-text-secondary); font-size: 13px; }

@media (max-width: 900px) {
  .info-body { flex-direction: column; text-align: center; align-items: center; }
  .info-stats { padding-left: 0; border-left: 0; padding-top: 24px; border-top: 1px solid var(--cb-border); }
  .is-item { text-align: center; }
  .cards-row { grid-template-columns: 1fr; }
}
@media (max-width: 640px) {
  .form-grid { grid-template-columns: 1fr; }
  .storage-inline { flex-direction: column; }
}
</style>
