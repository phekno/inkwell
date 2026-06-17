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
    <div class="pane bg-ink-50 dark:bg-ink-900 p-5 w-80">
      <h3 class="text-sm font-bold mb-3"><span class="prompt-accent">›</span> move to folder</h3>
      <select
        v-model="choice"
        class="input-term w-full text-sm mb-3"
      >
        <option value="">(root)</option>
        <option v-for="f in folders" :key="f" :value="f">{{ f }}</option>
      </select>
      <input
        v-model="custom"
        placeholder="or type a new folder path"
        class="input-term w-full text-sm"
      />
      <div class="flex gap-2 mt-4">
        <button
          class="btn-term text-sm"
          @click="confirm"
        >[ move ]</button>
        <button
          class="btn-term text-sm"
          @click="emit('cancel')"
        >[ cancel ]</button>
      </div>
    </div>
  </div>
</template>
