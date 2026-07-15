# Sidebar Usability + Bulk Move Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Put `[ + new here ]` at the top of expanded folders, fix the Move dialog's transparent background, and add select-mode bulk move for entries.

**Architecture:** All changes are in `web/` (Vue 3 + Tailwind v4 + vitest); no API changes. Bulk move loops the existing `api.move()` per selected id, with the list-update logic extracted into a pure, unit-tested helper. Select-mode state lives in `Entries.vue` and flows to tree nodes via provide/inject, matching the existing `folderTreeKeys.ts` pattern.

**Tech Stack:** Vue 3 `<script setup>` + TypeScript, Tailwind CSS v4, vitest.

**Spec:** `docs/superpowers/specs/2026-07-14-sidebar-usability-bulk-move-design.md`

## Global Constraints

- All paths below are relative to the repo root; run npm commands from `web/`.
- No backend/API changes; no new npm dependencies.
- Component-level Vue changes have no unit-test harness in this repo (no @vue/test-utils) — for those tasks the test cycle is `npm run typecheck` plus the end-to-end verification in Task 6. Pure logic (Task 3) is TDD'd with vitest.
- Match the terminal aesthetic: bracketed `[ ... ]` button labels, existing `btn-term`/`pane`/`prompt-accent` classes, `ink-*` colors.

---

### Task 1: `[ + new here ]` at the top of expanded folders

**Files:**
- Modify: `web/src/components/FolderTreeNode.vue:38-58`

**Interfaces:**
- Consumes: nothing new.
- Produces: no API change — template reorder only.

- [ ] **Step 1: Move the button to the top of the expanded block**

In `FolderTreeNode.vue`, inside `<template v-if="open">`, move the `[ + new here ]` button from last position to first, so the block reads:

```vue
    <template v-if="open">
      <button
        class="w-full text-left px-3 py-1 text-xs opacity-50 hover:opacity-100"
        :style="{ paddingLeft: pad(depth + 1) }"
        @click="newEntry(node.path)"
      >[ + new here ]</button>
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
      ><span class="prompt-accent" :class="selectedId === e.id ? '' : 'opacity-0'">&gt;</span> {{ e.title }}</button>
    </template>
```

(The button markup is unchanged — only its position moves.)

- [ ] **Step 2: Typecheck**

Run: `cd web && npm run typecheck`
Expected: exits 0, no errors.

- [ ] **Step 3: Commit**

```bash
git add web/src/components/FolderTreeNode.vue
git commit -m "fix(web): put [ + new here ] at the top of expanded folders"
```

---

### Task 2: Solid Move-dialog background via `@layer components`

**Files:**
- Modify: `web/src/style.css:61-96`

**Interfaces:**
- Consumes: nothing.
- Produces: `.pane`, `.btn-term`, `.input-term` now live in `@layer components`, so Tailwind utilities (e.g. `bg-ink-50`) override them. Task 4's action bar relies on utility backgrounds working.

- [ ] **Step 1: Wrap the custom component classes in `@layer components`**

In `web/src/style.css`, wrap the `.pane`, `.btn-term` (including `:hover`/`:disabled`), and `.input-term` (including `:focus`) rules — currently unlayered — in a single `@layer components { ... }` block. The rule bodies are unchanged; only the layer wrapper is added:

```css
@layer components {
  /* A bordered terminal panel. Layered so Tailwind utilities (e.g. a modal's
     bg-ink-50) can override these defaults — unlayered CSS always beats
     layered utilities, which is what made the Move dialog transparent. */
  .pane {
    border: 1px solid var(--term-border);
    border-radius: 0.375rem;
    background: transparent;
  }

  /* Bracketed/boxed monospace button. */
  .btn-term {
    font: inherit;
    border: 1px solid var(--term-border);
    border-radius: 0.25rem;
    padding: 0.3rem 0.7rem;
    background: transparent;
    transition: color 0.15s, border-color 0.15s, background-color 0.15s;
  }
  .btn-term:hover:not(:disabled) {
    color: var(--term-accent);
    border-color: var(--term-accent);
  }
  .btn-term:disabled { opacity: 0.5; }

  /* Transparent bordered text controls with an accent caret + focus ring. */
  .input-term {
    font: inherit;
    background: transparent;
    border: 1px solid var(--term-border);
    border-radius: 0.25rem;
    padding: 0.4rem 0.6rem;
    caret-color: var(--term-accent);
  }
  .input-term:focus {
    outline: none;
    border-color: var(--term-accent);
    box-shadow: 0 0 0 2px color-mix(in srgb, var(--term-accent) 35%, transparent);
  }
}
```

