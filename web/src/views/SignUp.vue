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
    <form @submit.prevent="submit" class="pane w-full max-w-sm space-y-4 p-6">
      <h2 class="text-xl font-bold"><span class="prompt-accent">›</span> create an account</h2>

      <label class="block">
        <span class="text-sm opacity-70">Email</span>
        <input
          v-model="email" type="email" autocomplete="email" required
          class="input-term mt-1 w-full"
        />
      </label>

      <label class="block">
        <span class="text-sm opacity-70">Password</span>
        <input
          v-model="password" type="password" autocomplete="new-password" required minlength="12"
          class="input-term mt-1 w-full"
        />
        <span class="text-xs opacity-60">12+ chars, upper, lower, and a number.</span>
      </label>

      <p v-if="error" class="text-sm text-red-600 dark:text-red-400">{{ error }}</p>

      <button
        type="submit" :disabled="submitting"
        class="btn-term w-full"
      >{{ submitting ? '[ creating… ]' : '[ create account ]' }}</button>

      <p class="text-sm opacity-70 text-center">
        Already have one? <RouterLink to="/sign-in" class="underline">sign in</RouterLink>
      </p>
    </form>
  </section>
</template>
