<script setup lang="ts">
import { provide, toRef } from 'vue'
import type { TreeNode } from '../lib/tree'
import type { EntryMeta } from '../lib/api'
import FolderTreeNode from './FolderTreeNode.vue'
import { NewKey, SelectKey, SelectedIdKey } from './folderTreeKeys'

const props = defineProps<{ tree: TreeNode; selectedId: string | null }>()
const emit = defineEmits<{
  select: [entry: EntryMeta]
  newEntry: [folder: string]
}>()

provide(SelectKey, (e: EntryMeta) => emit('select', e))
provide(NewKey, (folder: string) => emit('newEntry', folder))
provide(SelectedIdKey, toRef(props, 'selectedId'))
</script>

<template>
  <div class="text-sm">
    <FolderTreeNode
      v-for="f in tree.folders"
      :key="f.path"
      :node="f"
      :depth="0"
    />
    <button
      v-for="e in tree.entries"
      :key="e.id"
      class="w-full text-left px-3 py-1.5 truncate hover:bg-ink-100/50 dark:hover:bg-ink-800/50"
      :class="selectedId === e.id ? 'bg-ink-100 dark:bg-ink-800' : ''"
      style="padding-left: 12px"
      @click="emit('select', e)"
    ><span class="prompt-accent" :class="selectedId === e.id ? '' : 'opacity-0'">&gt;</span> {{ e.title }}</button>
  </div>
</template>
