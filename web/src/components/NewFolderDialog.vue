<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { normalizeFolder } from '../lib/tree'

// Folders only exist as entry paths server-side, so "creating" one just picks
// a path; the parent view shows it in the tree and opens a new entry there.
const props = defineProps<{ parent: string }>()
const emit = defineEmits<{
  create: [folder: string]
  cancel: []
}>()

const name = ref('')
const input = ref<HTMLInputElement>()
const path = computed(() => normalizeFolder(`${props.parent}/${name.value}`))
const valid = computed(() => normalizeFolder(name.value) !== '')

function confirm() {
  if (valid.value) emit('create', path.value)
}

onMounted(() => input.value?.focus())
</script>

<template>
  <div
    class="fixed inset-0 bg-black/40 flex items-center justify-center z-10"
    @click.self="emit('cancel')"
  >
    <div class="pane bg-ink-50 dark:bg-ink-900 p-5 w-80">
      <h3 class="text-sm font-bold mb-3"><span class="prompt-accent">›</span> new folder</h3>
      <p v-if="parent" class="text-xs opacity-60 mb-2">in {{ parent }}</p>
      <input
        ref="input"
        v-model="name"
        placeholder="name (a/b for nested)"
        class="input-term w-full text-sm"
        @keydown.enter="confirm"
        @keydown.esc="emit('cancel')"
      />
      <p class="text-xs opacity-60 mt-2">
        Opens a new entry in it; the folder sticks once that entry is saved.
      </p>
      <div class="flex gap-2 mt-4">
        <button class="btn-term text-sm" :disabled="!valid" @click="confirm">[ create ]</button>
        <button class="btn-term text-sm" @click="emit('cancel')">[ cancel ]</button>
      </div>
    </div>
  </div>
</template>
