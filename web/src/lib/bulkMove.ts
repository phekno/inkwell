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
