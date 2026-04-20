import { useMemo, useState } from 'react'
import { useWallet } from '../context/WalletContext.jsx'
import { formatCurrency, formatDate } from '../utils/format.js'

export default function Transactions() {
  const { transactions } = useWallet()
  const [filter, setFilter] = useState('all')
  const [q, setQ] = useState('')

  const filtered = useMemo(() => {
    return transactions.filter((t) => {
      if (filter !== 'all' && t.type !== filter) return false
      if (q && !t.description.toLowerCase().includes(q.toLowerCase()))
        return false
      return true
    })
  }, [transactions, filter, q])

  return (
    <section className="page">
      <h1>Transactions</h1>

      <div className="toolbar">
        <input
          type="search"
          placeholder="Search transactions…"
          value={q}
          onChange={(e) => setQ(e.target.value)}
        />
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
        {filtered.length === 0 ? (
          <p className="muted">No transactions found.</p>
        ) : (
          <ul className="tx-list">
            {filtered.map((t) => (
              <li key={t.id} className="tx-item">
                <div className={`tx-icon ${t.type}`}>
                  {t.type === 'credit' ? '↓' : '↑'}
                </div>
                <div className="tx-main">
                  <strong>{t.description}</strong>
                  <span className="muted small">
                    {formatDate(t.date)} · {t.status}
                  </span>
                </div>
                <div className={`tx-amount ${t.type}`}>
                  {t.type === 'credit' ? '+' : '-'}
                  {formatCurrency(t.amount)}
                </div>
              </li>
            ))}
          </ul>
        )}
      </div>
    </section>
  )
}
