import { useEffect, useState } from 'react'
import { useAuth } from '../context/AuthContext.jsx'
import { api } from '../utils/api.js'
import { formatCurrency, formatDate } from '../utils/format.js'

export default function Profile() {
  const { user, signOut } = useAuth()
  const [balance, setBalance] = useState(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  useEffect(() => {
    const fetchBalance = async () => {
      try {
        setLoading(true)
        setError('')
        const data = await api.getBalance()
        if (data && data.balance) {
          setBalance(data.balance)
        }
      } catch (err) {
        setError(err.message || 'Failed to load balance')
      } finally {
        setLoading(false)
      }
    }
    fetchBalance()
  }, [])

  return (
    <section className="page narrow">
      <h1>Profile</h1>

      <div className="panel profile">
        <div className="avatar">{user?.name?.[0]?.toUpperCase()}</div>
        <div>
          <h2>{user?.name}</h2>
          <p className="muted">{user?.email}</p>
          <p className="muted small">
            Member since {formatDate(user?.joinedAt)}
          </p>
        </div>
      </div>

      {error && <div className="alert error">{error}</div>}

      <div className="panel">
        <h3>Wallet summary</h3>
        <div className="kv">
          <div>
            <span className="muted">Available credits</span>
            <strong>{loading ? '...' : formatCurrency(balance || 0)}</strong>
          </div>
        </div>
      </div>

      <button className="btn btn-outline" onClick={signOut}>
        Sign out
      </button>
    </section>
  )
}