Leave the theme tokens, `html, body`, `.prose-ink`, and `.prompt-accent` rules unlayered and untouched.

- [ ] **Step 2: Build to confirm CSS compiles**

Run: `cd web && npm run build`
Expected: exits 0 (vue-tsc + vite build both pass).

- [ ] **Step 3: Commit**

```bash
git add web/src/style.css
git commit -m "fix(web): layer component classes so utilities win; fixes transparent Move dialog"
```

---

### Task 3: `applyMoveResults` helper (TDD)

**Files:**
- Create: `web/src/lib/bulkMove.ts`
- Test: `web/src/lib/bulkMove.test.ts`

**Interfaces:**
- Consumes: `EntryMeta` from `web/src/lib/api.ts`.
- Produces (Task 5 relies on these exact names/types):

```ts
export interface MoveOutcome {
  id: string
  // updated_at from the PATCH response on success, null on failure
  resp: Pick<EntryMeta, 'updated_at'> | null
}

export function applyMoveResults(
  list: EntryMeta[],
  folder: string,
  outcomes: MoveOutcome[],
): { list: EntryMeta[]; moved: string[]; failed: string[] }
```

- [ ] **Step 1: Write the failing tests**

Create `web/src/lib/bulkMove.test.ts`:

```ts
import { describe, expect, it } from 'vitest'
import type { EntryMeta } from './api'
import { applyMoveResults } from './bulkMove'

function entry(id: string, folder = 'old'): EntryMeta {
  return { id, title: `t-${id}`, folder, created_at: '2026-01-01T00:00:00Z', updated_at: '2026-01-01T00:00:00Z' }
}

describe('applyMoveResults', () => {
  it('applies folder and updated_at to moved entries only', () => {
    const list = [entry('a'), entry('b'), entry('c')]
    const { list: next, moved, failed } = applyMoveResults(list, 'new/place', [
      { id: 'a', resp: { updated_at: '2026-07-14T10:00:00Z' } },
      { id: 'c', resp: { updated_at: '2026-07-14T10:00:01Z' } },
    ])
    expect(moved).toEqual(['a', 'c'])
    expect(failed).toEqual([])
    expect(next.find((e) => e.id === 'a')).toMatchObject({ folder: 'new/place', updated_at: '2026-07-14T10:00:00Z' })
    expect(next.find((e) => e.id === 'b')).toMatchObject({ folder: 'old', updated_at: '2026-01-01T00:00:00Z' })
    expect(next.find((e) => e.id === 'c')).toMatchObject({ folder: 'new/place', updated_at: '2026-07-14T10:00:01Z' })
  })

  it('splits failures out and leaves their entries untouched', () => {
    const list = [entry('a'), entry('b')]
    const { list: next, moved, failed } = applyMoveResults(list, 'dest', [
      { id: 'a', resp: null },
      { id: 'b', resp: { updated_at: '2026-07-14T10:00:00Z' } },
    ])
    expect(moved).toEqual(['b'])
    expect(failed).toEqual(['a'])
    expect(next.find((e) => e.id === 'a')).toMatchObject({ folder: 'old' })
  })

  it('does not mutate the input list', () => {
    const list = [entry('a')]
    applyMoveResults(list, 'dest', [{ id: 'a', resp: { updated_at: '2026-07-14T10:00:00Z' } }])
    expect(list[0].folder).toBe('old')
  })

  it('handles empty outcomes', () => {
    const list = [entry('a')]
    const { list: next, moved, failed } = applyMoveResults(list, 'dest', [])
    expect(next).toEqual(list)
    expect(moved).toEqual([])
    expect(failed).toEqual([])
  })
})
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd web && npx vitest run src/lib/bulkMove.test.ts`
Expected: FAIL — cannot resolve `./bulkMove`.

