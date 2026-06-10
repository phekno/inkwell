import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'

// Mock the cognito module so api.ts doesn't go anywhere near real Amplify.
vi.mock('./cognito', () => ({
  fetchIdToken: vi.fn(async () => 'tok'),
}))

import { api } from './api'
import { fetchIdToken } from './cognito'
import { useAuthStore } from '../stores/auth'

const realFetch = globalThis.fetch
const mockedFetchIdToken = fetchIdToken as ReturnType<typeof vi.fn>

beforeEach(() => {
  setActivePinia(createPinia())
  mockedFetchIdToken.mockResolvedValue('tok')
})

afterEach(() => {
  globalThis.fetch = realFetch
  vi.restoreAllMocks()
})

function mockFetch(response: { status: number; body?: unknown }) {
  const fetchMock = vi.fn<typeof fetch>(async () =>
    new Response(response.body == null ? null : JSON.stringify(response.body), {
      status: response.status,
      headers: { 'content-type': 'application/json' },
    }),
  )
  globalThis.fetch = fetchMock
  return fetchMock
}

describe('api client', () => {
  it('sends Bearer auth on list', async () => {
    const f = mockFetch({ status: 200, body: [{ id: 'a', title: 't', created_at: '2026-01-01' }] })
    const got = await api.list()
    expect(got).toHaveLength(1)
    const init = f.mock.calls[0][1] as RequestInit
    expect(init.headers).toMatchObject({ authorization: 'Bearer tok' })
  })

  it('encodes JSON on create', async () => {
    const f = mockFetch({ status: 201, body: { id: 'x', title: 't', created_at: '2026-01-01' } })
    await api.create('hello', 'body')
    const init = f.mock.calls[0][1] as RequestInit
    expect(init.body).toBe(JSON.stringify({ title: 'hello', body: 'body' }))
    expect(init.headers).toMatchObject({ 'content-type': 'application/json' })
  })

  it('signs the user out on 401', async () => {
    mockFetch({ status: 401 })
    const auth = useAuthStore()
    const signOutSpy = vi.spyOn(auth, 'signOut').mockResolvedValue()
    await expect(api.list()).rejects.toThrow('session expired')
    expect(signOutSpy).toHaveBeenCalled()
  })

  it('returns undefined for 204', async () => {
    mockFetch({ status: 204 })
    await expect(api.delete('abc')).resolves.toBeUndefined()
  })

  it('throws when no token is available', async () => {
    mockedFetchIdToken.mockResolvedValueOnce(null)
    const auth = useAuthStore()
    vi.spyOn(auth, 'signOut').mockResolvedValue()
    await expect(api.list()).rejects.toThrow('not signed in')
  })
})
