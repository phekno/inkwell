<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { api, type Entry, type EntryMeta } from '../lib/api'

const list = ref<EntryMeta[]>([])
const selected = ref<Entry | null>(null)
const composing = ref(false)
const title = ref('')
const body = ref('')
const error = ref('')
const loading = ref(false)

async function refresh() {
  loading.value = true
  error.value = ''
  try {
    list.value = await api.list()
  } catch (e: any) {
    error.value = e?.message ?? 'failed to load'
  } finally {
    loading.value = false
  }
}

async function open(meta: EntryMeta) {
  composing.value = false
  selected.value = null
  try {
    selected.value = await api.get(meta.id)
  } catch (e: any) {
    error.value = e?.message ?? 'failed to open'
  }
}

function startCompose() {
  composing.value = true
  selected.value = null
  title.value = ''
  body.value = ''
}

async function save() {
  if (!title.value.trim()) return
  try {
    const created = await api.create(title.value.trim(), body.value)
    list.value = [created, ...list.value]
    composing.value = false
    title.value = ''
    body.value = ''
  } catch (e: any) {
    error.value = e?.message ?? 'save failed'
  }
}

async function remove(id: string) {
  try {
    await api.delete(id)
    list.value = list.value.filter((e) => e.id !== id)
    if (selected.value?.id === id) selected.value = null
  } catch (e: any) {
    error.value = e?.message ?? 'delete failed'
  }
}

function fmt(iso: string) {
  return new Date(iso).toLocaleString()
}

onMounted(refresh)
</script>

<template>
  <section class="grid md:grid-cols-[18rem_1fr] h-[calc(100vh-65px)]">
    <aside class="border-r border-ink-100 dark:border-ink-800 overflow-y-auto">
      <div class="p-3 border-b border-ink-100 dark:border-ink-800 flex items-center justify-between">
        <span class="text-sm opacity-70">{{ list.length }} entries</span>
        <button
          class="rounded-md px-3 py-1.5 text-sm bg-ink-800 dark:bg-ink-100 text-ink-50 dark:text-ink-900 hover:opacity-90"
          @click="startCompose"
        >+ new</button>
      </div>

      <p v-if="loading" class="p-4 text-sm opacity-60">loading…</p>
      <p v-else-if="!list.length" class="p-4 text-sm opacity-60">no entries yet</p>

      <ul>
        <li
          v-for="e in list" :key="e.id"
          class="px-3 py-2 border-b border-ink-100 dark:border-ink-800 cursor-pointer hover:bg-ink-100/50 dark:hover:bg-ink-800/50"
          :class="selected?.id === e.id ? 'bg-ink-100 dark:bg-ink-800' : ''"
          @click="open(e)"
        >
          <p class="text-sm font-medium truncate">{{ e.title }}</p>
          <p class="text-xs opacity-60">{{ fmt(e.created_at) }}</p>
        </li>
      </ul>
    </aside>

    <article class="overflow-y-auto p-6">
      <p v-if="error" class="text-sm text-red-600 dark:text-red-400 mb-3">{{ error }}</p>

      <template v-if="composing">
        <input
          v-model="title" placeholder="Title"
          class="w-full bg-transparent text-2xl font-medium border-b border-ink-100 dark:border-ink-800 focus:outline-none pb-2 mb-4"
        />
        <textarea
          v-model="body" placeholder="Write…"
          class="w-full min-h-[60vh] bg-transparent resize-none focus:outline-none leading-relaxed"
        ></textarea>
        <div class="mt-4 flex gap-2">
          <button
            class="rounded-md px-4 py-2 text-sm bg-ink-800 dark:bg-ink-100 text-ink-50 dark:text-ink-900 hover:opacity-90"
            @click="save"
          >save</button>
          <button
            class="rounded-md px-4 py-2 text-sm border border-ink-100 dark:border-ink-800 hover:bg-ink-100 dark:hover:bg-ink-800"
            @click="composing = false"
          >cancel</button>
        </div>
      </template>

      <template v-else-if="selected">
        <div class="flex items-start justify-between mb-2">
          <h2 class="text-2xl font-medium">{{ selected.title }}</h2>
          <button
            class="text-sm opacity-60 hover:opacity-100 hover:text-red-600 dark:hover:text-red-400"
            @click="remove(selected.id)"
          >delete</button>
        </div>
        <p class="text-xs opacity-60 mb-6">{{ fmt(selected.created_at) }}</p>
        <pre class="whitespace-pre-wrap font-sans leading-relaxed">{{ selected.body }}</pre>
      </template>

      <template v-else>
        <p class="opacity-60 text-sm">Pick an entry, or start a new one.</p>
      </template>
    </article>
  </section>
</template>
