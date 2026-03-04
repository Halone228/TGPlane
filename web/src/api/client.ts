import axios from 'axios'

const api = axios.create({ baseURL: '/api/v1' })

// Inject API key from localStorage on every request.
api.interceptors.request.use((config) => {
  const key = localStorage.getItem('apiKey')
  if (key) config.headers['X-Api-Key'] = key
  return config
})

export default api

// ── Types ──────────────────────────────────────────────────────────────────

export interface Account {
  id: number
  phone: string
  session_id: string
  status: string
  first_name?: string
  last_name?: string
  username?: string
  tg_user_id?: number
  worker_id?: string
  created_at: string
  updated_at: string
}

export interface Bot {
  id: number
  token: string
  session_id: string
  status: string
  username?: string
  tg_user_id?: number
  worker_id?: string
  created_at: string
  updated_at: string
}

export interface Session {
  id: string
  type: 'account' | 'bot'
  status: string
}

export interface WorkerMetrics {
  worker_id: string
  session_count: number
  ready_count: number
  error_count: number
  updates_total: number
  collected_at: number
}

export interface ApiKey {
  id: number
  name: string
  key_prefix: string
  created_at: string
}

export interface Webhook {
  id: number
  url: string
  events: string[]
  created_at: string
}

export interface BulkResult {
  total: number
  succeeded: number
  failed: number
  items: Array<{ index: number; ok: boolean; error?: string; data?: unknown }>
}

// ── Accounts ───────────────────────────────────────────────────────────────

export const accountsApi = {
  list: (params?: { limit?: number; offset?: number }) =>
    api.get<Account[]>('/accounts', { params }).then((r) => r.data),
  get: (id: number) => api.get<Account>(`/accounts/${id}`).then((r) => r.data),
  create: (phone: string) =>
    api.post<Account>('/accounts', { phone }).then((r) => r.data),
  remove: (id: number) => api.delete(`/accounts/${id}`),
}

// ── Bots ───────────────────────────────────────────────────────────────────

export const botsApi = {
  list: (params?: { limit?: number; offset?: number }) =>
    api.get<Bot[]>('/bots', { params }).then((r) => r.data),
  get: (id: number) => api.get<Bot>(`/bots/${id}`).then((r) => r.data),
  create: (token: string) =>
    api.post<Bot>('/bots', { token }).then((r) => r.data),
  remove: (id: number) => api.delete(`/bots/${id}`),
}

// ── Sessions ───────────────────────────────────────────────────────────────

export const sessionsApi = {
  list: () => api.get<Session[]>('/sessions').then((r) => r.data),
  get: (id: string) => api.get<Session>(`/sessions/${id}`).then((r) => r.data),
  stop: (id: string) => api.delete(`/sessions/${id}`),
}

// ── Workers ────────────────────────────────────────────────────────────────

export const workersApi = {
  list: () => api.get<{ workers: string[] }>('/workers').then((r) => r.data.workers),
  metrics: () => api.get<WorkerMetrics[]>('/workers/metrics').then((r) => r.data),
  add: (id: string, addr: string) =>
    api.post('/workers', { id, addr }).then((r) => r.data),
  remove: (id: string) => api.delete(`/workers/${id}`),
  drain: (id: string) =>
    api.post<{ migrated: number }>(`/workers/${id}/drain`).then((r) => r.data),
}

// ── API Keys ───────────────────────────────────────────────────────────────

export const apiKeysApi = {
  list: () => api.get<ApiKey[]>('/auth/keys').then((r) => r.data),
  create: (name: string) =>
    api.post<ApiKey & { key: string }>('/auth/keys', { name }).then((r) => r.data),
  remove: (id: number) => api.delete(`/auth/keys/${id}`),
}

// ── Webhooks ───────────────────────────────────────────────────────────────

export const webhooksApi = {
  list: () => api.get<Webhook[]>('/webhooks').then((r) => r.data),
  create: (url: string, secret: string, events: string[]) =>
    api.post<Webhook>('/webhooks', { url, secret, events }).then((r) => r.data),
  remove: (id: number) => api.delete(`/webhooks/${id}`),
}

// ── Bulk ───────────────────────────────────────────────────────────────────

export const bulkApi = {
  addBots: (tokens: string[]) =>
    api
      .post<BulkResult>('/bulk/bots', { items: tokens.map((t) => ({ token: t })) })
      .then((r) => r.data),
  addAccounts: (phones: string[]) =>
    api
      .post<BulkResult>('/bulk/accounts', {
        items: phones.map((p) => ({ phone: p })),
      })
      .then((r) => r.data),
  removeSessions: (ids: string[]) =>
    api
      .delete<BulkResult>('/bulk/sessions', { data: { session_ids: ids } })
      .then((r) => r.data),
}
