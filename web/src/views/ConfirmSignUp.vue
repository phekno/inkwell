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
    <form @submit.prevent="submit" class="pane w-full max-w-sm space-y-4 p-6">
      <h2 class="text-xl font-bold"><span class="prompt-accent">›</span> confirm your email</h2>
      <p class="text-sm opacity-70">Enter the code we sent to your inbox.</p>

      <label class="block">
        <span class="text-sm opacity-70">Email</span>
        <input
          v-model="email" type="email" required
          class="input-term mt-1 w-full"
        />
      </label>

      <label class="block">
        <span class="text-sm opacity-70">Confirmation code</span>
        <input
          v-model="code" type="text" inputmode="numeric" pattern="[0-9]*" required
          class="input-term mt-1 w-full"
        />
      </label>

      <p v-if="error" class="text-sm text-red-600 dark:text-red-400">{{ error }}</p>
      <p v-if="info" class="text-sm text-green-700 dark:text-green-400">{{ info }}</p>

      <button
        type="submit" :disabled="submitting"
        class="btn-term w-full"
      >{{ submitting ? '[ confirming… ]' : '[ confirm ]' }}</button>

      <p class="text-sm opacity-70 text-center">
        Didn't get it? <button type="button" class="underline" @click="resend">resend</button>
      </p>
    </form>
  </section>
</template>
