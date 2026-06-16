# Web UI → TUI parity — Design

Date: 2026-06-16
Status: Approved (pending implementation)

## Summary

Bring the Vue web app (`web/`) up to feature parity with the TUI by adding the
backend capabilities it already supports but the web lacks: **folder browsing**,
**editing**, **moving entries between folders**, and **markdown rendering**. No
API or infrastructure changes — the backend (`POST`/`GET`/`PATCH`/`DELETE
/entries`) already exposes everything needed.

Folder navigation uses an **expandable tree sidebar** (web-idiomatic; diverges
intentionally from the TUI's drill-in model). Editing uses a **live split
preview** (markdown textarea beside a rendered preview). Moving uses a **folder
picker** (choose an existing folder or type a new path).

## Current gap

`web/src/views/Entries.vue` is a single file supporting only list / open /
create / delete over a flat list. Missing vs the backend + TUI:

- No `folder` concept; `EntryMeta` lacks `folder` and `updated_at`.
- No editing (PATCH unused).
- No move.
- Bodies render as raw text in a `<pre>`, not markdown.

The Notion import also added hundreds of entries, so a flat list is unwieldy —
the tree is what makes the dataset navigable.

## API contract (already implemented, for reference)

- `POST /entries` `{title, body, folder}` → `entryView` `{id, title, folder,
  created_at, updated_at}`.
- `GET /entries` → `EntryMeta[]` (`{id, title, folder, created_at, updated_at}`).
- `GET /entries/{id}` → full entry (adds `body`).
- `PATCH /entries/{id}` partial `{title?, body?, folder?}` → `entryView` with the
  patched fields + a bumped `updated_at`. Body present re-seals via KMS; folder
  present moves with no KMS round-trip. Folder is normalized server-side.
- `DELETE /entries/{id}` → 204.

## Architecture

Decompose the current monolithic `Entries.vue` into focused, independently
testable units. No Pinia entries store (state is single-view — local refs
suffice; YAGNI).

### `web/src/lib/api.ts`
- `EntryMeta` gains `folder: string` and `updated_at: string`. `Entry` extends it
  (adds `body`).
- `create(title, body, folder)` — sends `folder`.
- `update(id, { title, body })` → `PATCH /entries/{id}` (re-seals body).
- `move(id, folder)` → `PATCH /entries/{id}` with `{ folder }` only.
- Both PATCH calls return the updated view; callers refresh local state from it.

### `web/src/lib/tree.ts` — core new logic
Pure `buildTree(metas: EntryMeta[]): TreeNode`. Each entry's `folder` is a
`/`-separated path (e.g. `Personal/Health`). Produces a nested structure of
folders and entries:
- Root entries (`folder === ""`) sit at the top level.
- Within a node, folders sort before entries (alphabetically by name).
- Entries sort newest-first by `created_at` (consistent with ULID ordering).

Unit-tested: empty list, root-only, nested paths, mixed folder/entry sort order.

### `web/src/lib/markdown.ts`
`renderMarkdown(src: string): string` — parse with **`marked`**, sanitize the
resulting HTML with **`DOMPurify`** before it is bound via `v-html`. Used by both
the read view and the editor preview. (Content is the user's own private journal,
but sanitizing is cheap insurance.)

### Components
- **`FolderTree.vue`** + recursive **`FolderTreeNode.vue`** — the expandable
  sidebar. Tracks expanded-folder state as a `Set<string>` of paths. Emits
  `select(entry)`. Exposes the "current folder" (the focused/active folder path)
  so `+ new` composes into it.
- **`EntryEditor.vue`** — live split preview used for both compose and edit:
  title input + markdown `<textarea>` on the left, `renderMarkdown` preview on the
  right updating as you type. Emits `save({ title, body })` / `cancel`.
- **`MoveDialog.vue`** — folder picker: lists existing folder paths (derived from
  the tree) plus a free-text field for a new path. Emits `move(folder)`.

### `web/src/views/Entries.vue` (orchestrator)
Holds list / selected / mode (`view | compose | edit`) state, owns the api calls,
and composes the above. Move and edit update local state optimistically from the
PATCH response so the tree re-derives without a full refetch.

## Data flow

1. On mount, `api.list()` → flat `EntryMeta[]` → `buildTree` → `FolderTree`.
2. Selecting an entry → `api.get(id)` → rendered via `renderMarkdown` in the read
   view.
3. `+ new` (into current folder) or `edit` → `EntryEditor`; `save` calls
   `api.create(..., folder)` or `api.update(id, ...)`; local list updated from the
   response; mode returns to `view`.
4. `move` → `MoveDialog` → `api.move(id, folder)`; the entry's `folder` updated
   locally; tree re-derives.
5. `delete` → `api.delete(id)`; removed from local list.

## Error handling

Follow the existing pattern in `Entries.vue`: a single `error` ref surfaced in the
UI; `api.ts` already maps 401 → sign-out and non-OK → thrown error. Save/move
buttons guard against double-submit while a request is in flight (mirrors the
TUI's `saving` guard).

## Testing

- `tree.test.ts` — the tree builder (highest-value; pure logic).
- Extend `api.test.ts` — `update`/`move` issue the correct PATCH bodies and path;
  `create` passes `folder`.
- Component tests kept light (existing setup: vitest + jsdom).
- `npm run typecheck` (vue-tsc) and `npm run build` clean.

## Out of scope (YAGNI — tracked in `docs/backlog.md`)

- Search / filter.
- Drag-and-drop move (folder picker only this round).
- Folder rename / delete.
- Tags, attachments/images.
- Pinia entries store.
