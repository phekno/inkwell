import { useAuthStore } from '../stores/auth'

const base = import.meta.env.VITE_API_URL.replace(/\/$/, '')

async function call<T>(method: string, path: string, body?: unknown): Promise<T> {
  const auth = useAuthStore()
  const token = auth.idToken
  if (!token) throw new Error('not signed in')

  const res = await fetch(`${base}${path}`, {
    method,
    headers: {
      authorization: `Bearer ${token}`,
      ...(body ? { 'content-type': 'application/json' } : {}),
    },
    body: body ? JSON.stringify(body) : undefined,
  })

  if (res.status === 401) {
    auth.signOut()
    throw new Error('session expired')
  }
  if (!res.ok) {
    const text = await res.text()
    throw new Error(`${res.status}: ${text}`)
  }
  if (res.status === 204) return undefined as T
  return (await res.json()) as T
}

export interface EntryMeta {
  id: string
  title: string
  created_at: string
}

export interface Entry extends EntryMeta {
  body: string
}

export const api = {
  list: () => call<EntryMeta[]>('GET', '/entries'),
  get: (id: string) => call<Entry>('GET', `/entries/${id}`),
  create: (title: string, body: string) =>
    call<EntryMeta>('POST', '/entries', { title, body }),
  delete: (id: string) => call<void>('DELETE', `/entries/${id}`),
}
