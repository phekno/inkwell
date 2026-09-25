<script setup lang="ts">
import { defineAsyncComponent, ref } from 'vue'

// CodeMirror (plus the HTML/JS/CSS parsers lang-markdown embeds) is large, so
// load it only once the user actually opens the editor.
const MarkdownEditor = defineAsyncComponent(() => import('./MarkdownEditor.vue'))

const props = defineProps<{
  initialTitle?: string
  initialBody?: string
  folder?: string
  saving?: boolean
}>()
const emit = defineEmits<{
  save: [payload: { title: string; body: string }]
  cancel: []
}>()

const title = ref(props.initialTitle ?? '')
const body = ref(props.initialBody ?? '')
const editor = ref<{ focus: () => void }>()

function save() {
  if (!title.value.trim() || props.saving) return
  emit('save', { title: title.value.trim(), body: body.value })
}
</script>

<template>
  <div class="flex flex-col h-full">
    <p v-if="folder" class="text-xs opacity-60 mb-2">in {{ folder }}</p>
    <input
      v-model="title"
      placeholder="Title"
      class="input-term w-full text-lg font-bold mb-4"
      @keydown.enter.prevent="editor?.focus()"
      @keydown.meta.s.prevent="save"
      @keydown.ctrl.s.prevent="save"
    />
    <div class="pane flex-1 min-h-0 p-4 focus-within:border-[var(--term-accent)] transition">
      <MarkdownEditor ref="editor" v-model="body" placeholder="Write…" @save="save" />
    </div>
    <div class="mt-4 flex items-center gap-2">
      <button class="btn-term text-sm" :disabled="saving" @click="save">
        {{ saving ? '[ saving… ]' : '[ save ]' }}
      </button>
      <button class="btn-term text-sm" @click="emit('cancel')">[ cancel ]</button>
      <span class="ml-auto text-xs opacity-50">⌘/ctrl+s to save</span>
    </div>
  </div>
</template>
