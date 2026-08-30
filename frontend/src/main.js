import { createApp } from 'vue'
import App from './App.vue'
import router from './router/index.js'
import axios from 'axios'
import store from './store/index.js'

// 引入全局css
import './assets/cloud-ui.css'
import '@/assets/wxlogin/iconfont.css'
import '@/assets/zfblogin/iconfont.css'
import '@/assets/qqlogin/iconfont.css'

const app = createApp(App)

// 迁移到 HttpOnly Cookie 后，主动清理旧版本遗留的可读 JWT。
localStorage.removeItem('token')
sessionStorage.removeItem('token')

// 全局挂载
app.config.globalProperties.$axios = axios
app.use(router)
app.use(store)
app.mount('#app')
