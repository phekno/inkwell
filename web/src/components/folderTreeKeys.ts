import type { InjectionKey, Ref } from 'vue'
import type { EntryMeta } from '../lib/api'

// Provided by FolderTree, injected by FolderTreeNode at any depth — avoids
// bubbling emits up a recursive component tree.
export const SelectKey: InjectionKey<(e: EntryMeta) => void> = Symbol('tree-select')
export const NewKey: InjectionKey<(folder: string) => void> = Symbol('tree-new')
export const SelectedIdKey: InjectionKey<Ref<string | null>> = Symbol('tree-selected-id')

export const SelectModeKey: InjectionKey<Ref<boolean>> = Symbol('tree-select-mode')
export const SelectedIdsKey: InjectionKey<Ref<Set<string>>> = Symbol('tree-selected-ids')
export const ToggleSelectedKey: InjectionKey<(id: string) => void> = Symbol('tree-toggle-selected')

export const NewFolderKey: InjectionKey<(parent: string) => void> = Symbol('tree-new-folder')
// Folder path to expand to (every ancestor opens), e.g. a just-created folder.
export const RevealKey: InjectionKey<Ref<string>> = Symbol('tree-reveal')
