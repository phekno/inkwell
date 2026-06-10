# Edit Entries & Folder Grouping — Design

Date: 2026-06-10
Status: Approved (pending implementation plan)

## Summary

Two features for the inkwell journal:

1. **Edit entries** — change an existing entry's title and body.
2. **Folders** — group entries into a nested, slash-path folder structure, navigated
   in the TUI as a drill-in browser.

Both are delivered through a single new partial-update API endpoint.

## Decisions

- **Folder model:** a folder is a plaintext slash-path attribute on the entry
  (e.g. `""`, `"work"`, `"work/journal"`). Chosen over first-class folder items
  and tags for minimal new surface; it composes with the edit feature.
- **Nesting:** nested via path string. Storage cost is zero (just a string); the
  TUI renders/drills the tree.
- **Folders are implicit:** a folder exists because some entry references its path.
  Accepted trade-off: **no empty folders**, no separate folder records, no
  distinct rename operation (rename = move entries).
- **TUI navigation:** drill-in browser with a current-folder breadcrumb.
- **Edit scope:** edit changes title + body. Moving an entry between folders is a
  **separate** action, not part of the edit form.

## 1. Data model & storage

Single-table DynamoDB is unchanged in key structure; the sort key stays
`ENTRY#<ulid>` so entry identity is stable.

```
PK = USER#<sub>   SK = ENTRY#<ulid>
attrs: title, ciphertext, nonce, wrapped_dek, created_at,
       folder      (S)  — "" | "work" | "work/journal"   [new]
       updated_at  (S, RFC3339)                          [new]
```

- `folder` is **plaintext**, consistent with `title` already being plaintext. It
  reveals structure but not content. Empty string = root.
- **Why an attribute, not part of the sort key:** a move becomes a one-attribute
  update instead of delete-and-reinsert, and it requires no KMS round-trip.
  Per-folder filtering is done client-side over the entry list (fine at personal
  scale).
- **No migration:** entries without a `folder` attribute read as root (`""`).
- `created_at` remains the list sort key — ordering is unchanged. `updated_at` is
  informational (may be shown in the view).

## 2. API

One new route, plus `updated_at` flowing through existing responses.

```
PATCH /entries/{id}   body: { title?, body?, folder? }   -> 200 entryView
```

Handler builds a DynamoDB `UpdateExpression` from whichever fields are present:

- `body` present  -> re-seal via KMS; update `ciphertext`/`nonce`/`wrapped_dek`.
- `title` present -> set `title` (reject blank title).
- `folder` present -> set `folder` (after path normalization).
- Always bump `updated_at`.
- 404 if the entry does not exist.

This single endpoint serves both features:

- **Edit form** sends `{title, body}`.
- **Move action** sends `{folder}` only — and because `folder` is a plaintext
  attribute, a move needs **no body and no KMS decrypt/re-encrypt**.

`store.Update` is one new method using `UpdateItem`. `store.List` adds `folder`
and `updated_at` to its projection; `EntryMeta` gains `Folder` and `UpdatedAt`.

## 3. TUI flows

The list view becomes a **drill-in browser** tracking a current folder path. Child
folders at the current level are computed client-side from entries' paths relative
to the current location.

```
inkwell — work/
  📁 journal/
  📝 planning doc
> enter open · .. up · n new · m move · e edit · d delete · q quit
```

- **Navigate:** `enter` on a folder drills in; `..`/backspace goes up. `enter` on
  an entry opens the existing view mode.
- **New (`n`):** composes into the **current folder** (folder inherited from the
  browser location — no folder field on the compose form). Composing into a
  never-seen path is how a new folder comes into existence.
- **Edit (`e`):** new mode reusing the compose form, prefilled with title + body;
  `ctrl+s` sends `PATCH {title, body}`.
- **Move (`m`):** a small prompt prefilled with the entry's current path; type any
  destination path (existing or not) -> `PATCH {folder}`.
- **Delete (`d`):** unchanged.

## 4. Edge cases & testing

- **Path normalization:** one helper that trims whitespace, collapses repeated
  slashes, strips leading/trailing slashes, so `work/`, `/work`, and `work` all
  normalize to `work`. Unit-tested.
- **Store tests:** `Update` builds the correct expression for each field subset
  (title-only, body-only re-seals, folder-only); `List` projects `folder` and
  `updated_at`; back-compat — an item with no `folder` attribute reads as root.
- **Handler tests:** PATCH with each field subset; empty body `{}` is a benign
  200; 404 for unknown id; blank title rejected.
- **TUI tests:** tree-building from a set of folder paths (root, nested, mixed);
  drill up/down; move updates the in-memory list.

## Out of scope (YAGNI)

- Empty folders (consequence of implicit folders).
- A distinct folder-rename operation (rename by moving entries; revisit later).
- Changing list sort order (`created_at` stays the sort key).
- Encrypting folder paths (consistent with title being plaintext today).
