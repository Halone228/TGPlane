import { useState, useEffect } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { accountsApi, type Account } from '../api/client'
import StatusBadge from '../components/StatusBadge'
import { Trash2, Plus, Send, Loader2, KeyRound, ShieldCheck } from 'lucide-react'

export default function Accounts() {
  const qc = useQueryClient()
  const [phone, setPhone] = useState('')

  const { data = [], isLoading } = useQuery({
    queryKey: ['accounts'],
    queryFn: () => accountsApi.list(),
    refetchInterval: 5000,
  })

  const create = useMutation({
    mutationFn: (p: string) => accountsApi.create(p),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['accounts'] }); setPhone('') },
  })

  const remove = useMutation({
    mutationFn: (id: number) => accountsApi.remove(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['accounts'] }),
  })

  return (
    <div className="p-8">
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-2xl font-bold text-white">Accounts</h1>
        <form
          className="flex gap-2"
          onSubmit={(e) => { e.preventDefault(); phone && create.mutate(phone) }}
        >
          <input
            type="text"
            placeholder="+79001234567"
            value={phone}
            onChange={(e) => setPhone(e.target.value)}
            className="bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-sm text-white placeholder-gray-500 focus:outline-none focus:border-indigo-500"
          />
          <button
            type="submit"
            disabled={create.isPending}
            className="flex items-center gap-2 px-4 py-2 bg-indigo-600 hover:bg-indigo-500 text-white text-sm font-medium rounded-lg transition-colors disabled:opacity-50"
          >
            <Plus size={14} /> Add
          </button>
        </form>
      </div>

      <AccountTable accounts={data} loading={isLoading} onRemove={(id) => remove.mutate(id)} />
    </div>
  )
}

function AuthPanel({ account }: { account: Account }) {
  const qc = useQueryClient()
  const [code, setCode] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')

  const { data: authState } = useQuery({
    queryKey: ['auth-state', account.id],
    queryFn: () => accountsApi.getAuthState(account.id),
    refetchInterval: 2000,
    enabled: account.status === 'authorizing',
  })

  const state = authState?.state ?? ''

  // Clear error on state change
  useEffect(() => { setError('') }, [state])

  // When account becomes ready, refresh the list
  useEffect(() => {
    if (state === 'ready') {
      qc.invalidateQueries({ queryKey: ['accounts'] })
    }
  }, [state, qc])

  const submitCode = useMutation({
    mutationFn: () => accountsApi.sendAuthCode(account.id, code),
    onSuccess: () => { setCode(''); setError('') },
    onError: (e: any) => setError(e.response?.data?.error ?? 'Failed to submit code'),
  })

  const submitPassword = useMutation({
    mutationFn: () => accountsApi.sendPassword(account.id, password),
    onSuccess: () => { setPassword(''); setError('') },
    onError: (e: any) => setError(e.response?.data?.error ?? 'Failed to submit password'),
  })

  if (state === 'ready') return null

  return (
    <tr className="border-b border-gray-800 last:border-0 bg-gray-800/30">
      <td colSpan={6} className="px-5 py-3">
        <div className="flex items-center gap-3 flex-wrap">
          {(state === 'waiting_phone' || state === '') && (
            <div className="flex items-center gap-2 text-gray-400 text-xs">
              <Loader2 size={14} className="animate-spin" />
              <span>Connecting to Telegram...</span>
            </div>
          )}

          {state === 'waiting_code' && (
            <form
              className="flex items-center gap-2"
              onSubmit={(e) => { e.preventDefault(); code && submitCode.mutate() }}
            >
              <ShieldCheck size={14} className="text-blue-400 shrink-0" />
              <span className="text-blue-400 text-xs shrink-0">Enter auth code:</span>
              <input
                type="text"
                inputMode="numeric"
                placeholder="12345"
                value={code}
                onChange={(e) => setCode(e.target.value)}
                autoFocus
                className="bg-gray-800 border border-gray-600 rounded px-2 py-1 text-sm text-white placeholder-gray-500 focus:outline-none focus:border-blue-500 w-24"
              />
              <button
                type="submit"
                disabled={submitCode.isPending || !code}
                className="flex items-center gap-1 px-3 py-1 bg-blue-600 hover:bg-blue-500 text-white text-xs font-medium rounded transition-colors disabled:opacity-50"
              >
                {submitCode.isPending ? <Loader2 size={12} className="animate-spin" /> : <Send size={12} />}
                Submit
              </button>
            </form>
          )}

          {state === 'waiting_password' && (
            <form
              className="flex items-center gap-2"
              onSubmit={(e) => { e.preventDefault(); password && submitPassword.mutate() }}
            >
              <KeyRound size={14} className="text-amber-400 shrink-0" />
              <span className="text-amber-400 text-xs shrink-0">Enter 2FA password:</span>
              <input
                type="password"
                placeholder="Password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                autoFocus
                className="bg-gray-800 border border-gray-600 rounded px-2 py-1 text-sm text-white placeholder-gray-500 focus:outline-none focus:border-amber-500 w-40"
              />
              <button
                type="submit"
                disabled={submitPassword.isPending || !password}
                className="flex items-center gap-1 px-3 py-1 bg-amber-600 hover:bg-amber-500 text-white text-xs font-medium rounded transition-colors disabled:opacity-50"
              >
                {submitPassword.isPending ? <Loader2 size={12} className="animate-spin" /> : <Send size={12} />}
                Submit
              </button>
            </form>
          )}

          {state === 'error' && (
            <span className="text-red-400 text-xs">Authorization failed. Try removing and re-adding the account.</span>
          )}

          {error && <span className="text-red-400 text-xs">{error}</span>}
        </div>
      </td>
    </tr>
  )
}

function AccountTable({
  accounts,
  loading,
  onRemove,
}: {
  accounts: Account[]
  loading: boolean
  onRemove: (id: number) => void
}) {
  if (loading) return <p className="text-gray-500 text-sm">Loading...</p>
  if (!accounts.length) return <p className="text-gray-500 text-sm">No accounts yet.</p>

  return (
    <div className="bg-gray-900 border border-gray-800 rounded-xl overflow-hidden">
      <table className="w-full text-sm">
        <thead>
          <tr className="text-left text-gray-500 border-b border-gray-800">
            <th className="px-5 py-3">Phone</th>
            <th className="px-5 py-3">Status</th>
            <th className="px-5 py-3">Worker</th>
            <th className="px-5 py-3">Session ID</th>
            <th className="px-5 py-3">Created</th>
            <th className="px-5 py-3" />
          </tr>
        </thead>
        <tbody>
          {accounts.map((a) => (
            <>
              <tr key={a.id} className="border-b border-gray-800 last:border-0">
                <td className="px-5 py-3 font-medium text-white">{a.phone}</td>
                <td className="px-5 py-3"><StatusBadge status={a.status} /></td>
                <td className="px-5 py-3 text-gray-400 font-mono text-xs">{a.worker_id ?? '-'}</td>
                <td className="px-5 py-3 text-gray-400 font-mono text-xs">{a.session_id.slice(0, 12)}...</td>
                <td className="px-5 py-3 text-gray-500">{new Date(a.created_at).toLocaleDateString()}</td>
                <td className="px-5 py-3">
                  <button
                    onClick={() => onRemove(a.id)}
                    className="text-gray-600 hover:text-red-400 transition-colors"
                  >
                    <Trash2 size={14} />
                  </button>
                </td>
              </tr>
              {a.status === 'authorizing' && <AuthPanel key={`auth-${a.id}`} account={a} />}
            </>
          ))}
        </tbody>
      </table>
    </div>
  )
}
