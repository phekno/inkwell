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
