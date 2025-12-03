import { createRouter, createWebHistory } from 'vue-router'
import Container from '@/views/Container.vue'
import { ElMessage } from 'element-plus'
import store from '@/store'

const routes = [
  {
    path: '/login',
    name: 'Login',
    component: () => import('@/views/Login.vue'),
    meta: { requiresAuth: false, title: '登录' }
  },
  {
    path: '/',
    component: Container,
    meta: { requiresAuth: true },
    children: [
        {path: '', name: 'MyDrive', component: () => import('@/views/MyDrive.vue'), meta: {title: '全部文件'}},
        {path: 'recent', name: 'Recent', component: () => import('@/views/RecentFiles.vue'), meta: {title: '最近文件'}},
        {
            path: 'starred',
            name: 'Starred',
            component: () => import('@/views/StarredFiles.vue'),
            meta: {title: '收藏夹'}
        },
        {
            path: 'file',
            name: 'FileCategory',
            component: () => import('@/views/FileCategory.vue'),
            meta: {title: '文件分类', icon: 'FolderOpened'}
        },
        {
            path: 'file/:type',
            name: 'FileCategoryType',
            component: () => import('@/views/FileCategory.vue'),
            meta: {title: '文件分类'}
        },
        {path: 'shared', name: 'Shared', component: () => import('@/views/SharedFiles.vue'), meta: {title: '我的分享'}},
        {path: 'recycle', name: 'Recycle', component: () => import('@/views/Recycle.vue'), meta: {title: '回收站'}},
        {
            path: 'user',
            name: 'UserProfile',
            component: () => import('@/views/UserProfile.vue'),
            meta: {title: '个人中心'}
        },
        // {path: 'settings', name: 'Settings', component: () => import('@/views/Settings.vue'), meta: {title: '设置'}}
    ]
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

function setPageTitle(title) {
  document.title = title ? `${title} - 云网盘` : '云网盘'
}

router.beforeEach((to, from, next) => {
  setPageTitle(to.meta.title)
  const requiresAuth = to.matched.some(record => record.meta.requiresAuth)
  const token = store.state.token || localStorage.getItem('token')

  if (requiresAuth && !token) {
    ElMessage.warning('请先登录')
    next({ path: '/login' })
  } else if (to.path === '/login' && token) {
    ElMessage.info('您已登录')
    next({ path: '/' })
  } else {
    next()
  }
})

export default router
