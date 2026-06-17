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
    class="btn-term text-sm"
    :aria-pressed="isDark"
    @click="isDark = !isDark"
  >
    {{ isDark ? '[ ☾ dark ]' : '[ ☀ light ]' }}
  </button>
</template>
