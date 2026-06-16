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
