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
