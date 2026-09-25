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

// normalizeFolder cleans a user-typed folder path: trims each segment and drops
// empty ones, so ' /Work// 2026 /' becomes 'Work/2026' and blank becomes ''.
export function normalizeFolder(path: string): string {
  return path
    .split('/')
    .map((s) => s.trim())
    .filter((s) => s.length > 0)
    .join('/')
}

// ensurePath walks (creating as needed) the folder nodes for a slash path.
function ensurePath(root: TreeNode, folder: string): TreeNode {
  let node = root
  const segments = folder.split('/').filter((s) => s.length > 0)
  for (const seg of segments) {
    const childPath = node.path ? `${node.path}/${seg}` : seg
    let child = node.folders.find((f) => f.name === seg)
    if (!child) {
      child = newNode(seg, childPath)
      node.folders.push(child)
    }
    node = child
  }
  return node
}

// buildTree turns the flat entry list into a nested folder structure. Each
// entry's `folder` is a slash path; empty means the root. Word-named folders
// sort alphabetically; purely numeric siblings (year folders like 2026) sort
// descending so the newest year is on top, matching the newest-first entries.
//
// Folders only exist server-side as entry paths, so `extraFolders` lets the
// caller show folders the user just created that have no entries yet.
export function buildTree(metas: EntryMeta[], extraFolders: string[] = []): TreeNode {
  const root = newNode('', '')

  for (const m of metas) ensurePath(root, m.folder).entries.push(m)
  for (const f of extraFolders) ensurePath(root, normalizeFolder(f))

  sortNode(root)
  return root
}

const numeric = (s: string) => /^\d+$/.test(s)

function compareFolderNames(a: string, b: string): number {
  const an = numeric(a)
  const bn = numeric(b)
  if (an && bn) return Number(b) - Number(a) // years newest-first
  if (an !== bn) return an ? -1 : 1 // numeric before words
  return a.localeCompare(b)
}

function sortNode(node: TreeNode): void {
  node.folders.sort((a, b) => compareFolderNames(a.name, b.name))
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
