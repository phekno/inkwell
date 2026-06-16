# Web UI → TUI Parity Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Bring the Vue web app to feature parity with the TUI — folder-tree sidebar, editing (live split preview), moving entries (folder picker), and markdown rendering.

**Architecture:** Decompose the monolithic `Entries.vue` into focused units: pure libs (`tree.ts`, `markdown.ts`) that are unit-tested, an extended `api.ts`, three new components (recursive folder tree, editor, move dialog), and a thin `Entries.vue` orchestrator. No API or infrastructure changes — the backend already exposes `POST`/`GET`/`PATCH`/`DELETE /entries`.

**Tech Stack:** Vue 3 (`<script setup>` + TS), Vite, Tailwind v4, Pinia, vue-router, vitest + jsdom. New deps: `marked` (markdown parse), `dompurify` (HTML sanitize).

## Global Constraints

- All code is TypeScript with `strict: true`. No `any` except the existing `catch (e: any)` error pattern already used in `Entries.vue`.
- Tests use `vitest` with explicit imports from `'vitest'` (no reliance on globals), following the pattern in `web/src/lib/api.test.ts`.
- There is **no** `@vue/test-utils` in the project; do not add it. Pure logic (`tree.ts`, `markdown.ts`, `api.ts`) is covered by unit tests. `.vue` components are verified by `npm run typecheck` (vue-tsc type-checks every file under `src/**/*.vue`) and `npm run build`, plus manual verification at the end.
- Tailwind design tokens in use: `ink-50`, `ink-100`, `ink-800`, `ink-900`; dark mode via the `dark:` variant (class-based). Match the existing visual style in `web/src/views/Entries.vue` and `web/src/App.vue`.
- Run all `npm` commands from the `web/` directory.
- The backend `PATCH /entries/{id}` response (`entryView`) serializes **all** fields without `omitempty`: an edit (title/body) returns `folder: ""`, and a move returns `title: ""`. **Never** copy those zero-value fields into local state. Trust only `updated_at` from PATCH responses and apply the values the user supplied locally.

---

### Task 1: Markdown rendering (`lib/markdown.ts`)

**Files:**
- Modify: `web/package.json` (add `marked`, `dompurify`)
- Create: `web/src/lib/markdown.ts`
- Test: `web/src/lib/markdown.test.ts`

**Interfaces:**
- Consumes: nothing.
- Produces: `renderMarkdown(src: string): string` — parses markdown to HTML and sanitizes it. Safe to bind via `v-html`.

- [ ] **Step 1: Install dependencies**

Run (from `web/`):
```bash
npm install marked dompurify
```
Expected: `package.json` `dependencies` now include `marked` and `dompurify`; install completes with no errors. (Both ship their own TypeScript types — do **not** add `@types/*` packages.)

- [ ] **Step 2: Write the failing test**

Create `web/src/lib/markdown.test.ts`:
```ts
import { describe, expect, it } from 'vitest'
import { renderMarkdown } from './markdown'

describe('renderMarkdown', () => {
  it('renders basic markdown to HTML', () => {
    const html = renderMarkdown('# Hello\n\nsome **bold** text')
    expect(html).toContain('<h1')
    expect(html).toContain('Hello')
    expect(html).toContain('<strong>bold</strong>')
  })

  it('strips dangerous markup', () => {
    const html = renderMarkdown('ok <img src=x onerror="alert(1)"> <script>alert(2)<\/script>')
    expect(html).not.toContain('onerror')
    expect(html).not.toContain('<script')
  })

  it('handles empty input', () => {
    expect(renderMarkdown('')).toBe('')
  })
})
```

- [ ] **Step 3: Run test to verify it fails**

Run: `npm test -- markdown`
Expected: FAIL — `Cannot find module './markdown'` (or "renderMarkdown is not a function").

- [ ] **Step 4: Write the implementation**

Create `web/src/lib/markdown.ts`:
```ts
import { marked } from 'marked'
import DOMPurify from 'dompurify'

// Parse markdown to HTML and sanitize before it is bound via v-html. Content is
// the user's own private journal, but sanitizing is cheap insurance and keeps
// any pasted HTML from doing something surprising.
export function renderMarkdown(src: string): string {
  if (!src) return ''
  const raw = marked.parse(src, { async: false }) as string
  return DOMPurify.sanitize(raw)
}
```

- [ ] **Step 5: Run test to verify it passes**

Run: `npm test -- markdown`
Expected: PASS (3 tests).

