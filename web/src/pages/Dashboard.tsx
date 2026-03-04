import { useQuery } from '@tanstack/react-query'
import { accountsApi, botsApi, sessionsApi, workersApi } from '../api/client'
import { Users, Bot, Radio, Server } from 'lucide-react'

function StatCard({
  icon: Icon,
  label,
  value,
  color,
}: {
  icon: React.ElementType
  label: string
  value: number | string
  color: string
}) {
  return (
    <div className="bg-gray-900 border border-gray-800 rounded-xl p-5 flex items-center gap-4">
      <div className={`p-3 rounded-lg ${color}`}>
        <Icon size={20} />
      </div>
      <div>
        <p className="text-sm text-gray-400">{label}</p>
        <p className="text-2xl font-bold text-white">{value}</p>
      </div>
    </div>
  )
}

export default function Dashboard() {
  const accounts = useQuery({ queryKey: ['accounts'], queryFn: () => accountsApi.list() })
  const bots = useQuery({ queryKey: ['bots'], queryFn: () => botsApi.list() })
  const sessions = useQuery({ queryKey: ['sessions'], queryFn: () => sessionsApi.list() })
  const workers = useQuery({ queryKey: ['workers'], queryFn: () => workersApi.list() })

  return (
    <div className="p-8">
      <h1 className="text-2xl font-bold text-white mb-6">Dashboard</h1>

      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4 mb-8">
        <StatCard
          icon={Users}
          label="Accounts"
          value={accounts.data?.length ?? '–'}
          color="bg-indigo-500/20 text-indigo-400"
        />
        <StatCard
          icon={Bot}
          label="Bots"
          value={bots.data?.length ?? '–'}
          color="bg-purple-500/20 text-purple-400"
        />
        <StatCard
          icon={Radio}
          label="Active Sessions"
          value={sessions.data?.length ?? '–'}
          color="bg-green-500/20 text-green-400"
        />
        <StatCard
          icon={Server}
          label="Workers"
          value={workers.data?.length ?? '–'}
          color="bg-blue-500/20 text-blue-400"
        />
      </div>

      <WorkerMetricsTable />
    </div>
  )
}

function WorkerMetricsTable() {
  const { data, isLoading } = useQuery({
    queryKey: ['worker-metrics'],
    queryFn: workersApi.metrics,
    refetchInterval: 10_000,
  })

  return (
    <div className="bg-gray-900 border border-gray-800 rounded-xl">
      <div className="px-5 py-4 border-b border-gray-800">
        <h2 className="font-semibold text-white">Worker Health</h2>
      </div>
      {isLoading ? (
        <p className="p-5 text-gray-500 text-sm">Loading…</p>
      ) : !data?.length ? (
        <p className="p-5 text-gray-500 text-sm">No workers connected.</p>
      ) : (
        <table className="w-full text-sm">
          <thead>
            <tr className="text-left text-gray-500 border-b border-gray-800">
              <th className="px-5 py-3">Worker</th>
              <th className="px-5 py-3">Sessions</th>
              <th className="px-5 py-3">Ready</th>
              <th className="px-5 py-3">Errors</th>
              <th className="px-5 py-3">Updates</th>
            </tr>
          </thead>
          <tbody>
            {data.map((w) => (
              <tr key={w.worker_id} className="border-b border-gray-800 last:border-0">
                <td className="px-5 py-3 font-mono text-indigo-400">{w.worker_id}</td>
                <td className="px-5 py-3">{w.session_count}</td>
                <td className="px-5 py-3 text-green-400">{w.ready_count}</td>
                <td className="px-5 py-3 text-red-400">{w.error_count}</td>
                <td className="px-5 py-3">{w.updates_total.toLocaleString()}</td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </div>
  )
}
