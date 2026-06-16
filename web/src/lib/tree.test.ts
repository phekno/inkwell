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
