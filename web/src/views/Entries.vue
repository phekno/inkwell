<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { api, type Entry, type EntryMeta } from '../lib/api'
import { buildTree, folderPaths } from '../lib/tree'
import { renderMarkdown } from '../lib/markdown'
import FolderTree from '../components/FolderTree.vue'
import EntryEditor from '../components/EntryEditor.vue'
import MoveDialog from '../components/MoveDialog.vue'

type Mode = 'view' | 'compose' | 'edit'

const list = ref<EntryMeta[]>([])
const selected = ref<Entry | null>(null)
const mode = ref<Mode>('view')
const composeFolder = ref('')
const showMove = ref(false)
const error = ref('')
const loading = ref(false)
const saving = ref(false)

const tree = computed(() => buildTree(list.value))
const folders = computed(() => folderPaths(tree.value))
const rendered = computed(() => (selected.value ? renderMarkdown(selected.value.body) : ''))

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
  mode.value = 'view'
  selected.value = null
  error.value = ''
  try {
    selected.value = await api.get(meta.id)
  } catch (e: any) {
    error.value = e?.message ?? 'failed to open'
  }
}

function startCompose(folder: string) {
  composeFolder.value = folder
  selected.value = null
  mode.value = 'compose'
}

function startEdit() {
  if (selected.value) mode.value = 'edit'
}

async function saveCompose(payload: { title: string; body: string }) {
  saving.value = true
  error.value = ''
  try {
    const created = await api.create(payload.title, payload.body, composeFolder.value)
    list.value = [created, ...list.value]
    mode.value = 'view'
  } catch (e: any) {
    error.value = e?.message ?? 'save failed'
  } finally {
    saving.value = false
  }
}

async function saveEdit(payload: { title: string; body: string }) {
  if (!selected.value) return
  const id = selected.value.id
  saving.value = true
  error.value = ''
  try {
    // PATCH response only carries updated_at reliably (folder comes back ""),
    // so apply the user's values locally and take updated_at from the response.
    const resp = await api.update(id, payload)
    list.value = list.value.map((e) =>
      e.id === id ? { ...e, title: payload.title, updated_at: resp.updated_at } : e,
    )
    selected.value = {
      ...selected.value,
      title: payload.title,
      body: payload.body,
      updated_at: resp.updated_at,
    }
    mode.value = 'view'
  } catch (e: any) {
    error.value = e?.message ?? 'save failed'
  } finally {
    saving.value = false
  }
}

async function doMove(folder: string) {
  if (!selected.value) return
  const id = selected.value.id
  showMove.value = false
  error.value = ''
  try {
    // PATCH-move response returns title "", so trust only folder (local) + updated_at.
    const resp = await api.move(id, folder)
    list.value = list.value.map((e) =>
      e.id === id ? { ...e, folder, updated_at: resp.updated_at } : e,
    )
    selected.value = { ...selected.value, folder, updated_at: resp.updated_at }
  } catch (e: any) {
    error.value = e?.message ?? 'move failed'
  }
}

async function remove(id: string) {
  error.value = ''
  try {
    await api.delete(id)
    list.value = list.value.filter((e) => e.id !== id)
    if (selected.value?.id === id) {
      selected.value = null
      mode.value = 'view'
    }
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
  <section class="h-full grid md:grid-cols-[20rem_1fr] grid-rows-1 overflow-hidden">
    <aside class="border-r border-ink-100 dark:border-ink-800 overflow-y-auto min-h-0">
      <div class="p-3 border-b border-ink-100 dark:border-ink-800 flex items-center justify-between">
        <span class="text-sm opacity-70">{{ list.length }} entries</span>
        <button class="btn-term text-sm" @click="startCompose('')">[ + new ]</button>
      </div>

      <p v-if="loading" class="p-4 text-sm opacity-60">loading…</p>
      <p v-else-if="!list.length" class="p-4 text-sm opacity-60">no entries yet</p>

      <FolderTree
        v-else
        :tree="tree"
        :selected-id="selected?.id ?? null"
        @select="open"
        @new-entry="startCompose"
      />
    </aside>

    <article class="overflow-y-auto min-h-0 p-6">
      <p v-if="error" class="text-sm text-red-600 dark:text-red-400 mb-3">{{ error }}</p>

      <EntryEditor
        v-if="mode === 'compose'"
        :saving="saving"
        @save="saveCompose"
        @cancel="mode = 'view'"
      />

      <EntryEditor
        v-else-if="mode === 'edit' && selected"
        :initial-title="selected.title"
        :initial-body="selected.body"
        :saving="saving"
        @save="saveEdit"
        @cancel="mode = 'view'"
      />

      <template v-else-if="selected">
        <div class="flex items-start justify-between mb-2 gap-3">
          <h2 class="text-xl font-bold"><span class="prompt-accent">#</span> {{ selected.title }}</h2>
          <div class="shrink-0 flex gap-2">
            <button class="btn-term text-sm" @click="startEdit">[ edit ]</button>
            <button class="btn-term text-sm" @click="showMove = true">[ move ]</button>
            <button
              class="btn-term text-sm text-red-700 dark:text-red-300 hover:!text-red-600 hover:!border-red-600"
              @click="remove(selected.id)"
            >[ delete ]</button>
          </div>
        </div>
        <p class="text-xs opacity-60 mb-6">
          {{ fmt(selected.created_at) }}
          <span v-if="selected.folder" class="ml-2">· {{ selected.folder }}</span>
        </p>
        <div class="prose-ink leading-relaxed" v-html="rendered"></div>
      </template>

      <template v-else>
        <p class="opacity-60 text-sm">Pick an entry, or start a new one.</p>
      </template>
    </article>

    <MoveDialog
      v-if="showMove && selected"
      :folders="folders"
      :current="selected.folder"
      @move="doMove"
      @cancel="showMove = false"
    />
  </section>
</template>
