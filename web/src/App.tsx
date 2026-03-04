import { Routes, Route } from 'react-router-dom'
import Layout from './components/Layout'
import Dashboard from './pages/Dashboard'
import Accounts from './pages/Accounts'
import Bots from './pages/Bots'
import Sessions from './pages/Sessions'
import Workers from './pages/Workers'
import ApiKeys from './pages/ApiKeys'
import Webhooks from './pages/Webhooks'

export default function App() {
  return (
    <Routes>
      <Route element={<Layout />}>
        <Route path="/" element={<Dashboard />} />
        <Route path="/accounts" element={<Accounts />} />
        <Route path="/bots" element={<Bots />} />
        <Route path="/sessions" element={<Sessions />} />
        <Route path="/workers" element={<Workers />} />
        <Route path="/api-keys" element={<ApiKeys />} />
        <Route path="/webhooks" element={<Webhooks />} />
      </Route>
    </Routes>
  )
}
