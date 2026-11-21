import { createApp, h } from 'vue'
import { createPinia } from 'pinia'
import { ElConfigProvider } from 'element-plus'
import zhCn from 'element-plus/es/locale/lang/zh-cn'
import 'element-plus/es/components/base/style/css'
import App from './App.vue'
import router from './router'

const app = createApp({
  render: () => h(ElConfigProvider, { locale: zhCn }, { default: () => h(App) })
})
app.use(createPinia())
app.use(router)
app.mount('#app')
