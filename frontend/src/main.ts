import { createApp } from 'vue'
import App from './App.vue'
import SelfCheckPage from './components/SelfCheckPage.vue'
import './styles/base.css'

createApp(window.location.pathname === '/self-check' ? SelfCheckPage : App).mount('#app')
