import { createApp } from 'vue'
import ElementPlus from 'element-plus'
import en from 'element-plus/es/locale/lang/en'
import 'element-plus/dist/index.css'
import * as ElementPlusIconsVue from '@element-plus/icons-vue'
import App from './App.vue'
import router from './router'

const app = createApp(App)

for (const [key, component] of Object.entries(ElementPlusIconsVue)) {
  app.component(key, component)
}

app.use(ElementPlus, { locale: en })
app.use(router)
app.mount('#app')