- [ ] **Step 6: Commit**

```bash
git add web/package.json web/package-lock.json web/src/lib/markdown.ts web/src/lib/markdown.test.ts
git commit -m "feat(web): markdown rendering via marked + DOMPurify"
```

---

### Task 2: Folder tree builder (`lib/tree.ts`)

**Files:**
- Create: `web/src/lib/tree.ts`
- Test: `web/src/lib/tree.test.ts`

**Interfaces:**
- Consumes: `EntryMeta` from `./api` (currently `{ id, title, created_at }`; gains `folder` and `updated_at` in Task 3 — the tree only reads `folder` and `created_at`, both of which exist by the time this is used end-to-end).
- Produces:
  - `interface TreeNode { name: string; path: string; folders: TreeNode[]; entries: EntryMeta[] }`
  - `buildTree(metas: EntryMeta[]): TreeNode` — root node has `name: ''`, `path: ''`. Folders sorted by `name` (ascending, `localeCompare`); entries sorted by `created_at` descending (newest first).
  - `folderPaths(node: TreeNode): string[]` — every folder path in the tree, depth-first in tree order (root itself excluded).

- [ ] **Step 1: Write the failing test**

Create `web/src/lib/tree.test.ts`:
```ts
import { describe, expect, it } from 'vitest'
import { buildTree, folderPaths } from './tree'
import type { EntryMeta } from './api'

function meta(id: string, folder: string, created_at: string): EntryMeta {
  return { id, title: id, folder, created_at, updated_at: created_at }
}

describe('buildTree', () => {
  it('returns an empty root for no entries', () => {
    const root = buildTree([])
    expect(root).toEqual({ name: '', path: '', folders: [], entries: [] })
  })

  it('places root-folder entries at the top level', () => {
    const root = buildTree([meta('a', '', '2026-01-01'), meta('b', '', '2026-01-02')])
    expect(root.folders).toHaveLength(0)
    expect(root.entries.map((e) => e.id)).toEqual(['b', 'a']) // newest first
  })

  it('nests entries under their folder path', () => {
    const root = buildTree([meta('x', 'Personal/Health', '2026-01-01')])
    expect(root.folders.map((f) => f.name)).toEqual(['Personal'])
    const personal = root.folders[0]
    expect(personal.path).toBe('Personal')
    expect(personal.entries).toHaveLength(0)
    expect(personal.folders.map((f) => f.name)).toEqual(['Health'])
    const health = personal.folders[0]
    expect(health.path).toBe('Personal/Health')
    expect(health.entries.map((e) => e.id)).toEqual(['x'])
  })

  it('sorts folders alphabetically, before entries', () => {
    const root = buildTree([
      meta('e1', '', '2026-01-01'),
      meta('e2', 'Work', '2026-01-01'),
      meta('e3', 'Journal', '2026-01-01'),
    ])
    expect(root.folders.map((f) => f.name)).toEqual(['Journal', 'Work'])
    expect(root.entries.map((e) => e.id)).toEqual(['e1'])
  })
})

describe('folderPaths', () => {
  it('flattens all folder paths depth-first', () => {
    const root = buildTree([
      meta('a', 'Personal/Health', '2026-01-01'),
      meta('b', 'Work', '2026-01-01'),
    ])
    expect(folderPaths(root)).toEqual(['Personal', 'Personal/Health', 'Work'])
  })
})
```

- [ ] **Step 2: Run test to verify it fails**

Run: `npm test -- tree`
Expected: FAIL — `Cannot find module './tree'`.

- [ ] **Step 3: Write the implementation**

