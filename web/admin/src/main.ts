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
import '@koris/theme/styles/components.css'
import './styles/micro-interactions.css'
import '@koris/theme/styles/polish.css'
// UI/UX overhaul layer — must load last so it wins the cascade across all tabs.
import '@koris/theme/styles/overhaul.css'

// Import i18n to register admin translation messages with the shared i18n system
import './i18n'

const app = createApp(App)
app.use(createPinia())
app.use(router)
app.mount('#app')
