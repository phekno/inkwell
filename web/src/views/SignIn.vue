<script setup lang="ts">
import { ref } from 'vue'
import { useRouter, RouterLink } from 'vue-router'
import { useAuthStore } from '../stores/auth'

const email = ref('')
const password = ref('')
const error = ref('')
const submitting = ref(false)

const auth = useAuthStore()
const router = useRouter()

async function submit() {
  error.value = ''
  submitting.value = true
  try {
    await auth.signIn(email.value.trim().toLowerCase(), password.value)
    router.push('/entries')
  } catch (e: any) {
    error.value = e?.message ?? 'sign-in failed'
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <section class="flex items-center justify-center px-6 py-12">
    <form @submit.prevent="submit" class="w-full max-w-sm space-y-4">
      <h2 class="text-2xl font-medium">Sign in</h2>

      <label class="block">
        <span class="text-sm opacity-70">Email</span>
        <input
          v-model="email" type="email" autocomplete="email" required
          class="mt-1 w-full rounded-md px-3 py-2 bg-transparent border border-ink-100 dark:border-ink-800 focus:outline-none focus:ring-2 focus:ring-ink-800 dark:focus:ring-ink-100"
        />
      </label>

      <label class="block">
        <span class="text-sm opacity-70">Password</span>
        <input
          v-model="password" type="password" autocomplete="current-password" required
          class="mt-1 w-full rounded-md px-3 py-2 bg-transparent border border-ink-100 dark:border-ink-800 focus:outline-none focus:ring-2 focus:ring-ink-800 dark:focus:ring-ink-100"
        />
      </label>

      <p v-if="error" class="text-sm text-red-600 dark:text-red-400">{{ error }}</p>

      <button
        type="submit" :disabled="submitting"
        class="w-full rounded-md px-3 py-2 bg-ink-800 dark:bg-ink-100 text-ink-50 dark:text-ink-900 hover:opacity-90 transition disabled:opacity-50"
      >{{ submitting ? 'signing in…' : 'sign in' }}</button>

      <p class="text-sm opacity-70 text-center">
        No account? <RouterLink to="/sign-up" class="underline">sign up</RouterLink>
      </p>
    </form>
  </section>
</template>