- [ ] **Step 3: Implement the helper**

Create `web/src/lib/bulkMove.ts`:

```ts
import type { EntryMeta } from './api'

export interface MoveOutcome {
  id: string
  // updated_at from the PATCH response on success, null on failure. The
  // PATCH-move response is only trusted for updated_at (see Entries.vue).
  resp: Pick<EntryMeta, 'updated_at'> | null
}

// Applies the results of a bulk move to an entry list without mutating it.
export function applyMoveResults(
  list: EntryMeta[],
  folder: string,
  outcomes: MoveOutcome[],
): { list: EntryMeta[]; moved: string[]; failed: string[] } {
  const moved = outcomes.filter((o) => o.resp !== null)
  const failed = outcomes.filter((o) => o.resp === null).map((o) => o.id)
  const byId = new Map(moved.map((o) => [o.id, o.resp!.updated_at]))
  const next = list.map((e) =>
    byId.has(e.id) ? { ...e, folder, updated_at: byId.get(e.id)! } : e,
  )
  return { list: next, moved: moved.map((o) => o.id), failed }
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd web && npx vitest run src/lib/bulkMove.test.ts`
Expected: 4 passed.

- [ ] **Step 5: Run the full web test suite**

Run: `cd web && npm test`
Expected: all tests pass (existing api/markdown/tree tests plus the new file).

- [ ] **Step 6: Commit**

```bash
git add web/src/lib/bulkMove.ts web/src/lib/bulkMove.test.ts
git commit -m "feat(web): pure helper to apply bulk-move results to the entry list"
```

---

### Task 4: Select mode — state, tree checkboxes, action bar

**Files:**
- Modify: `web/src/components/folderTreeKeys.ts`
- Modify: `web/src/components/FolderTree.vue`
- Modify: `web/src/components/FolderTreeNode.vue`
- Modify: `web/src/views/Entries.vue`

**Interfaces:**
- Consumes: existing provide/inject pattern (`NewKey`, `SelectKey`, `SelectedIdKey`).
- Produces (Task 5 relies on these):
  - `Entries.vue` refs: `selectMode: Ref<boolean>`, `selectedIds: Ref<Set<string>>`, functions `toggleSelectMode(): void`, `toggleSelected(id: string): void`.
  - Injection keys: `SelectModeKey: InjectionKey<Ref<boolean>>`, `SelectedIdsKey: InjectionKey<Ref<Set<string>>>`, `ToggleSelectedKey: InjectionKey<(id: string) => void>`.
  - `FolderTree.vue` props gain `selectMode: boolean; selectedIds: Set<string>` and emit `toggleSelected: [id: string]`.

- [ ] **Step 1: Add the injection keys**

Append to `web/src/components/folderTreeKeys.ts`:

```ts
export const SelectModeKey: InjectionKey<Ref<boolean>> = Symbol('tree-select-mode')
export const SelectedIdsKey: InjectionKey<Ref<Set<string>>> = Symbol('tree-selected-ids')
export const ToggleSelectedKey: InjectionKey<(id: string) => void> = Symbol('tree-toggle-selected')
```

- [ ] **Step 2: Provide them from `FolderTree.vue` and handle root-level entries**

Replace the script block of `web/src/components/FolderTree.vue`:

```vue
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
```

And update the root-level entry buttons in its template (folders are unchanged):

```vue
    <button
      v-for="e in tree.entries"
      :key="e.id"
      class="w-full text-left px-3 py-1.5 truncate hover:bg-ink-100/50 dark:hover:bg-ink-800/50"
      :class="selectedId === e.id ? 'bg-ink-100 dark:bg-ink-800' : ''"
      style="padding-left: 12px"
      @click="selectMode ? emit('toggleSelected', e.id) : emit('select', e)"
    ><template v-if="selectMode">{{ selectedIds.has(e.id) ? '[x]' : '[ ]' }}</template><span v-else class="prompt-accent" :class="selectedId === e.id ? '' : 'opacity-0'">&gt;</span> {{ e.title }}</button>
```

