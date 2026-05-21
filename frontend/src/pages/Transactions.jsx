import { useEffect, useMemo, useState } from 'react'
import { api } from '../utils/api.js'
import { formatCurrency, formatDate } from '../utils/format.js'

export default function Transactions() {
  const [transactions, setTransactions] = useState([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [filter, setFilter] = useState('all')

  useEffect(() => {
    const fetchTransactions = async () => {
      try {
        setLoading(true)
        setError('')
        const data = await api.getTransactions()
        if (data && Array.isArray(data.transactions)) {
          setTransactions(data.transactions)
        }
      } catch (err) {
        setError(err.message || 'Failed to load transactions')
      } finally {
        setLoading(false)
      }
    }
    fetchTransactions()
  }, [])

  const filtered = useMemo(() => {
    return transactions.filter((t) => filter === 'all' || t.direction === filter)
  }, [transactions, filter])

  return (
    <section className="page">
      <h1>Transactions</h1>

      <div className="toolbar">
        <div className="tabs">
          {['all', 'credit', 'debit'].map((f) => (
            <button
              key={f}
              className={`tab ${filter === f ? 'active' : ''}`}
              onClick={() => setFilter(f)}
            >
              {f === 'all' ? 'All' : f === 'credit' ? 'Incoming' : 'Outgoing'}
            </button>
          ))}
        </div>
      </div>

      <div className="panel">
        {loading && <p className="muted">Loading transactions…</p>}
        {error && <div className="alert error">{error}</div>}
        {!loading && filtered.length === 0 ? (
          <p className="muted">No transactions found.</p>
        ) : !loading ? (
          <ul className="tx-list">
            {filtered.map((t) => (
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
