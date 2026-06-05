import { createRouter, createWebHistory } from 'vue-router'
import { auth } from './store'
import LoginView from './views/LoginView.vue'
import DashboardView from './views/DashboardView.vue'
import BroadcastView from './views/BroadcastView.vue'
import ListenView from './views/ListenView.vue'
import ProfileView from './views/ProfileView.vue'

const router = createRouter({
  history: createWebHistory('/vox/'),
  routes: [
    { path: '/',                       component: LoginView },
    { path: '/host',                   component: DashboardView,  meta: { auth: true } },
    { path: '/host/:id/broadcast',     component: BroadcastView,  meta: { auth: true } },
    { path: '/listen/:id?',            component: ListenView },
    { path: '/profile',                component: ProfileView,    meta: { auth: true } },
  ],
})

router.beforeEach((to) => {
  if (to.meta.auth && !auth.loading && !auth.user) return '/'
})

export default router
