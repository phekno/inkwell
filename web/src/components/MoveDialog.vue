<script setup lang="ts">
import { ref } from 'vue'

const props = defineProps<{ folders: string[]; current: string }>()
const emit = defineEmits<{
  move: [folder: string]
  cancel: []
}>()

const choice = ref(props.current)
const custom = ref('')

function confirm() {
  emit('move', custom.value.trim() || choice.value)
}
</script>

<template>
  <div
    class="fixed inset-0 bg-black/40 flex items-center justify-center z-10"
    @click.self="emit('cancel')"
  >
    <div class="bg-ink-50 dark:bg-ink-900 border border-ink-100 dark:border-ink-800 rounded-lg p-5 w-80">
      <h3 class="text-sm font-medium mb-3">Move to folder</h3>
      <select
        v-model="choice"
        class="w-full bg-transparent border border-ink-100 dark:border-ink-800 rounded-md px-2 py-1.5 text-sm mb-3"
      >
        <option value="">(root)</option>
        <option v-for="f in folders" :key="f" :value="f">{{ f }}</option>
      </select>
      <input
        v-model="custom"
        placeholder="or type a new folder path"
        class="w-full bg-transparent border border-ink-100 dark:border-ink-800 rounded-md px-2 py-1.5 text-sm"
      />
      <div class="flex gap-2 mt-4">
        <button
          class="rounded-md px-4 py-2 text-sm bg-ink-800 dark:bg-ink-100 text-ink-50 dark:text-ink-900 hover:opacity-90"
          @click="confirm"
        >move</button>
        <button
          class="rounded-md px-4 py-2 text-sm border border-ink-100 dark:border-ink-800 hover:bg-ink-100 dark:hover:bg-ink-800"
          @click="emit('cancel')"
        >cancel</button>
      </div>
    </div>
  </div>
</template>
