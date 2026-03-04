import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { apiKeysApi } from '../api/client'
import { Trash2, Plus, Copy, Check } from 'lucide-react'

export default function ApiKeys() {
  const qc = useQueryClient()
  const [name, setName] = useState('')
  const [newKey, setNewKey] = useState<string | null>(null)
  const [copied, setCopied] = useState(false)

  const { data = [] } = useQuery({ queryKey: ['api-keys'], queryFn: apiKeysApi.list })

  const create = useMutation({
    mutationFn: (n: string) => apiKeysApi.create(n),
    onSuccess: (data) => {
      qc.invalidateQueries({ queryKey: ['api-keys'] })
      setName('')
      setNewKey(data.key)
    },
  })

  const remove = useMutation({
    mutationFn: (id: number) => apiKeysApi.remove(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['api-keys'] }),
  })

  const copy = () => {
    if (newKey) {
      navigator.clipboard.writeText(newKey)
      setCopied(true)
      setTimeout(() => setCopied(false), 2000)
    }
  }

  return (
    <div className="p-8">
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-2xl font-bold text-white">API Keys</h1>
        <form
          className="flex gap-2"
          onSubmit={(e) => { e.preventDefault(); name && create.mutate(name) }}
        >
          <input
            placeholder="key name"
            value={name}
            onChange={(e) => setName(e.target.value)}
            className="bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-sm text-white placeholder-gray-500 focus:outline-none focus:border-indigo-500"
          />
          <button
            type="submit"
            disabled={create.isPending}
            className="flex items-center gap-2 px-4 py-2 bg-indigo-600 hover:bg-indigo-500 text-white text-sm font-medium rounded-lg transition-colors disabled:opacity-50"
          >
            <Plus size={14} /> Create
          </button>
        </form>
      </div>

      {newKey && (
        <div className="mb-6 p-4 bg-green-900/30 border border-green-700 rounded-xl">
          <p className="text-sm text-green-400 mb-2 font-medium">
            Key created — copy it now, it won't be shown again.
          </p>
          <div className="flex items-center gap-3">
            <code className="flex-1 text-xs font-mono text-white bg-gray-900 px-3 py-2 rounded-lg break-all">
              {newKey}
            </code>
            <button onClick={copy} className="text-green-400 hover:text-green-300 transition-colors">
              {copied ? <Check size={16} /> : <Copy size={16} />}
            </button>
          </div>
        </div>
      )}

      <div className="bg-gray-900 border border-gray-800 rounded-xl overflow-hidden">
        {!data.length ? (
          <p className="p-5 text-gray-500 text-sm">No API keys yet.</p>
        ) : (
          <table className="w-full text-sm">
            <thead>
              <tr className="text-left text-gray-500 border-b border-gray-800">
                <th className="px-5 py-3">Name</th>
                <th className="px-5 py-3">Prefix</th>
                <th className="px-5 py-3">Created</th>
                <th className="px-5 py-3" />
              </tr>
            </thead>
            <tbody>
              {data.map((k) => (
                <tr key={k.id} className="border-b border-gray-800 last:border-0">
                  <td className="px-5 py-3 font-medium text-white">{k.name}</td>
                  <td className="px-5 py-3 font-mono text-xs text-gray-400">{k.key_prefix}…</td>
                  <td className="px-5 py-3 text-gray-500">
                    {new Date(k.created_at).toLocaleDateString()}
                  </td>
                  <td className="px-5 py-3">
                    <button
                      onClick={() => remove.mutate(k.id)}
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

      <p className="mt-4 text-xs text-gray-600">
        Set your active key in localStorage:{' '}
        <code className="font-mono">localStorage.setItem('apiKey', 'your-key')</code>
      </p>
    </div>
  )
}
