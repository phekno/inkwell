<script setup lang="ts">
import { ref } from 'vue'
import { useRouter, RouterLink } from 'vue-router'
import { signUp } from '../lib/cognito'

const email = ref('')
const password = ref('')
const error = ref('')
const submitting = ref(false)

const router = useRouter()

async function submit() {
  error.value = ''
  submitting.value = true
  try {
    await signUp(email.value.trim().toLowerCase(), password.value)
    router.push({ path: '/confirm', query: { email: email.value.trim().toLowerCase() } })
  } catch (e: any) {
    error.value = e?.message ?? 'sign-up failed'
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <section class="h-full overflow-y-auto flex items-center justify-center px-6 py-12">
    <form @submit.prevent="submit" class="w-full max-w-sm space-y-4">
      <h2 class="text-2xl font-medium">Create an account</h2>

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
          v-model="password" type="password" autocomplete="new-password" required minlength="12"
          class="mt-1 w-full rounded-md px-3 py-2 bg-transparent border border-ink-100 dark:border-ink-800 focus:outline-none focus:ring-2 focus:ring-ink-800 dark:focus:ring-ink-100"
        />
        <span class="text-xs opacity-60">12+ chars, upper, lower, and a number.</span>
      </label>

      <p v-if="error" class="text-sm text-red-600 dark:text-red-400">{{ error }}</p>

      <button
        type="submit" :disabled="submitting"
        class="w-full rounded-md px-3 py-2 bg-ink-800 dark:bg-ink-100 text-ink-50 dark:text-ink-900 hover:opacity-90 transition disabled:opacity-50"
      >{{ submitting ? 'creating…' : 'create account' }}</button>

      <p class="text-sm opacity-70 text-center">
        Already have one? <RouterLink to="/sign-in" class="underline">sign in</RouterLink>
      </p>
    </form>
  </section>
</template>
