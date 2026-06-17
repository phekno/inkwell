<script setup lang="ts">
import { ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { confirmSignUp, resendConfirmation } from '../lib/cognito'

const route = useRoute()
const router = useRouter()

const email = ref(String(route.query.email ?? ''))
const code = ref('')
const error = ref('')
const info = ref('')
const submitting = ref(false)

async function submit() {
  error.value = ''
  info.value = ''
  submitting.value = true
  try {
    await confirmSignUp(email.value.trim().toLowerCase(), code.value.trim())
    router.push('/sign-in')
  } catch (e: any) {
    error.value = e?.message ?? 'confirmation failed'
  } finally {
    submitting.value = false
  }
}

async function resend() {
  error.value = ''
  info.value = ''
  try {
    await resendConfirmation(email.value.trim().toLowerCase())
    info.value = 'A new code has been sent.'
  } catch (e: any) {
    error.value = e?.message ?? 'resend failed'
  }
}
</script>

<template>
  <section class="h-full overflow-y-auto flex items-center justify-center px-6 py-12">
    <form @submit.prevent="submit" class="w-full max-w-sm space-y-4">
      <h2 class="text-2xl font-medium">Confirm your email</h2>
      <p class="text-sm opacity-70">Enter the code we sent to your inbox.</p>

      <label class="block">
        <span class="text-sm opacity-70">Email</span>
        <input
          v-model="email" type="email" required
          class="mt-1 w-full rounded-md px-3 py-2 bg-transparent border border-ink-100 dark:border-ink-800 focus:outline-none focus:ring-2 focus:ring-ink-800 dark:focus:ring-ink-100"
        />
      </label>

      <label class="block">
        <span class="text-sm opacity-70">Confirmation code</span>
        <input
          v-model="code" type="text" inputmode="numeric" pattern="[0-9]*" required
          class="mt-1 w-full rounded-md px-3 py-2 bg-transparent border border-ink-100 dark:border-ink-800 focus:outline-none focus:ring-2 focus:ring-ink-800 dark:focus:ring-ink-100"
        />
      </label>

      <p v-if="error" class="text-sm text-red-600 dark:text-red-400">{{ error }}</p>
      <p v-if="info" class="text-sm text-green-700 dark:text-green-400">{{ info }}</p>

      <button
        type="submit" :disabled="submitting"
        class="w-full rounded-md px-3 py-2 bg-ink-800 dark:bg-ink-100 text-ink-50 dark:text-ink-900 hover:opacity-90 transition disabled:opacity-50"
      >{{ submitting ? 'confirming…' : 'confirm' }}</button>

      <p class="text-sm opacity-70 text-center">
        Didn't get it? <button type="button" class="underline" @click="resend">resend</button>
      </p>
    </form>
  </section>
</template>
