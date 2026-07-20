import { createApp } from 'vue'
import { createPinia } from 'pinia'
import router from '@/router/ops'
import App from './App.vue'
import '../style.css'
import { initTheme } from '../composables/useTheme'
import { registerTDesign } from '../tdesign'

initTheme()

const app = createApp(App)
app.use(createPinia())
registerTDesign(app)
app.use(router)
app.mount('#app')
