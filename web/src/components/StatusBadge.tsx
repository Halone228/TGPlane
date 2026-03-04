const colors: Record<string, string> = {
  ready: 'bg-green-500/20 text-green-400',
  pending: 'bg-yellow-500/20 text-yellow-400',
  authorizing: 'bg-blue-500/20 text-blue-400',
  disconnected: 'bg-gray-500/20 text-gray-400',
  error: 'bg-red-500/20 text-red-400',
}

export default function StatusBadge({ status }: { status: string }) {
  const cls = colors[status] ?? 'bg-gray-500/20 text-gray-400'
  return (
    <span className={`inline-flex items-center px-2 py-0.5 rounded text-xs font-medium ${cls}`}>
      {status}
    </span>
  )
}
