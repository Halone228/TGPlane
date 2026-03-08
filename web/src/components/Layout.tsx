import { useState } from 'react'
import { NavLink, Outlet } from 'react-router-dom'
import {
  Users,
  Bot,
  Radio,
  Server,
  Key,
  Webhook,
  LayoutDashboard,
  Settings,
  Check,
} from 'lucide-react'

const nav = [
  { to: '/', label: 'Dashboard', icon: LayoutDashboard },
  { to: '/accounts', label: 'Accounts', icon: Users },
  { to: '/bots', label: 'Bots', icon: Bot },
  { to: '/sessions', label: 'Sessions', icon: Radio },
  { to: '/workers', label: 'Workers', icon: Server },
  { to: '/api-keys', label: 'API Keys', icon: Key },
  { to: '/webhooks', label: 'Webhooks', icon: Webhook },
]

function ApiKeyModal({ onClose }: { onClose: () => void }) {
  const [val, setVal] = useState(localStorage.getItem('apiKey') ?? '')
  const save = () => {
    localStorage.setItem('apiKey', val.trim())
    onClose()
    window.location.reload()
  }
  return (
    <div className="fixed inset-0 bg-black/60 flex items-center justify-center z-50">
      <div className="bg-gray-900 border border-gray-700 rounded-xl p-6 w-[480px] shadow-xl">
        <h2 className="text-white font-semibold mb-4">Set API Key</h2>
        <input
          autoFocus
          type="text"
          value={val}
          onChange={(e) => setVal(e.target.value)}
          onKeyDown={(e) => e.key === 'Enter' && save()}
          placeholder="Paste your API key..."
          className="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-sm text-white placeholder-gray-500 focus:outline-none focus:border-indigo-500 font-mono"
        />
        <div className="flex justify-end gap-2 mt-4">
          <button onClick={onClose} className="px-4 py-2 text-sm text-gray-400 hover:text-white transition-colors">Cancel</button>
          <button onClick={save} className="flex items-center gap-2 px-4 py-2 bg-indigo-600 hover:bg-indigo-500 text-white text-sm font-medium rounded-lg transition-colors">
            <Check size={14} /> Save & Reload
          </button>
        </div>
      </div>
    </div>
  )
}

export default function Layout() {
  const [showModal, setShowModal] = useState(false)
  const hasKey = !!localStorage.getItem('apiKey')

  return (
    <div className="flex h-screen bg-gray-950 text-gray-100">
      {showModal && <ApiKeyModal onClose={() => setShowModal(false)} />}
      {/* Sidebar */}
      <aside className="w-56 flex-shrink-0 bg-gray-900 border-r border-gray-800 flex flex-col">
        <div className="px-5 py-4 border-b border-gray-800">
          <span className="text-lg font-bold tracking-tight text-white">TGPlane</span>
        </div>
        <nav className="flex-1 px-3 py-4 space-y-1">
          {nav.map(({ to, label, icon: Icon }) => (
            <NavLink
              key={to}
              to={to}
              end={to === '/'}
              className={({ isActive }) =>
                `flex items-center gap-3 px-3 py-2 rounded-md text-sm font-medium transition-colors ${
                  isActive
                    ? 'bg-indigo-600 text-white'
                    : 'text-gray-400 hover:text-white hover:bg-gray-800'
                }`
              }
            >
              <Icon size={16} />
              {label}
            </NavLink>
          ))}
        </nav>
        <div className="px-3 py-3 border-t border-gray-800">
          <button
            onClick={() => setShowModal(true)}
            className={`flex items-center gap-2 w-full px-3 py-2 rounded-md text-sm font-medium transition-colors ${
              hasKey ? 'text-gray-400 hover:text-white hover:bg-gray-800' : 'text-yellow-400 hover:text-yellow-300 hover:bg-gray-800'
            }`}
          >
            <Settings size={16} />
            {hasKey ? 'API Key' : 'Set API Key'}
          </button>
        </div>
      </aside>

      {/* Main */}
      <main className="flex-1 overflow-auto">
        <Outlet />
      </main>
    </div>
  )
}
