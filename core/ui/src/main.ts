import { createApp } from 'vue'
import App from './App.vue'
import router from './router'
import './style.css'
import { applyAppearance } from './appearance'

applyAppearance()
createApp(App).use(router).mount('#app')
