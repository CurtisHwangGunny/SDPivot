import { createApp } from 'vue'
import { createPinia } from 'pinia'
import App from './App.vue'
import router from './router'
import './style.css'
import { initTheme } from './composables/useTheme'
import { registerTDesign } from './tdesign'
import { migrateLegacyStorage } from './utils/storage'

migrateLegacyStorage()
initTheme()

const app = createApp(App)
app.use(createPinia())
app.use(router)
registerTDesign(app)
app.mount('#app')
