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
  <main class="min-h-screen flex flex-col">
    <header class="flex items-center justify-between px-6 py-4 border-b border-ink-100 dark:border-ink-800">
      <RouterLink to="/" class="text-xl font-semibold tracking-tight">inkwell</RouterLink>
      <div class="flex items-center gap-3">
        <template v-if="auth.isSignedIn">
          <span class="text-sm opacity-70">{{ auth.email }}</span>
          <button
            class="rounded-md px-3 py-1.5 text-sm border border-ink-100 dark:border-ink-800 hover:bg-ink-100 dark:hover:bg-ink-800 transition"
            @click="signOut"
          >sign out</button>
        </template>
        <DarkModeToggle />
      </div>
    </header>

    <RouterView class="flex-1" />
  </main>
</template>
