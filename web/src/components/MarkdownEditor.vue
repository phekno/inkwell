<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { EditorState } from '@codemirror/state'
import { EditorView, keymap, placeholder as placeholderExt } from '@codemirror/view'
import { defaultKeymap, history, historyKeymap, indentWithTab } from '@codemirror/commands'
import { HighlightStyle, syntaxHighlighting } from '@codemirror/language'
import { markdown, markdownLanguage } from '@codemirror/lang-markdown'
import { tags } from '@lezer/highlight'
import { livePreview } from '../lib/livePreview'

// Single-pane markdown editor: CodeMirror 6 with an Obsidian-style live
// preview (see lib/livePreview.ts). The model value is always plain markdown,
// so entries round-trip unchanged with the TUI.

const props = defineProps<{ modelValue: string; placeholder?: string }>()
const emit = defineEmits<{
  'update:modelValue': [value: string]
  save: []
}>()

const host = ref<HTMLDivElement>()
let view: EditorView | null = null

// Colors come from the terminal theme tokens so light/dark just work.
const highlight = HighlightStyle.define([
  { tag: tags.strong, fontWeight: '700' },
  { tag: tags.emphasis, fontStyle: 'italic' },
  { tag: tags.strikethrough, textDecoration: 'line-through' },
  { tag: tags.link, color: 'var(--term-accent)', textDecoration: 'underline' },
  { tag: tags.url, opacity: '0.6' },
  { tag: [tags.processingInstruction, tags.contentSeparator], opacity: '0.45' },
])

const theme = EditorView.theme({
  '&': { height: '100%', fontSize: '0.875rem' },
  '&.cm-focused': { outline: 'none' },
  '.cm-scroller': { fontFamily: 'inherit', lineHeight: '1.65', overflow: 'auto' },
  '.cm-content': { caretColor: 'var(--term-accent)', padding: '0' },
  '.cm-cursor': { borderLeftColor: 'var(--term-accent)', borderLeftWidth: '2px' },
  '.cm-placeholder': { opacity: '0.5' },
  '&.cm-focused .cm-selectionBackground, .cm-selectionBackground, ::selection': {
    backgroundColor: 'color-mix(in srgb, var(--term-accent) 25%, transparent) !important',
  },
  '.cm-md-h1': { fontSize: '1.35rem', fontWeight: '700', paddingTop: '0.6rem' },
  '.cm-md-h2': { fontSize: '1.15rem', fontWeight: '700', paddingTop: '0.5rem' },
  '.cm-md-h3, .cm-md-h4, .cm-md-h5, .cm-md-h6': { fontWeight: '700', paddingTop: '0.3rem' },
  '.cm-md-quote': { borderLeft: '2px solid var(--term-accent)', paddingLeft: '0.85rem', opacity: '0.85' },
  '.cm-md-codeblock': {
    backgroundColor: 'color-mix(in srgb, var(--term-border) 30%, transparent)',
    paddingLeft: '0.6rem',
  },
  '.cm-md-bullet': { color: 'var(--term-accent)' },
  '.cm-md-inline-code': {
    backgroundColor: 'color-mix(in srgb, var(--term-border) 45%, transparent)',
    borderRadius: '0.2rem',
  },
  '.cm-md-hr': {
    backgroundImage: 'linear-gradient(var(--term-border), var(--term-border))',
    backgroundSize: '100% 1px',
    backgroundPosition: 'center',
    backgroundRepeat: 'no-repeat',
  },
})

onMounted(() => {
  view = new EditorView({
    parent: host.value!,
    state: EditorState.create({
      doc: props.modelValue,
      extensions: [
        history(),
        keymap.of([
          { key: 'Mod-s', preventDefault: true, run: () => (emit('save'), true) },
          ...defaultKeymap,
          ...historyKeymap,
          indentWithTab,
        ]),
        markdown({ base: markdownLanguage }),
        syntaxHighlighting(highlight),
        livePreview,
        EditorView.lineWrapping,
        placeholderExt(props.placeholder ?? ''),
        theme,
        EditorView.updateListener.of((u) => {
          if (u.docChanged) emit('update:modelValue', u.state.doc.toString())
        }),
      ],
    }),
  })
})

// External value changes (not ones we just emitted) replace the document.
watch(
  () => props.modelValue,
  (v) => {
    if (view && v !== view.state.doc.toString()) {
      view.dispatch({ changes: { from: 0, to: view.state.doc.length, insert: v } })
    }
  },
)

onBeforeUnmount(() => view?.destroy())

defineExpose({ focus: () => view?.focus() })
</script>

<template>
  <div ref="host" class="h-full"></div>
</template>
