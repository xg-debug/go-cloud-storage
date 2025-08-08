import { createApp } from 'vue'
import App from './App.vue'
import router from './router'
import axios from 'axios'
import store from './store'
import { getProfile } from '@/api/user'

// 引入全局css
import './assets/global.css'
import '@/assets/wxlogin/iconfont.css'
import '@/assets/zfblogin/iconfont.css'
import '@/assets/qqlogin/iconfont.css'

import * as ElementPlusIconsVue from '@element-plus/icons-vue'

const app = createApp(App)

// 注册所有图标
for (const [key, component] of Object.entries(ElementPlusIconsVue)) {
  app.component(key, component)
}

const token = localStorage.getItem('token')

if (token) {
  getProfile()
    .then(res => {
      store.commit('setUserInfo', res)
    })
    .catch(() => {
      // token 无效了，清理
      store.commit('setUserInfo', null)
      localStorage.removeItem('token')
    })
}

// 全局挂载
app.config.globalProperties.$axios = axios
app.use(router)
app.use(store)
app.mount('#app')
