<script setup lang="ts">
import { ref, watchEffect } from 'vue'

const isDark = ref(document.documentElement.classList.contains('dark'))

watchEffect(() => {
  document.documentElement.classList.toggle('dark', isDark.value)
  localStorage.setItem('inkwell-theme', isDark.value ? 'dark' : 'light')
})
</script>

<template>
  <button
    type="button"
    class="rounded-md px-3 py-1.5 text-sm border border-ink-100 dark:border-ink-800 hover:bg-ink-100 dark:hover:bg-ink-800 transition"
    :aria-pressed="isDark"
    @click="isDark = !isDark"
  >
    {{ isDark ? '☾ dark' : '☀ light' }}
  </button>
</template>