Create `web/src/lib/tree.ts`:
```ts
import type { EntryMeta } from './api'

export interface TreeNode {
  name: string // folder segment name ('' for root)
  path: string // full slash path ('' for root, 'Personal/Health' nested)
  folders: TreeNode[]
  entries: EntryMeta[]
}

function newNode(name: string, path: string): TreeNode {
  return { name, path, folders: [], entries: [] }
}

// buildTree turns the flat entry list into a nested folder structure. Each
// entry's `folder` is a slash path; empty means the root. Folders sort
// alphabetically; entries sort newest-first by created_at.
export function buildTree(metas: EntryMeta[]): TreeNode {
  const root = newNode('', '')

  for (const m of metas) {
    let node = root
    const segments = m.folder.split('/').filter((s) => s.length > 0)
    for (const seg of segments) {
      const childPath = node.path ? `${node.path}/${seg}` : seg
      let child = node.folders.find((f) => f.name === seg)
      if (!child) {
        child = newNode(seg, childPath)
        node.folders.push(child)
      }
      node = child
    }
    node.entries.push(m)
  }

  sortNode(root)
  return root
}

function sortNode(node: TreeNode): void {
  node.folders.sort((a, b) => a.name.localeCompare(b.name))
  node.entries.sort((a, b) => b.created_at.localeCompare(a.created_at))
  for (const f of node.folders) sortNode(f)
}

// folderPaths lists every folder path in the tree, depth-first (root excluded).
export function folderPaths(node: TreeNode): string[] {
  const out: string[] = []
  for (const f of node.folders) {
    out.push(f.path)
    out.push(...folderPaths(f))
  }
  return out
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `npm test -- tree`
Expected: PASS (5 tests).

- [ ] **Step 5: Commit**

```bash
git add web/src/lib/tree.ts web/src/lib/tree.test.ts
git commit -m "feat(web): pure folder-tree builder from entry paths"
```

---

### Task 3: API client — folders, edit, move (`lib/api.ts`)

**Files:**
- Modify: `web/src/lib/api.ts`
- Test: `web/src/lib/api.test.ts`

**Interfaces:**
- Consumes: the existing `call<T>(method, path, body?)` helper (unchanged).
- Produces (the `api` object gains/changes):
  - `EntryMeta` = `{ id: string; title: string; folder: string; created_at: string; updated_at: string }`
  - `Entry extends EntryMeta` = adds `body: string`
  - `create(title: string, body: string, folder?: string): Promise<EntryMeta>` — `folder` defaults to `''`
  - `update(id: string, fields: { title: string; body: string }): Promise<EntryMeta>` — `PATCH`
  - `move(id: string, folder: string): Promise<EntryMeta>` — `PATCH` with only `{ folder }`

- [ ] **Step 1: Write the failing tests**

Edit `web/src/lib/api.test.ts`. Replace the existing `'encodes JSON on create'` test and add two new tests inside the `describe('api client', …)` block:

```ts
  it('encodes title, body, and folder on create', async () => {
    const f = mockFetch({ status: 201, body: { id: 'x', title: 't', folder: 'Work', created_at: '2026-01-01', updated_at: '2026-01-01' } })
    await api.create('hello', 'body', 'Work')
    const init = f.mock.calls[0][1] as RequestInit
    expect(init.body).toBe(JSON.stringify({ title: 'hello', body: 'body', folder: 'Work' }))
    expect(init.headers).toMatchObject({ 'content-type': 'application/json' })
  })

  it('PATCHes title and body on update', async () => {
    const f = mockFetch({ status: 200, body: { id: 'x', updated_at: '2026-02-02' } })
    await api.update('x', { title: 'new', body: 'changed' })
    const [url, init] = f.mock.calls[0] as [string, RequestInit]
    expect(url).toMatch(/\/entries\/x$/)
    expect(init.method).toBe('PATCH')
    expect(init.body).toBe(JSON.stringify({ title: 'new', body: 'changed' }))
  })

  it('PATCHes only folder on move', async () => {
    const f = mockFetch({ status: 200, body: { id: 'x', updated_at: '2026-02-02' } })
    await api.move('x', 'Personal/Health')
    const init = f.mock.calls[0][1] as RequestInit
    expect(init.method).toBe('PATCH')
    expect(init.body).toBe(JSON.stringify({ folder: 'Personal/Health' }))
  })
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `npm test -- api`
Expected: FAIL — `api.update is not a function` / `api.move is not a function`, and the create test fails because `folder` is not yet sent.

- [ ] **Step 3: Update the implementation**

Edit `web/src/lib/api.ts`. Replace the `EntryMeta`/`Entry` interfaces and the `api` object:

