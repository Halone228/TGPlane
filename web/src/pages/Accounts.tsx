import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { accountsApi, type Account } from '../api/client'
import StatusBadge from '../components/StatusBadge'
import { Trash2, Plus } from 'lucide-react'

export default function Accounts() {
  const qc = useQueryClient()
  const [phone, setPhone] = useState('')

  const { data = [], isLoading } = useQuery({
    queryKey: ['accounts'],
    queryFn: () => accountsApi.list(),
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

function AccountTable({
  accounts,
  loading,
  onRemove,
}: {
  accounts: Account[]
  loading: boolean
  onRemove: (id: number) => void
}) {
  if (loading) return <p className="text-gray-500 text-sm">Loading…</p>
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
            <tr key={a.id} className="border-b border-gray-800 last:border-0">
              <td className="px-5 py-3 font-medium text-white">{a.phone}</td>
              <td className="px-5 py-3"><StatusBadge status={a.status} /></td>
              <td className="px-5 py-3 text-gray-400 font-mono text-xs">{a.worker_id ?? '–'}</td>
              <td className="px-5 py-3 text-gray-400 font-mono text-xs">{a.session_id.slice(0, 12)}…</td>
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
          ))}
        </tbody>
      </table>
    </div>
  )
}
