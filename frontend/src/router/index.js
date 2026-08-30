import { createRouter, createWebHistory } from 'vue-router'
import store from '@/store'
import { fetchProfile, refreshSession } from '@/services/authSession'

const routes = [
  {
    path: '/login',
    name: 'Login',
    component: () => import('@/views/Login.vue'),
    meta: { requiresAuth: false, title: '登录' }
  },
  {
    path: '/forgot-password',
    name: 'ForgotPassword',
    component: () => import('@/views/ForgotPassword.vue'),
    meta: { requiresAuth: false, title: '忘记密码' }
  },
  {
    path: '/reset-password',
    name: 'ResetPassword',
    component: () => import('@/views/ResetPassword.vue'),
    meta: { requiresAuth: false, title: '重置密码' }
  },
  {
    path: '/',
    component: () => import('@/views/Container.vue'),
    meta: { requiresAuth: true },
    children: [
      { path: '', name: 'MyDrive', component: () => import('@/views/MyDrive.vue'), meta: { title: '全部文件' } },
      { path: 'recent', name: 'Recent', component: () => import('@/views/RecentFiles.vue'), meta: { title: '最近文件' } },
      { path: 'starred', name: 'Starred', component: () => import('@/views/StarredFiles.vue'), meta: { title: '收藏夹' } },
      { path: 'file', name: 'FileCategory', component: () => import('@/views/FileCategory.vue'), meta: { title: '文件分类' } },
      { path: 'file/:type', name: 'FileCategoryType', component: () => import('@/views/FileCategory.vue'), meta: { title: '文件分类' } },
      { path: 'shared', name: 'Shared', component: () => import('@/views/SharedFiles.vue'), meta: { title: '我的分享' } },
      { path: 'recycle', name: 'Recycle', component: () => import('@/views/Recycle.vue'), meta: { title: '回收站' } },
      { path: 'duplicates', name: 'Duplicates', component: () => import('@/views/DuplicateFiles.vue'), meta: { title: '重复文件' } },
      { path: 'user', name: 'UserProfile', component: () => import('@/views/UserProfile.vue'), meta: { title: '个人中心' } }
    ]
  },
  {
    path: '/s/:token',
    name: 'ShareLink',
    component: () => import('@/views/ShareLink.vue'),
    meta: { title: '文件分享', requiresAuth: false }
  },
  {
    path: '/:pathMatch(.*)*',
    name: 'NotFound',
    component: () => import('@/views/NotFound.vue'),
    meta: { title: '页面不存在', requiresAuth: false }
  }
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

async function ensureAuthenticated() {
  if (store.state.authChecked) {
    return store.state.isAuthenticated
  }

  try {
    const profile = await fetchProfile()
    store.commit('setUserInfo', profile)
    store.commit('setAuthChecked', true)
    return true
  } catch {
    try {
      await refreshSession()
      const profile = await fetchProfile()
      store.commit('setUserInfo', profile)
      store.commit('setAuthChecked', true)
      return true
    } catch {
      store.commit('clearAuth')
      return false
    }
  }
}

router.beforeEach(async (to, from, next) => {
  document.title = to.meta.title ? `${to.meta.title} - CloudBox` : 'CloudBox'
  const requiresAuth = to.matched.some(record => record.meta.requiresAuth)
  const shouldCheckAuth = requiresAuth || to.path === '/login'
  const authenticated = shouldCheckAuth ? await ensureAuthenticated() : store.state.isAuthenticated

  if (requiresAuth && !authenticated) {
    next({ path: '/login' })
  } else if (to.path === '/login' && authenticated) {
    next({ path: '/' })
  } else {
    next()
  }
})

export default router
