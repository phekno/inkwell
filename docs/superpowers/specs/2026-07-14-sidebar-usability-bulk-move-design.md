# Sidebar Usability + Bulk Move — Design

**Date:** 2026-07-14
**Scope:** web app only (`web/`). No API changes.

Three items: a folder-tree ergonomics fix, a Move-dialog rendering bug fix, and a
new select-mode + bulk-move feature.

## 1. `[ + new here ]` at the top of expanded folders

**Problem:** In `web/src/components/FolderTreeNode.vue`, the `[ + new here ]`
button renders after all subfolders and entries. Folders with many entries
(40–50 per year) require scrolling to the bottom to create an entry.

**Change:** Move the button to be the first element rendered when a folder is
expanded — above subfolders and entries, indented one level (`pad(depth + 1)`)
as today. Template reorder only; no logic changes.

## 2. Move dialog renders transparent

**Problem:** The Move dialog panel sometimes appears to have no background,
letting the page content show through.

**Diagnosis:** The panel background is *always* transparent — it's just only
noticeable over busy content. `.pane` in `web/src/style.css` sets
`background: transparent` as **unlayered** CSS. Under CSS cascade layers
(Tailwind v4), unlayered author CSS beats layered utilities, so the
`bg-ink-50 dark:bg-ink-900` classes on the dialog panel never apply.

**Change:** Wrap the custom component classes (`.pane`, `.btn-term`,
`.input-term`) in `@layer components` in `style.css`. With Tailwind's layer
order (`base → components → utilities`), utility classes then correctly
override them, restoring the dialog's solid background in both light and dark
themes. No component changes.

## 3. Bulk move via select mode

**Problem:** Entries can only be moved one at a time. Reorganizing years of
entries (400+) into folders is impractical.

**UX (approved):** Explicit select mode with terminal-style checkboxes.

```
┌ sidebar ──────────────────┐
│ 412 entries   [ + new ]   │
│ [ select ]                │
├───────────────────────────┤
│ ▾ 📁 2024                 │
│    [ + new here ]         │
│    [x] Jan trip notes     │
│    [ ] Feb retro          │
│    [x] March ideas        │
├───────────────────────────┤
│ 2 selected                │
│ [ move 2 ]  [ cancel ]    │
└───────────────────────────┘
```

**Behavior:**

- A `[ select ]` button in the sidebar header (shown only when entries exist)
  toggles select mode on.
- In select mode, clicking an entry in the tree toggles its `[ ]`/`[x]`
  checkbox instead of opening it. Folders still expand/collapse normally.
  `[ + new here ]` remains functional.
- A sticky action bar at the bottom of the sidebar shows `N selected`,
  `[ move N ]`, and `[ cancel ]`.
- `[ cancel ]` exits select mode and clears the selection.
- `[ move N ]` opens the existing `MoveDialog` with `current=""` (selections
  can span folders). On confirm, the selected entries are moved; on success the
  app exits select mode and clears the selection.

**Implementation:**

- **State** in `web/src/views/Entries.vue`: `selectMode: boolean` and
  `selectedIds: Set<string>` (a `ref` replaced immutably on toggle so Vue
  reactivity is straightforward).
- **Plumbing:** the select-mode flag, selected set, and a toggle function flow
  to tree nodes via provide/inject, matching the existing
  `web/src/components/folderTreeKeys.ts` pattern (new injection keys alongside
  `NewKey`/`SelectKey`/`SelectedIdKey`).
- **Move execution:** loop the existing `api.move(id, folder)` over selected
  ids with `Promise.allSettled`. Apply successes to the local `list` (update
  `folder` and `updated_at`); if any fail, show `moved X of N` in the existing
  error line and keep the failed entries selected so the user can retry.
  No backend/bulk endpoint — 40–50 sequential-ish PATCHes is fine at this
  scale (YAGNI).
- **Interaction with single-entry Move:** the existing per-entry `[ move ]`
  button and `doMove` flow are unchanged; `MoveDialog` is reused as-is for
  both paths.

## Error handling

- Bulk move partial failure: `moved X of N` in the error line; failed entries
  stay selected, select mode stays active.
- Bulk move total failure: standard error message, selection untouched.

## Testing

- Unit tests (vitest) for extractable logic: applying bulk-move results to the
  entry list and computing the moved/failed split.
- Behavioral verification by driving the app: button position in a long
  folder, dialog solid background in light + dark, full select → move → verify
  flow.

## Out of scope

- Bulk delete, select-all, shift-click ranges (can layer on later).
- Backend bulk-move endpoint.
- TUI changes.
