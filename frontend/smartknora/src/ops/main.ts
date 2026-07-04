import { createApp } from 'vue'
import { createPinia } from 'pinia'
import TDesign from 'tdesign-vue-next'
import 'tdesign-vue-next/es/style/index.css'
import router from '@/router/ops'
import App from './App.vue'

const app = createApp(App)
app.use(createPinia())
app.use(TDesign)
app.use(router)
app.mount('#app')
