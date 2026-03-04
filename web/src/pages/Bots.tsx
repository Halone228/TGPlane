import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { botsApi, type Bot } from '../api/client'
import StatusBadge from '../components/StatusBadge'
import { Trash2, Plus } from 'lucide-react'

export default function Bots() {
  const qc = useQueryClient()
  const [token, setToken] = useState('')

  const { data = [], isLoading } = useQuery({
    queryKey: ['bots'],
    queryFn: () => botsApi.list(),
  })

  const create = useMutation({
    mutationFn: (t: string) => botsApi.create(t),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['bots'] }); setToken('') },
  })

  const remove = useMutation({
    mutationFn: (id: number) => botsApi.remove(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['bots'] }),
  })

  return (
    <div className="p-8">
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-2xl font-bold text-white">Bots</h1>
        <form
          className="flex gap-2"
          onSubmit={(e) => { e.preventDefault(); token && create.mutate(token) }}
        >
          <input
            type="text"
            placeholder="123456:ABC..."
            value={token}
            onChange={(e) => setToken(e.target.value)}
            className="bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-sm text-white placeholder-gray-500 focus:outline-none focus:border-indigo-500 w-64"
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

      {isLoading ? (
        <p className="text-gray-500 text-sm">Loading…</p>
      ) : !data.length ? (
        <p className="text-gray-500 text-sm">No bots yet.</p>
      ) : (
        <div className="bg-gray-900 border border-gray-800 rounded-xl overflow-hidden">
          <table className="w-full text-sm">
            <thead>
              <tr className="text-left text-gray-500 border-b border-gray-800">
                <th className="px-5 py-3">Username</th>
                <th className="px-5 py-3">Status</th>
                <th className="px-5 py-3">Worker</th>
                <th className="px-5 py-3">Token (prefix)</th>
                <th className="px-5 py-3">Created</th>
                <th className="px-5 py-3" />
              </tr>
            </thead>
            <tbody>
              {data.map((b: Bot) => (
                <tr key={b.id} className="border-b border-gray-800 last:border-0">
                  <td className="px-5 py-3 font-medium text-white">
                    {b.username ? `@${b.username}` : <span className="text-gray-500">–</span>}
                  </td>
                  <td className="px-5 py-3"><StatusBadge status={b.status} /></td>
                  <td className="px-5 py-3 text-gray-400 font-mono text-xs">{b.worker_id ?? '–'}</td>
                  <td className="px-5 py-3 text-gray-400 font-mono text-xs">
                    {b.token.split(':')[0]}:***
                  </td>
                  <td className="px-5 py-3 text-gray-500">{new Date(b.created_at).toLocaleDateString()}</td>
                  <td className="px-5 py-3">
                    <button
                      onClick={() => remove.mutate(b.id)}
                      className="text-gray-600 hover:text-red-400 transition-colors"
                    >
                      <Trash2 size={14} />
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  )
}