```ts
export interface EntryMeta {
  id: string
  title: string
  folder: string
  created_at: string
  updated_at: string
}

export interface Entry extends EntryMeta {
  body: string
}

export const api = {
  list: () => call<EntryMeta[]>('GET', '/entries'),
  get: (id: string) => call<Entry>('GET', `/entries/${id}`),
  create: (title: string, body: string, folder = '') =>
    call<EntryMeta>('POST', '/entries', { title, body, folder }),
  update: (id: string, fields: { title: string; body: string }) =>
    call<EntryMeta>('PATCH', `/entries/${id}`, fields),
  move: (id: string, folder: string) =>
    call<EntryMeta>('PATCH', `/entries/${id}`, { folder }),
  delete: (id: string) => call<void>('DELETE', `/entries/${id}`),
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `npm test -- api`
Expected: PASS (all api-client tests, including the two new PATCH tests and the updated create test).

- [ ] **Step 5: Run the full test suite and typecheck**

Run: `npm test && npm run typecheck`
Expected: all tests PASS; `vue-tsc` reports no errors. (Existing `Entries.vue` still compiles — it does not yet read `folder`/`updated_at`, and the added fields are optional to its current usage.)

- [ ] **Step 6: Commit**

```bash
git add web/src/lib/api.ts web/src/lib/api.test.ts
git commit -m "feat(web): api client gains folder, update, and move"
```

---

### Task 4: Folder-tree components (`FolderTree.vue` + recursive `FolderTreeNode.vue`)

**Files:**
- Create: `web/src/components/folderTreeKeys.ts`
- Create: `web/src/components/FolderTreeNode.vue`
- Create: `web/src/components/FolderTree.vue`

**Interfaces:**
- Consumes: `TreeNode` from `../lib/tree`; `EntryMeta` from `../lib/api`.
- Produces:
  - `folderTreeKeys.ts`: injection keys `SelectKey: InjectionKey<(e: EntryMeta) => void>`, `NewKey: InjectionKey<(folder: string) => void>`, `SelectedIdKey: InjectionKey<Ref<string | null>>`.
  - `FolderTree.vue`: props `{ tree: TreeNode; selectedId: string | null }`; emits `select(entry: EntryMeta)`, `newEntry(folder: string)`. Provides the injection keys for descendants.
  - `FolderTreeNode.vue`: props `{ node: TreeNode; depth: number }`; renders recursively, injecting the keys. No emits (calls injected callbacks directly).

- [ ] **Step 1: Create the injection keys**

Create `web/src/components/folderTreeKeys.ts`:
```ts
import type { InjectionKey, Ref } from 'vue'
import type { EntryMeta } from '../lib/api'

// Provided by FolderTree, injected by FolderTreeNode at any depth — avoids
// bubbling emits up a recursive component tree.
export const SelectKey: InjectionKey<(e: EntryMeta) => void> = Symbol('tree-select')
export const NewKey: InjectionKey<(folder: string) => void> = Symbol('tree-new')
export const SelectedIdKey: InjectionKey<Ref<string | null>> = Symbol('tree-selected-id')
```

- [ ] **Step 2: Create the recursive node component**

Create `web/src/components/FolderTreeNode.vue`:
```vue
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
```

- [ ] **Step 3: Create the tree root component**

Create `web/src/components/FolderTree.vue`:
```vue
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
    >{{ e.title }}</button>
  </div>
</template>
```

- [ ] **Step 4: Typecheck**

Run: `npm run typecheck`
Expected: `vue-tsc` reports no errors. (The recursive self-reference in `FolderTreeNode.vue` resolves by filename; the injected callbacks are non-null asserted because `FolderTree` always provides them.)

- [ ] **Step 5: Commit**

```bash
git add web/src/components/folderTreeKeys.ts web/src/components/FolderTreeNode.vue web/src/components/FolderTree.vue
git commit -m "feat(web): expandable folder-tree sidebar components"
```

---

### Task 5: Entry editor with live preview (`EntryEditor.vue`)

**Files:**
- Create: `web/src/components/EntryEditor.vue`

**Interfaces:**
- Consumes: `renderMarkdown` from `../lib/markdown`.
- Produces: `EntryEditor.vue` with props `{ initialTitle?: string; initialBody?: string; saving?: boolean }`; emits `save(payload: { title: string; body: string })` and `cancel()`. Used for both compose (empty initials) and edit (prefilled).

- [ ] **Step 1: Create the component**

Create `web/src/components/EntryEditor.vue`:
```vue
<script setup lang="ts">
import { computed, ref } from 'vue'
import { renderMarkdown } from '../lib/markdown'

const props = defineProps<{
  initialTitle?: string
  initialBody?: string
  saving?: boolean
}>()
const emit = defineEmits<{
  save: [payload: { title: string; body: string }]
  cancel: []
}>()

const title = ref(props.initialTitle ?? '')
const body = ref(props.initialBody ?? '')
const preview = computed(() => renderMarkdown(body.value))

function save() {
  if (!title.value.trim() || props.saving) return
  emit('save', { title: title.value.trim(), body: body.value })
}
</script>

