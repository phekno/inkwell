# Notion → inkwell Importer — Design

Date: 2026-06-10
Status: Approved (pending implementation)

## Summary

A one-time CLI that imports a Notion **Markdown & CSV** export into inkwell by
sealing each page and writing it directly to DynamoDB — reusing the existing
`crypto.Sealer` and `store.Store`, with no HTTP API and no API changes.

## Why direct-to-DB (not via the API)

`store.Put` already accepts `CreatedAt`, so writing directly lets us preserve
each page's original Notion date. The HTTP `create` handler hard-codes "now",
and we don't want to add a client-settable `created_at`/`id` to the public API
for a one-time migration. Bodies must still be envelope-encrypted (KMS-wrapped
DEK + XChaCha20-Poly1305) or the TUI can't read them — so the tool seals exactly
as the API does.

## Verified preconditions (2026-06-10)

- Identity: `AdministratorAccess` via the `phekno` SSO profile.
- `kms:GenerateDataKey` on `alias/inkwell-entries` with a per-user
  `EncryptionContext{sub=…}` succeeds (no key-policy change needed).
- Table `inkwell-entries` is ACTIVE and holds **1 pre-existing entry** (a
  hand-made root entry). The import must not clobber it (it won't — see IDs).
- Cognito `sub` for the target user (josh@joshfechner.com):
  `d4284468-4091-70a5-f819-78e552c8f8df`.
- Region: `us-east-1`.

## Architecture

- `api/cmd/import/` — the CLI (flag parsing, filesystem walk, AWS wiring, the
  seal+Put loop).
- `api/internal/notion/` — pure, unit-tested parsing/transform logic.

## Parsing (`internal/notion`)

Per `.md` file:

- **Title** = the `# ` heading line (leading `# ` stripped).
- **Property block** = consecutive `Key: value` lines immediately after the
  title, up to the first blank line. Capture `Created:` and `Updated:`; drop all
  other keys (`Tags:` and any one-offs). Body text always follows the blank
  line, so this never eats content.
- **Body** = everything after the property block, trimmed.
- **Folder** = the file's directory path relative to the export root
  (`Journal`, `Work/OE and Debt`, …). Top-level files map to root (`""`).

## Which files import

- Inline `Created:` present → import (≈857 files).
- No `Created:` → attempt to parse a date from the title (full date like
  "February 17th, 2023", else "Month YYYY" → first of month). Parses → import
  with that date (≈20 `Work/OE and Debt` pages). Doesn't parse → **skip**
  (≈6 structural/index stubs: Work, Home, the "Personal" stub, Joshua Fechner).
- Every skip and every title-dated entry is logged.

## Timestamps & deterministic IDs

- `Created:` / `Updated:` parsed with layout `"January 2, 2006 3:04 PM"` in the
  machine's local timezone; `UpdatedAt` falls back to `CreatedAt` when absent.
- **ID = a ULID whose timestamp is the entry's `CreatedAt` and whose entropy is
  `sha256(folder + "\x00" + title)` (first 10 bytes).** This is:
  - *deterministic* — re-running produces identical IDs, so a partial run is
    safe to resume and never duplicates;
  - *chronological* — ULIDs sort by their timestamp, so imported entries list
    newest-first by original date, exactly like normal entries.
- Tags (and other Notion properties) are dropped; folders already provide
  grouping.

## CLI

```
import <export-dir> --sub <cognito-sub> \
    [--table <name>     | $ENTRIES_TABLE] \
    [--kms-key-id <id>  | $KMS_KEY_ID]    \
    [--region <region>] [--dry-run]
```

- `--dry-run` parses everything and prints the plan (total, per-folder counts,
  the skip list, the title-dated list) with **no AWS calls** — run first.
- Real run seals each body and `Put`s it. AWS creds come from the standard SDK
  chain (e.g. `AWS_PROFILE=phekno`).

## Testing (TDD)

Pure functions in `internal/notion`:

- property-block parsing: drops `Tags:`/other keys, captures `Created`/`Updated`,
  preserves body, handles a page with no property block;
- `folderFromPath`;
- `dateFromTitle`: the OE title variants + a no-date failure case;
- deterministic ULID: identical across runs, and orders by `CreatedAt`.

The IO loop is thin and exercised via `--dry-run` against the real export.

## Out of scope (YAGNI)

- Live Notion API sync (export-file route chosen).
- Tag preservation.
- Importing structural/index stub pages.
- Attachments/images embedded in Notion pages (text bodies only).
