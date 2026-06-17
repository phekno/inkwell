<script setup lang="ts">
import { RouterLink, RouterView, useRouter } from 'vue-router'
import { useAuthStore } from './stores/auth'
import DarkModeToggle from './components/DarkModeToggle.vue'

const auth = useAuthStore()
const router = useRouter()

async function signOut() {
  await auth.signOut()
  router.push('/sign-in')
}
</script>

<template>
  <main class="h-dvh flex flex-col overflow-hidden">
    <header class="shrink-0 flex items-center justify-between px-6 py-4 border-b border-ink-100 dark:border-ink-800">
      <RouterLink to="/" class="text-lg font-bold tracking-tight">inkwell</RouterLink>
      <div class="flex items-center gap-3">
        <template v-if="auth.isSignedIn">
          <span class="text-sm opacity-70">{{ auth.email }}</span>
          <button class="btn-term text-sm" @click="signOut">[ sign out ]</button>
        </template>
        <DarkModeToggle />
      </div>
    </header>

    <div class="flex-1 min-h-0 overflow-hidden">
      <RouterView />
    </div>
  </main>
</template>
