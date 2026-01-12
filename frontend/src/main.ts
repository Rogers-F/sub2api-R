import { createApp } from 'vue'
import { createPinia } from 'pinia'
import App from './App.vue'
import router from './router'
import i18n from './i18n'
import './style.css'

const app = createApp(App)
const pinia = createPinia()
app.use(pinia)

// Initialize settings from injected config BEFORE mounting (prevents flash)
// This must happen after pinia is installed but before router and i18n
// Note: Backend SSR already handles title and favicon in HTML, so we only need
// to populate the store here for Vue components to use
import { useAppStore } from '@/stores/app'
const appStore = useAppStore()
appStore.initFromInjectedConfig()

app.use(router)
app.use(i18n)

// 等待路由器完成初始导航后再挂载，避免竞态条件导致的空白渲染
router.isReady().then(() => {
  app.mount('#app')
})
