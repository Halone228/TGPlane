import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { workersApi } from '../api/client'
import { Trash2, Plus, ArrowDownToLine } from 'lucide-react'

export default function Workers() {
  const qc = useQueryClient()
  const [id, setId] = useState('')
  const [addr, setAddr] = useState('')

  const { data: workers = [] } = useQuery({
    queryKey: ['workers'],
    queryFn: workersApi.list,
  })

  const { data: metrics = [] } = useQuery({
    queryKey: ['worker-metrics'],
    queryFn: workersApi.metrics,
    refetchInterval: 10_000,
  })

  const add = useMutation({
    mutationFn: () => workersApi.add(id, addr),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['workers'] })
      setId(''); setAddr('')
    },
  })

  const remove = useMutation({
    mutationFn: (wid: string) => workersApi.remove(wid),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['workers'] }),
  })

  const drain = useMutation({
    mutationFn: (wid: string) => workersApi.drain(wid),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['workers', 'worker-metrics'] }),
  })

  const metricsMap = Object.fromEntries(metrics.map((m) => [m.worker_id, m]))

  return (
    <div className="p-8">
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-2xl font-bold text-white">Workers</h1>
        <form
          className="flex gap-2"
          onSubmit={(e) => { e.preventDefault(); id && addr && add.mutate() }}
        >
          <input
            placeholder="worker-id"
            value={id}
            onChange={(e) => setId(e.target.value)}
            className="bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-sm text-white placeholder-gray-500 focus:outline-none focus:border-indigo-500 w-36"
          />
          <input
            placeholder="host:50051"
            value={addr}
            onChange={(e) => setAddr(e.target.value)}
            className="bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-sm text-white placeholder-gray-500 focus:outline-none focus:border-indigo-500 w-40"
          />
          <button
            type="submit"
            disabled={add.isPending}
            className="flex items-center gap-2 px-4 py-2 bg-indigo-600 hover:bg-indigo-500 text-white text-sm font-medium rounded-lg transition-colors disabled:opacity-50"
          >
            <Plus size={14} /> Add
          </button>
        </form>
      </div>

      {!workers.length ? (
        <p className="text-gray-500 text-sm">No workers connected.</p>
      ) : (
        <div className="bg-gray-900 border border-gray-800 rounded-xl overflow-hidden">
          <table className="w-full text-sm">
            <thead>
              <tr className="text-left text-gray-500 border-b border-gray-800">
                <th className="px-5 py-3">Worker ID</th>
                <th className="px-5 py-3">Sessions</th>
                <th className="px-5 py-3">Ready</th>
                <th className="px-5 py-3">Errors</th>
                <th className="px-5 py-3">Updates</th>
                <th className="px-5 py-3" />
              </tr>
            </thead>
            <tbody>
              {workers.map((wid) => {
                const m = metricsMap[wid]
                return (
                  <tr key={wid} className="border-b border-gray-800 last:border-0">
                    <td className="px-5 py-3 font-mono text-indigo-400">{wid}</td>
                    <td className="px-5 py-3">{m?.session_count ?? '–'}</td>
                    <td className="px-5 py-3 text-green-400">{m?.ready_count ?? '–'}</td>
                    <td className="px-5 py-3 text-red-400">{m?.error_count ?? '–'}</td>
                    <td className="px-5 py-3">{m?.updates_total?.toLocaleString() ?? '–'}</td>
                    <td className="px-5 py-3 flex gap-3">
                      <button
                        onClick={() => drain.mutate(wid)}
                        disabled={drain.isPending}
                        title="Drain (reassign sessions and remove)"
                        className="text-gray-600 hover:text-yellow-400 transition-colors"
                      >
                        <ArrowDownToLine size={14} />
                      </button>
                      <button
                        onClick={() => remove.mutate(wid)}
                        title="Force remove"
                        className="text-gray-600 hover:text-red-400 transition-colors"
                      >
                        <Trash2 size={14} />
                      </button>
                    </td>
                  </tr>
                )
              })}
            </tbody>
          </table>
        </div>
      )}
    </div>
  )
}
