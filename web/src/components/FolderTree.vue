<script setup lang="ts">
import { provide, toRef } from 'vue'
import type { TreeNode } from '../lib/tree'
import type { EntryMeta } from '../lib/api'
import FolderTreeNode from './FolderTreeNode.vue'
import {
  NewKey, SelectKey, SelectedIdKey,
  SelectModeKey, SelectedIdsKey, ToggleSelectedKey,
} from './folderTreeKeys'

const props = defineProps<{
  tree: TreeNode
  selectedId: string | null
  selectMode: boolean
  selectedIds: Set<string>
}>()
const emit = defineEmits<{
  select: [entry: EntryMeta]
  newEntry: [folder: string]
  toggleSelected: [id: string]
}>()

provide(SelectKey, (e: EntryMeta) => emit('select', e))
provide(NewKey, (folder: string) => emit('newEntry', folder))
provide(SelectedIdKey, toRef(props, 'selectedId'))
provide(SelectModeKey, toRef(props, 'selectMode'))
provide(SelectedIdsKey, toRef(props, 'selectedIds'))
provide(ToggleSelectedKey, (id: string) => emit('toggleSelected', id))
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
      @click="selectMode ? emit('toggleSelected', e.id) : emit('select', e)"
    ><template v-if="selectMode">{{ selectedIds.has(e.id) ? '[x]' : '[ ]' }}</template><span v-else class="prompt-accent" :class="selectedId === e.id ? '' : 'opacity-0'">&gt;</span> {{ e.title }}</button>
  </div>
</template>
