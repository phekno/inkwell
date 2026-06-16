# inkwell — feature backlog

Ideas worth doing later, deferred to keep current work focused. Not commitments —
a parking lot. When one gets picked up, brainstorm it into its own spec under
`docs/superpowers/specs/`.

## Web UI

- **Search / filter entries** — full-text over titles (and maybe bodies) so the
  now-large dataset is searchable, not just browsable.
- **Drag-and-drop move** — drag an entry onto a folder in the tree, on top of the
  folder-picker move shipped in the parity work.
- **Folder rename / delete** — bulk reorganization beyond moving one entry at a
  time. (Folders are derived from entry paths, so this means re-pathing entries.)
- **Markdown editor niceties** — formatting toolbar, keyboard shortcuts, etc.
- **Pinia entries store** — only if entry state needs sharing across views; local
  refs suffice today.

## Cross-cutting (web + TUI + API)

- **Tags** — Notion tags were dropped on import; folders provide grouping for now.
  Revisit if folder-only organization proves limiting.
- **Attachments / images** — the Notion importer skipped embedded media (text
  bodies only). Supporting images would touch storage, the API, and both clients.
