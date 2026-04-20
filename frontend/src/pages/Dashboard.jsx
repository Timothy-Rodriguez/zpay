import { Link } from 'react-router-dom'
import { useAuth } from '../context/AuthContext.jsx'
import { useWallet } from '../context/WalletContext.jsx'
import { formatCurrency, formatDate } from '../utils/format.js'

export default function Dashboard() {
  const { user } = useAuth()
  const { credits, transactions } = useWallet()

  const recent = transactions.slice(0, 5)
  const totalIn = transactions
    .filter((t) => t.type === 'credit')
    .reduce((a, b) => a + b.amount, 0)
  const totalOut = transactions
    .filter((t) => t.type === 'debit')
    .reduce((a, b) => a + b.amount, 0)

  return (
    <section className="page">
      <div className="page-head">
        <div>
          <h1>Hi, {user?.name} 👋</h1>
          <p className="muted">Here's what's happening with your wallet.</p>
        </div>
      </div>

      <div className="stats">
        <div className="stat-card primary">
          <span>Available credits</span>
          <h2>{formatCurrency(credits)}</h2>
          <div className="stat-actions">
            <Link to="/add-credits" className="btn btn-light">
              + Add credits
            </Link>
            <Link to="/pay" className="btn btn-outline">
              Send payment
            </Link>
          </div>
        </div>
        <div className="stat-card">
          <span>Total added</span>
          <h2>{formatCurrency(totalIn)}</h2>
        </div>
        <div className="stat-card">
          <span>Total spent</span>
          <h2>{formatCurrency(totalOut)}</h2>
        </div>
      </div>

      <div className="panel">
        <div className="panel-head">
          <h3>Recent activity</h3>
          <Link to="/transactions" className="link">
            View all →
          </Link>
        </div>
        {recent.length === 0 ? (
          <p className="muted">No transactions yet.</p>
        ) : (
          <ul className="tx-list">
            {recent.map((t) => (
              <li key={t.id} className="tx-item">
                <div className={`tx-icon ${t.type}`}>
                  {t.type === 'credit' ? '↓' : '↑'}
                </div>
                <div className="tx-main">
                  <strong>{t.description}</strong>
                  <span className="muted small">{formatDate(t.date)}</span>
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
