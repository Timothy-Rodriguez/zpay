import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { useAuth } from '../context/AuthContext.jsx'
import { api } from '../utils/api.js'
import { formatCurrency, formatDate } from '../utils/format.js'

export default function Profile() {
  const { user } = useAuth()
  const [balance, setBalance] = useState(null)
  const [transactions, setTransactions] = useState([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  useEffect(() => {
    const fetchData = async () => {
      try {
        setLoading(true)
        setError('')
        const [balData, txData] = await Promise.all([
          api.getBalance(),
          api.getTransactions(),
        ])
        if (balData && balData.balance) {
          setBalance(balData.balance)
        }
        if (txData && Array.isArray(txData.transactions)) {
          setTransactions(txData.transactions)
        }
      } catch (err) {
        setError(err.message || 'Failed to load data')
      } finally {
        setLoading(false)
      }
    }
    fetchData()
  }, [])

  const recent = transactions.slice(0, 5)
  const totalIn = transactions
    .filter((t) => t.direction === 'credit')
    .reduce((a, b) => a + Number(b.amount), 0)
  const totalOut = transactions
    .filter((t) => t.direction === 'debit')
    .reduce((a, b) => a + Number(b.amount), 0)

  return (
    <section className="page">
      <div className="page-head">
        <div>
          <h1>Hi, {user?.name} 👋</h1>
          <p className="muted">Here's what's happening with your wallet.</p>
        </div>
      </div>

      {error && <div className="alert error">{error}</div>}

      <div className="stats">
        <div className="stat-card primary">
          <span>Available credits</span>
          <h2>{loading ? '...' : formatCurrency(balance || 0)}</h2>
          <div className="stat-actions">
            <Link to="/payment" className="btn btn-outline">
              Send payment
            </Link>
          </div>
        </div>
        <div className="stat-card">
          <span>Total added</span>
          <h2>{loading ? '...' : formatCurrency(totalIn)}</h2>
        </div>
        <div className="stat-card">
          <span>Total spent</span>
          <h2>{loading ? '...' : formatCurrency(totalOut)}</h2>
        </div>
      </div>

      <div className="panel">
        <div className="panel-head">
          <h3>Recent activity</h3>
          <Link to="/transactions" className="link">
            View all →
          </Link>
        </div>
        {loading && <p className="muted">Loading…</p>}
        {!loading && recent.length === 0 ? (
          <p className="muted">No transactions yet.</p>
        ) : !loading ? (
          <ul className="tx-list">
            {recent.map((t) => (
              <li key={t.id} className="tx-item">
                <div className={`tx-icon ${t.direction}`}>
                  {t.direction === 'credit' ? '↓' : '↑'}
                </div>
                <div className="tx-main">
                  <strong>
                    {t.direction === 'credit'
                      ? `From ${t.from_email}`
                      : `To ${t.to_email}`}
                  </strong>
                  <span className="muted small">{formatDate(t.created_at)}</span>
                </div>
                <div className={`tx-amount ${t.direction}`}>
                  {t.direction === 'credit' ? '+' : '-'}
                  {formatCurrency(t.amount)}
                </div>
              </li>
            ))}
          </ul>
        ) : null}
      </div>
    </section>
  )
}
