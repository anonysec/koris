import { createApp } from 'vue'
import { createPinia } from 'pinia'
import router from './router'
import App from './App.vue'
import '@koris/styles/reset.css'
import '@koris/styles/tokens.css'
import '@koris/styles/transitions.css'
import '@koris/styles/utilities.css'
import '@koris/styles/rtl.css'
import './style.css'
import '@koris/styles/components.css'
import './styles/micro-interactions.css'

// Import i18n to register admin translation messages with the shared i18n system
import './i18n'

const app = createApp(App)
app.use(createPinia())
app.use(router)
app.mount('#app')
