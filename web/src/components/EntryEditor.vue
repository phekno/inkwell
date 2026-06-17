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
      class="w-full bg-transparent text-2xl font-medium border-b border-ink-100 dark:border-ink-800 focus:outline-none pb-2 mb-4"
    />
    <div class="grid grid-cols-2 gap-4 flex-1 min-h-0">
      <div
        class="min-h-0 rounded-lg border border-ink-100 dark:border-ink-800 p-3 focus-within:border-ink-800 dark:focus-within:border-ink-100 transition"
      >
        <textarea
          v-model="body"
          placeholder="Write…"
          class="w-full h-full bg-transparent resize-none focus:outline-none leading-relaxed font-mono text-sm"
        ></textarea>
      </div>
      <div
        class="prose-ink overflow-y-auto rounded-lg border border-ink-100 dark:border-ink-800 p-4 leading-relaxed"
        v-html="preview"
      ></div>
    </div>
    <div class="mt-4 flex gap-2">
      <button
        class="rounded-md px-4 py-2 text-sm bg-ink-800 dark:bg-ink-100 text-ink-50 dark:text-ink-900 hover:opacity-90 disabled:opacity-50"
        :disabled="saving"
        @click="save"
      >{{ saving ? 'saving…' : 'save' }}</button>
      <button
        class="rounded-md px-4 py-2 text-sm border border-ink-100 dark:border-ink-800 hover:bg-ink-100 dark:hover:bg-ink-800"
        @click="emit('cancel')"
      >cancel</button>
    </div>
  </div>
</template>
