import type { InjectionKey, Ref } from 'vue'
import type { EntryMeta } from '../lib/api'

// Provided by FolderTree, injected by FolderTreeNode at any depth — avoids
// bubbling emits up a recursive component tree.
export const SelectKey: InjectionKey<(e: EntryMeta) => void> = Symbol('tree-select')
export const NewKey: InjectionKey<(folder: string) => void> = Symbol('tree-new')
export const SelectedIdKey: InjectionKey<Ref<string | null>> = Symbol('tree-selected-id')
