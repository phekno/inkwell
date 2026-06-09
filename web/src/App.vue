<script setup lang="ts">
import DarkModeToggle from './components/DarkModeToggle.vue'
</script>

<template>
  <main class="min-h-screen flex flex-col">
    <header class="flex items-center justify-between px-6 py-4 border-b border-ink-100 dark:border-ink-800">
      <h1 class="text-xl font-semibold tracking-tight">inkwell</h1>
      <DarkModeToggle />
    </header>

    <section class="flex-1 flex items-center justify-center px-6">
      <div class="max-w-md text-center space-y-3">
        <h2 class="text-2xl font-medium">Your journal, encrypted in the cloud.</h2>
        <p class="opacity-70">Sign in to start writing.</p>
      </div>
    </section>
  </main>
</template>
