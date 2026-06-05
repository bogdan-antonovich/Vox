import { createApp } from 'vue'
import './style.css'
import App from './App.vue'
import router from './router'
import { auth } from './store'
import { userApi } from './api'

const app = createApp(App)
app.use(router)

userApi.info()
  .then(({ data }) => { auth.user = data })
  .catch(() => { auth.user = null })
  .finally(() => { auth.loading = false })

app.mount('#app')
