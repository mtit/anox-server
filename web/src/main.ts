import { createApp } from 'vue'
import ArcoVue from '@arco-design/web-vue'
import ArcoVueIcon from '@arco-design/web-vue/es/icon'
import '@arco-design/web-vue/dist/arco.css'
import './styles/responsive.css'

import App from './App.vue'
import router from './router'
import { createPinia } from 'pinia'
import { initI18n } from './i18n'

async function bootstrap() {
  await initI18n()

  const app = createApp(App)

  app.use(createPinia())
  app.use(router)
  app.use(ArcoVue)
  app.use(ArcoVueIcon)

  app.mount('#app')
}

bootstrap()
