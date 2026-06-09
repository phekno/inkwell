import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from './stores/auth'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', redirect: '/entries' },
    { path: '/sign-in', component: () => import('./views/SignIn.vue'), meta: { guest: true } },
    { path: '/sign-up', component: () => import('./views/SignUp.vue'), meta: { guest: true } },
    { path: '/confirm', component: () => import('./views/ConfirmSignUp.vue'), meta: { guest: true } },
    { path: '/entries', component: () => import('./views/Entries.vue'), meta: { auth: true } },
  ],
})

router.beforeEach(async (to) => {
  const auth = useAuthStore()
  if (!auth.ready) await auth.hydrate()
  if (to.meta.auth && !auth.isSignedIn) return '/sign-in'
  if (to.meta.guest && auth.isSignedIn) return '/entries'
})

export default router
