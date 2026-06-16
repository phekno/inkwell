<script setup lang="ts">
import { inject, ref } from 'vue'
import type { TreeNode } from '../lib/tree'
import { NewKey, SelectKey, SelectedIdKey } from './folderTreeKeys'

const props = defineProps<{ node: TreeNode; depth: number }>()

const open = ref(false)
const select = inject(SelectKey)!
const newEntry = inject(NewKey)!
const selectedId = inject(SelectedIdKey)!

function pad(depth: number): string {
  return `${depth * 12 + 12}px`
}
</script>

<template>
  <div>
    <button
      class="w-full text-left px-3 py-1.5 flex items-center gap-1 hover:bg-ink-100/50 dark:hover:bg-ink-800/50"
      :style="{ paddingLeft: pad(depth) }"
      @click="open = !open"
    >
      <span class="opacity-50 text-xs w-3">{{ open ? '▾' : '▸' }}</span>
      <svg
        class="w-3.5 h-3.5 shrink-0 opacity-60"
        viewBox="0 0 24 24" fill="none" stroke="currentColor"
        stroke-width="2" stroke-linecap="round" stroke-linejoin="round"
        aria-hidden="true"
      >
        <path v-if="open" d="M3 7a2 2 0 0 1 2-2h4l2 2h8a2 2 0 0 1 2 2v1H6l-3 7z" />
        <path v-else d="M3 7a2 2 0 0 1 2-2h4l2 2h8a2 2 0 0 1 2 2v8a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z" />
      </svg>
      <span class="truncate font-medium">{{ node.name }}</span>
    </button>

    <template v-if="open">
      <FolderTreeNode
        v-for="f in node.folders"
        :key="f.path"
        :node="f"
        :depth="depth + 1"
      />
      <button
        v-for="e in node.entries"
        :key="e.id"
        class="w-full text-left px-3 py-1.5 truncate hover:bg-ink-100/50 dark:hover:bg-ink-800/50"
        :class="selectedId === e.id ? 'bg-ink-100 dark:bg-ink-800' : ''"
        :style="{ paddingLeft: pad(depth + 1) }"
        @click="select(e)"
      >{{ e.title }}</button>
      <button
        class="w-full text-left px-3 py-1 text-xs opacity-50 hover:opacity-100"
        :style="{ paddingLeft: pad(depth + 1) }"
        @click="newEntry(node.path)"
      >+ new here</button>
    </template>
  </div>
</template>
