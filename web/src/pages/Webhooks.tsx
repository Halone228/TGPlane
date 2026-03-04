import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { webhooksApi } from '../api/client'
import { Trash2, Plus } from 'lucide-react'

export default function Webhooks() {
  const qc = useQueryClient()
  const [url, setUrl] = useState('')
  const [secret, setSecret] = useState('')
  const [events, setEvents] = useState('')

  const { data = [] } = useQuery({ queryKey: ['webhooks'], queryFn: webhooksApi.list })

  const create = useMutation({
    mutationFn: () =>
      webhooksApi.create(
        url,
        secret,
        events ? events.split(',').map((e) => e.trim()).filter(Boolean) : [],
      ),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['webhooks'] })
      setUrl(''); setSecret(''); setEvents('')
    },
  })

  const remove = useMutation({
    mutationFn: (id: number) => webhooksApi.remove(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['webhooks'] }),
  })

  return (
    <div className="p-8">
      <h1 className="text-2xl font-bold text-white mb-6">Webhooks</h1>

      {/* Create form */}
      <div className="bg-gray-900 border border-gray-800 rounded-xl p-5 mb-6">
        <h2 className="text-sm font-semibold text-gray-300 mb-4">Register Webhook</h2>
        <form
          className="grid grid-cols-1 sm:grid-cols-3 gap-3"
          onSubmit={(e) => { e.preventDefault(); url && create.mutate() }}
        >
          <input
            placeholder="https://example.com/hook"
            value={url}
            onChange={(e) => setUrl(e.target.value)}
            className="bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-sm text-white placeholder-gray-500 focus:outline-none focus:border-indigo-500 sm:col-span-1"
          />
          <input
            placeholder="HMAC secret (optional)"
            value={secret}
            onChange={(e) => setSecret(e.target.value)}
            className="bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-sm text-white placeholder-gray-500 focus:outline-none focus:border-indigo-500"
          />
          <input
            placeholder="events: message,photo (empty = all)"
            value={events}
            onChange={(e) => setEvents(e.target.value)}
            className="bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-sm text-white placeholder-gray-500 focus:outline-none focus:border-indigo-500"
          />
          <button
            type="submit"
            disabled={create.isPending}
            className="sm:col-span-3 flex items-center justify-center gap-2 px-4 py-2 bg-indigo-600 hover:bg-indigo-500 text-white text-sm font-medium rounded-lg transition-colors disabled:opacity-50"
          >
            <Plus size={14} /> Register
          </button>
        </form>
      </div>

      {/* Table */}
      <div className="bg-gray-900 border border-gray-800 rounded-xl overflow-hidden">
        {!data.length ? (
          <p className="p-5 text-gray-500 text-sm">No webhooks registered.</p>
        ) : (
          <table className="w-full text-sm">
            <thead>
              <tr className="text-left text-gray-500 border-b border-gray-800">
                <th className="px-5 py-3">URL</th>
                <th className="px-5 py-3">Events</th>
                <th className="px-5 py-3">Created</th>
                <th className="px-5 py-3" />
              </tr>
            </thead>
            <tbody>
              {data.map((w) => (
                <tr key={w.id} className="border-b border-gray-800 last:border-0">
                  <td className="px-5 py-3 font-mono text-xs text-blue-400 truncate max-w-xs">
                    {w.url}
                  </td>
                  <td className="px-5 py-3 text-gray-400 text-xs">
                    {w.events?.length ? w.events.join(', ') : 'all'}
                  </td>
                  <td className="px-5 py-3 text-gray-500">
                    {new Date(w.created_at).toLocaleDateString()}
                  </td>
                  <td className="px-5 py-3">
                    <button
                      onClick={() => remove.mutate(w.id)}
                      className="text-gray-600 hover:text-red-400 transition-colors"
                    >
                      <Trash2 size={14} />
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>
    </div>
  )
}
