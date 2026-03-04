import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { sessionsApi } from '../api/client'
import StatusBadge from '../components/StatusBadge'
import { Trash2 } from 'lucide-react'

export default function Sessions() {
  const qc = useQueryClient()
  const { data = [], isLoading } = useQuery({
    queryKey: ['sessions'],
    queryFn: sessionsApi.list,
    refetchInterval: 5_000,
  })

  const stop = useMutation({
    mutationFn: (id: string) => sessionsApi.stop(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['sessions'] }),
  })

  return (
    <div className="p-8">
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-2xl font-bold text-white">Sessions</h1>
        <span className="text-sm text-gray-500">{data.length} active</span>
      </div>

      {isLoading ? (
        <p className="text-gray-500 text-sm">Loading…</p>
      ) : !data.length ? (
        <p className="text-gray-500 text-sm">No active sessions.</p>
      ) : (
        <div className="bg-gray-900 border border-gray-800 rounded-xl overflow-hidden">
          <table className="w-full text-sm">
            <thead>
              <tr className="text-left text-gray-500 border-b border-gray-800">
                <th className="px-5 py-3">Session ID</th>
                <th className="px-5 py-3">Type</th>
                <th className="px-5 py-3">Status</th>
                <th className="px-5 py-3" />
              </tr>
            </thead>
            <tbody>
              {data.map((s) => (
                <tr key={s.id} className="border-b border-gray-800 last:border-0">
                  <td className="px-5 py-3 font-mono text-xs text-gray-300">{s.id}</td>
                  <td className="px-5 py-3">
                    <span className="text-xs font-medium text-gray-400 uppercase">{s.type}</span>
                  </td>
                  <td className="px-5 py-3"><StatusBadge status={s.status} /></td>
                  <td className="px-5 py-3">
                    <button
                      onClick={() => stop.mutate(s.id)}
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