<template>
  <div class="flex flex-col h-full">
    <input
      v-model="title"
      placeholder="Title"
      class="w-full bg-transparent text-2xl font-medium border-b border-ink-100 dark:border-ink-800 focus:outline-none pb-2 mb-4"
    />
    <div class="grid grid-cols-2 gap-4 flex-1 min-h-0">
      <textarea
        v-model="body"
        placeholder="Write…"
        class="w-full h-full bg-transparent resize-none focus:outline-none leading-relaxed font-mono text-sm"
      ></textarea>
      <div
        class="overflow-y-auto border-l border-ink-100 dark:border-ink-800 pl-4 leading-relaxed [&_h1]:text-2xl [&_h1]:font-medium [&_h2]:text-xl [&_h2]:font-medium [&_ul]:list-disc [&_ul]:pl-5 [&_a]:underline"
        v-html="preview"
      ></div>
    </div>
    <div class="mt-4 flex gap-2">
      <button
        class="rounded-md px-4 py-2 text-sm bg-ink-800 dark:bg-ink-100 text-ink-50 dark:text-ink-900 hover:opacity-90 disabled:opacity-50"
        :disabled="saving"
        @click="save"
      >{{ saving ? 'saving…' : 'save' }}</button>
      <button
        class="rounded-md px-4 py-2 text-sm border border-ink-100 dark:border-ink-800 hover:bg-ink-100 dark:hover:bg-ink-800"
        @click="emit('cancel')"
      >cancel</button>
    </div>
  </div>
</template>
```

- [ ] **Step 2: Typecheck**

Run: `npm run typecheck`
Expected: `vue-tsc` reports no errors.

- [ ] **Step 3: Commit**

```bash
git add web/src/components/EntryEditor.vue
git commit -m "feat(web): entry editor with live markdown preview"
```

---

### Task 6: Move dialog (`MoveDialog.vue`)

**Files:**
- Create: `web/src/components/MoveDialog.vue`

**Interfaces:**
- Consumes: nothing (folder list passed in as a prop).
- Produces: `MoveDialog.vue` with props `{ folders: string[]; current: string }`; emits `move(folder: string)` and `cancel()`. A typed new-folder path takes precedence over the dropdown selection.

- [ ] **Step 1: Create the component**

Create `web/src/components/MoveDialog.vue`:
```vue
<script setup lang="ts">
import { ref } from 'vue'

const props = defineProps<{ folders: string[]; current: string }>()
const emit = defineEmits<{
  move: [folder: string]
  cancel: []
}>()

const choice = ref(props.current)
const custom = ref('')

function confirm() {
  emit('move', custom.value.trim() || choice.value)
}
</script>

<template>
  <div
    class="fixed inset-0 bg-black/40 flex items-center justify-center z-10"
    @click.self="emit('cancel')"
  >
    <div class="bg-ink-50 dark:bg-ink-900 border border-ink-100 dark:border-ink-800 rounded-lg p-5 w-80">
      <h3 class="text-sm font-medium mb-3">Move to folder</h3>
      <select
        v-model="choice"
        class="w-full bg-transparent border border-ink-100 dark:border-ink-800 rounded-md px-2 py-1.5 text-sm mb-3"
      >
        <option value="">(root)</option>
        <option v-for="f in folders" :key="f" :value="f">{{ f }}</option>
      </select>
      <input
        v-model="custom"
        placeholder="or type a new folder path"
        class="w-full bg-transparent border border-ink-100 dark:border-ink-800 rounded-md px-2 py-1.5 text-sm"
      />
      <div class="flex gap-2 mt-4">
        <button
          class="rounded-md px-4 py-2 text-sm bg-ink-800 dark:bg-ink-100 text-ink-50 dark:text-ink-900 hover:opacity-90"
          @click="confirm"
        >move</button>
        <button
          class="rounded-md px-4 py-2 text-sm border border-ink-100 dark:border-ink-800 hover:bg-ink-100 dark:hover:bg-ink-800"
          @click="emit('cancel')"
        >cancel</button>
      </div>
    </div>
  </div>