- [ ] **Step 3: Inject in `FolderTreeNode.vue` and render checkboxes**

Update the script imports/injections in `web/src/components/FolderTreeNode.vue`:

```ts
import { inject, ref } from 'vue'
import type { TreeNode } from '../lib/tree'
import {
  NewKey, SelectKey, SelectedIdKey,
  SelectModeKey, SelectedIdsKey, ToggleSelectedKey,
} from './folderTreeKeys'

const props = defineProps<{ node: TreeNode; depth: number }>()

const open = ref(false)
const select = inject(SelectKey)!
const newEntry = inject(NewKey)!
const selectedId = inject(SelectedIdKey)!
const selectMode = inject(SelectModeKey)!
const selectedIds = inject(SelectedIdsKey)!
const toggleSelected = inject(ToggleSelectedKey)!
```

And update its entry buttons (inside `<template v-if="open">`; the `[ + new here ]` button from Task 1 stays first and unchanged):

```vue
      <button
        v-for="e in node.entries"
        :key="e.id"
        class="w-full text-left px-3 py-1.5 truncate hover:bg-ink-100/50 dark:hover:bg-ink-800/50"
        :class="selectedId === e.id ? 'bg-ink-100 dark:bg-ink-800' : ''"
        :style="{ paddingLeft: pad(depth + 1) }"
        @click="selectMode ? toggleSelected(e.id) : select(e)"
      ><template v-if="selectMode">{{ selectedIds.has(e.id) ? '[x]' : '[ ]' }}</template><span v-else class="prompt-accent" :class="selectedId === e.id ? '' : 'opacity-0'">&gt;</span> {{ e.title }}</button>
```

- [ ] **Step 4: Add state, header button, and action bar to `Entries.vue`**

In `web/src/views/Entries.vue` script, after the existing refs (`showMove` etc.), add:

```ts
const selectMode = ref(false)
const selectedIds = ref<Set<string>>(new Set())

function toggleSelectMode() {
  selectMode.value = !selectMode.value
  selectedIds.value = new Set()
}

function toggleSelected(id: string) {
  const next = new Set(selectedIds.value)
  if (next.has(id)) next.delete(id)
  else next.add(id)
  selectedIds.value = next
}
```

In the template, replace the sidebar header div and `FolderTree` usage, and add the action bar as the last child of `<aside>`:

```vue
    <aside class="border-r border-ink-100 dark:border-ink-800 overflow-y-auto min-h-0 flex flex-col">
      <div class="p-3 border-b border-ink-100 dark:border-ink-800 flex items-center justify-between">
        <span class="text-sm opacity-70">{{ list.length }} entries</span>
        <div class="flex gap-2">
          <button
            v-if="list.length && !selectMode"
            class="btn-term text-sm"
            @click="toggleSelectMode"
          >[ select ]</button>
          <button class="btn-term text-sm" @click="startCompose('')">[ + new ]</button>
        </div>
      </div>

      <p v-if="loading" class="p-4 text-sm opacity-60">loading…</p>
      <p v-else-if="!list.length" class="p-4 text-sm opacity-60">no entries yet</p>

      <div v-else class="flex-1 overflow-y-auto min-h-0">
        <FolderTree
          :tree="tree"
          :selected-id="selected?.id ?? null"
          :select-mode="selectMode"
          :selected-ids="selectedIds"
          @select="open"
          @new-entry="startCompose"
          @toggle-selected="toggleSelected"
        />
      </div>

      <div
        v-if="selectMode"
        class="border-t border-ink-100 dark:border-ink-800 bg-ink-50 dark:bg-ink-900 p-3 flex items-center justify-between gap-2"
      >
        <span class="text-sm opacity-70">{{ selectedIds.size }} selected</span>
        <div class="flex gap-2">
          <button
            class="btn-term text-sm"
            :disabled="!selectedIds.size"
            @click="showBulkMove = true"
          >[ move {{ selectedIds.size }} ]</button>
          <button class="btn-term text-sm" @click="toggleSelectMode">[ cancel ]</button>
        </div>
      </div>
    </aside>
```

