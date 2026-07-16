import { createApp } from 'vue'
import { createPinia } from 'pinia'
import router from '@/router/ops'
import App from './App.vue'
import { registerTDesign } from '../tdesign'

const app = createApp(App)
app.use(createPinia())
registerTDesign(app)
app.use(router)
app.mount('#app')