</template>
```

- [ ] **Step 2: Typecheck**

Run: `npm run typecheck`
Expected: `vue-tsc` reports no errors.

- [ ] **Step 3: Commit**

```bash
git add web/src/components/MoveDialog.vue
git commit -m "feat(web): folder-picker move dialog"
```

---

### Task 7: Orchestrate everything in `Entries.vue`

**Files:**
- Modify (full rewrite): `web/src/views/Entries.vue`

**Interfaces:**
- Consumes: `api` (`list`/`get`/`create`/`update`/`move`/`delete`), `buildTree`/`folderPaths`, `renderMarkdown`, and the three components.
- Produces: the wired-up entries view (no exports).

- [ ] **Step 1: Rewrite the view**

Replace the entire contents of `web/src/views/Entries.vue`:
```vue
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
  <section class="grid md:grid-cols-[20rem_1fr] h-[calc(100vh-65px)]">
    <aside class="border-r border-ink-100 dark:border-ink-800 overflow-y-auto">
      <div class="p-3 border-b border-ink-100 dark:border-ink-800 flex items-center justify-between">
        <span class="text-sm opacity-70">{{ list.length }} entries</span>
        <button
          class="rounded-md px-3 py-1.5 text-sm bg-ink-800 dark:bg-ink-100 text-ink-50 dark:text-ink-900 hover:opacity-90"
          @click="startCompose('')"
        >+ new</button>
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

    <article class="overflow-y-auto p-6">
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
          <h2 class="text-2xl font-medium">{{ selected.title }}</h2>
          <div class="shrink-0 flex gap-2">
            <button
              class="rounded-md px-3 py-1.5 text-sm border border-ink-100 dark:border-ink-800 hover:bg-ink-100 dark:hover:bg-ink-800"
              @click="startEdit"
            >edit</button>
            <button
              class="rounded-md px-3 py-1.5 text-sm border border-ink-100 dark:border-ink-800 hover:bg-ink-100 dark:hover:bg-ink-800"
              @click="showMove = true"
            >move</button>
            <button
              class="rounded-md px-3 py-1.5 text-sm border border-red-600/40 dark:border-red-400/40 text-red-700 dark:text-red-300 hover:bg-red-600 hover:text-white dark:hover:bg-red-500 dark:hover:text-white transition"
              @click="remove(selected.id)"
            >delete</button>
          </div>
        </div>
        <p class="text-xs opacity-60 mb-6">
          {{ fmt(selected.created_at) }}
          <span v-if="selected.folder" class="ml-2">· {{ selected.folder }}</span>
        </p>
        <div
          class="leading-relaxed [&_h1]:text-2xl [&_h1]:font-medium [&_h2]:text-xl [&_h2]:font-medium [&_ul]:list-disc [&_ul]:pl-5 [&_a]:underline"
          v-html="rendered"
        ></div>
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
```

- [ ] **Step 2: Typecheck and run the full test suite**

Run: `npm run typecheck && npm test`
Expected: `vue-tsc` reports no errors; all unit tests PASS (markdown, tree, api, plus any pre-existing).

- [ ] **Step 3: Build**

Run: `npm run build`
Expected: `vue-tsc --noEmit` clean and `vite build` completes with no errors.

- [ ] **Step 4: Manual verification**

Run: `npm run dev`, sign in, and confirm against the real API:
- The sidebar shows folders as an expandable tree; expanding a folder reveals subfolders and entries; root entries appear at the top level.
- Clicking an entry renders its body as **markdown** (headings, bold, lists) — not raw text.
- `+ new` (top) composes into root; `+ new here` inside a folder composes into that folder; the live preview updates as you type; save adds it to the tree under the right folder.
- `edit` on an open entry prefills the editor; saving updates the title/body and the entry stays in its folder (it does **not** jump to root).
- `move` opens the dialog; picking an existing folder or typing a new path relocates the entry in the tree; it does **not** lose its title.
- `delete` removes the entry and clears the detail pane.

- [ ] **Step 5: Commit**

```bash
git add web/src/views/Entries.vue
git commit -m "feat(web): folders, edit, move, and markdown in entries view"
```

---

## Notes for the implementer

- **Why local-value merging after PATCH (Tasks 3 & 7):** the backend `entryView` has no `omitempty`, so an edit returns `folder: ""` and a move returns `title: ""`. Copying the whole response into local state would wipe the folder on edit or the title on move. The orchestrator deliberately applies the values the user supplied and reads only `updated_at` back.
- **Recursive component:** `FolderTreeNode.vue` references itself by name; Vue resolves SFC self-references by filename, so no explicit registration is needed.
- **No component unit tests:** `@vue/test-utils` is not installed and is out of scope. Components are gated by `vue-tsc` typecheck + build + the manual pass in Task 7. The risk-bearing logic lives in the unit-tested pure libs.
```