Note the `<aside>` becomes a flex column with the tree in its own scroll container, so the action bar stays pinned at the bottom. `showBulkMove` is referenced here but defined in Task 5 — add `const showBulkMove = ref(false)` alongside the other new refs now so the template compiles.

- [ ] **Step 5: Typecheck**

Run: `cd web && npm run typecheck`
Expected: exits 0.

- [ ] **Step 6: Commit**

```bash
git add web/src/components/folderTreeKeys.ts web/src/components/FolderTree.vue web/src/components/FolderTreeNode.vue web/src/views/Entries.vue
git commit -m "feat(web): select mode with checkboxes and action bar in the entry tree"
```

---

### Task 5: Bulk move execution

**Files:**
- Modify: `web/src/views/Entries.vue`

**Interfaces:**
- Consumes: `applyMoveResults` / `MoveOutcome` from `web/src/lib/bulkMove.ts` (Task 3); `selectMode`, `selectedIds`, `toggleSelectMode`, `showBulkMove` from Task 4; existing `api.move`, `MoveDialog`.
- Produces: complete bulk-move flow.

- [ ] **Step 1: Add `doBulkMove` and the bulk `MoveDialog`**

In `web/src/views/Entries.vue` script, add the import and the function:

```ts
import { applyMoveResults, type MoveOutcome } from '../lib/bulkMove'
```

```ts
async function doBulkMove(folder: string) {
  const ids = [...selectedIds.value]
  showBulkMove.value = false
  error.value = ''
  const settled = await Promise.allSettled(ids.map((id) => api.move(id, folder)))
  const outcomes: MoveOutcome[] = ids.map((id, i) => {
    const s = settled[i]
    return { id, resp: s.status === 'fulfilled' ? { updated_at: s.value.updated_at } : null }
  })
  const { list: next, moved, failed } = applyMoveResults(list.value, folder, outcomes)
  list.value = next
  if (selected.value && moved.includes(selected.value.id)) {
    const updated = next.find((e) => e.id === selected.value!.id)!
    selected.value = { ...selected.value, folder, updated_at: updated.updated_at }
  }
  if (failed.length) {
    // Failed entries stay selected so [ move N ] retries just those.
    error.value = `moved ${moved.length} of ${ids.length}`
    selectedIds.value = new Set(failed)
  } else {
    selectMode.value = false
    selectedIds.value = new Set()
  }
}
```

In the template, after the existing single-entry `<MoveDialog>`, add:

```vue
    <MoveDialog
      v-if="showBulkMove"
      :folders="folders"
      current=""
      @move="doBulkMove"
      @cancel="showBulkMove = false"
    />
```

- [ ] **Step 2: Typecheck and run tests**

Run: `cd web && npm run typecheck && npm test`
Expected: both exit 0.

- [ ] **Step 3: Commit**

```bash
git add web/src/views/Entries.vue
git commit -m "feat(web): bulk move selected entries via the move dialog"
```

---

### Task 6: End-to-end verification

**Files:** none (verification only).

- [ ] **Step 1: Run the app**

Run the dev server (`cd web && npm run dev`) and drive the UI (per the repo's run/verify tooling). Verify:

1. Expanding a folder shows `[ + new here ]` as the first row, and clicking it opens compose targeted at that folder.
2. `[ move ]` on a single entry opens a dialog with a **solid** background in both light and dark mode (toggle dark mode to check both).
3. `[ select ]` → entries show `[ ]`, clicking toggles `[x]` without opening the entry, folders still expand/collapse, the action bar shows the right count.
4. `[ move N ]` opens the dialog; choosing a folder moves all selected entries (confirm they appear under the destination folder), exits select mode, and clears the selection.
5. `[ cancel ]` exits select mode and restores normal entry opening.

- [ ] **Step 2: Full build**

Run: `cd web && npm run build && npm test`
Expected: both exit 0.

- [ ] **Step 3: Commit any fixes found**

If verification surfaced fixes, commit them with a descriptive message.
