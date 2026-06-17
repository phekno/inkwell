<script setup lang="ts">
import { computed, ref } from 'vue'
import { renderMarkdown } from '../lib/markdown'

const props = defineProps<{
  initialTitle?: string
  initialBody?: string
  saving?: boolean
}>()
const emit = defineEmits<{
  save: [payload: { title: string; body: string }]
  cancel: []
}>()

const title = ref(props.initialTitle ?? '')
const body = ref(props.initialBody ?? '')
const preview = computed(() => renderMarkdown(body.value))

function save() {
  if (!title.value.trim() || props.saving) return
  emit('save', { title: title.value.trim(), body: body.value })
}
</script>

<template>
  <div class="flex flex-col h-full">
    <input
      v-model="title"
      placeholder="Title"
      class="input-term w-full text-lg font-bold mb-4"
    />
    <div class="grid grid-cols-2 gap-4 flex-1 min-h-0">
      <div class="flex flex-col min-h-0">
        <span class="text-xs opacity-60 mb-1 prompt-accent">› body</span>
        <div class="pane flex-1 min-h-0 p-3 focus-within:border-[var(--term-accent)] transition">
          <textarea
            v-model="body"
            placeholder="Write…"
            class="w-full h-full bg-transparent resize-none focus:outline-none leading-relaxed text-sm"
          ></textarea>
        </div>
      </div>
      <div class="flex flex-col min-h-0">
        <span class="text-xs opacity-60 mb-1">preview</span>
        <div class="pane prose-ink flex-1 min-h-0 overflow-y-auto p-4 leading-relaxed" v-html="preview"></div>
      </div>
    </div>
    <div class="mt-4 flex gap-2">
      <button class="btn-term text-sm" :disabled="saving" @click="save">
        {{ saving ? '[ saving… ]' : '[ save ]' }}
      </button>
      <button class="btn-term text-sm" @click="emit('cancel')">[ cancel ]</button>
    </div>
  </div>
</template>
