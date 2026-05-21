import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { useAuth } from '../context/AuthContext.jsx'
import { api } from '../utils/api.js'
import { formatCurrency, formatDate } from '../utils/format.js'

// ── SVG bar chart ──────────────────────────────────────────────────────────────
function ActivityChart({ data }) {
  const maxVal = Math.max(...data.flatMap((d) => [Number(d.credit), Number(d.debit)]), 1)
  const H = 110, barW = 16, gap = 6
  const colW = barW * 2 + gap + 16
  const W    = data.length * colW + 20
  return (
    <svg width="100%" viewBox={`0 0 ${W} ${H + 28}`} preserveAspectRatio="none" style={{ display: 'block' }}>
      <line x1={0} y1={H} x2={W} y2={H} stroke="#334155" strokeWidth="1" />
      {data.map((d, i) => {
        const x  = 10 + i * colW
        const cH = Math.round((Number(d.credit) / maxVal) * H)
        const dH = Math.round((Number(d.debit)  / maxVal) * H)
        return (
          <g key={`${d.date}-${i}`}>
            <rect x={x}            y={H - cH} width={barW} height={cH || 1} rx="3" fill="#10b981" opacity="0.85" />
            <rect x={x + barW + gap} y={H - dH} width={barW} height={dH || 1} rx="3" fill="#ef4444" opacity="0.8"  />
            <text x={x + barW} y={H + 18} textAnchor="middle" fill="#94a3b8" fontSize="10">{d.day}</text>
          </g>
        )
      })}
    </svg>
  )
}

// ── SVG donut chart ────────────────────────────────────────────────────────────
function DonutChart({ credit, debit }) {
  const c = Number(credit), d = Number(debit)
  const total     = c + d || 1
  const r         = 52
  const cx = 68, cy = 68
  const circ      = 2 * Math.PI * r
  const creditArc = (c / total) * circ
  const debitArc  = (d / total) * circ
  const creditPct = Math.round((c / total) * 100)
  return (
    <svg width="136" height="136" viewBox="0 0 136 136" style={{ display: 'block', margin: '0 auto' }}>
      <circle cx={cx} cy={cy} r={r} fill="none" stroke="#1e293b" strokeWidth="20" />
      {c > 0 && (
        <circle cx={cx} cy={cy} r={r} fill="none" stroke="#10b981" strokeWidth="20" opacity="0.9"
          strokeDasharray={`${creditArc} ${circ}`} strokeDashoffset="0" strokeLinecap="round"
          transform={`rotate(-90 ${cx} ${cy})`} />
      )}
      {d > 0 && (
        <circle cx={cx} cy={cy} r={r} fill="none" stroke="#ef4444" strokeWidth="20" opacity="0.85"
          strokeDasharray={`${debitArc} ${circ}`} strokeDashoffset={`-${creditArc}`} strokeLinecap="round"
          transform={`rotate(-90 ${cx} ${cy})`} />
      )}
      <text x={cx} y={cy - 5}  textAnchor="middle" fill="#e2e8f0" fontSize="15" fontWeight="700">{creditPct}%</text>
      <text x={cx} y={cy + 13} textAnchor="middle" fill="#94a3b8" fontSize="10">credits</text>
    </svg>
  )
}
// ──────────────────────────────────────────────────────────────────────────────

export default function Dashboard() {
  const { user } = useAuth()
  const [dash, setDash]     = useState(null)
  const [loading, setLoading] = useState(true)
  const [error, setError]   = useState('')

  useEffect(() => {
    api.getDashboard()
      .then((data) => setDash(data))
      .catch((err) => setError(err.message || 'Failed to load dashboard'))
      .finally(() => setLoading(false))
  }, [])

  const balance    = dash?.balance          ?? '0'
  const credited   = dash?.total_credited   ?? '0'
  const debited    = dash?.total_debited    ?? '0'
  const netFlow    = dash?.net_flow         ?? '0'
  const txCount    = dash?.transaction_count ?? 0
  const activity   = dash?.activity         ?? []
  const recent     = dash?.recent           ?? []

  return (
    <section className="page">
      <div className="page-head">
        <div>
          <h1>Hi, {user?.name} 👋</h1>
          <p className="muted">Here's your ledger overview.</p>
        </div>
        <Link to="/payment" className="btn btn-primary">+ Send payment</Link>
      </div>

      {error && <div className="alert error">{error}</div>}

      {/* ── Balance stats ─────────────────────────────────────────── */}
      <div className="stats">
        <div className="stat-card primary">
          <span>Available balance</span>
          <h2>{loading ? '…' : formatCurrency(balance)}</h2>
          <div className="stat-actions">
            <Link to="/payment" className="btn btn-outline">Send</Link>
          </div>
        </div>
        <div className="stat-card">
          <span>Total credited</span>
          <h2 className="amount-credit">{loading ? '…' : formatCurrency(credited)}</h2>
        </div>
        <div className="stat-card">
          <span>Total debited</span>
          <h2 className="amount-debit">{loading ? '…' : formatCurrency(debited)}</h2>
        </div>
      </div>

      {/* ── Charts row ────────────────────────────────────────────── */}
      <div className="dash-row">

        {/* Activity bar chart */}
        <div className="panel">
          <div className="panel-head">
            <h3>7-Day Activity</h3>
            <div style={{ display: 'flex', gap: '12px', alignItems: 'center' }}>
              <span className="chart-legend"><span className="dot dot-credit" />Credit</span>
              <span className="chart-legend"><span className="dot dot-debit"  />Debit</span>
            </div>
          </div>
          <div style={{ marginTop: '8px' }}>
            {loading ? <p className="muted">Loading…</p> : <ActivityChart data={activity} />}
          </div>
        </div>

        {/* Donut + split */}
        <div className="panel">
          <div className="panel-head">
            <h3>Credit / Debit Split</h3>
          </div>
          <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', gap: '16px', paddingTop: '8px' }}>
            {loading ? <p className="muted">Loading…</p> : <DonutChart credit={credited} debit={debited} />}
            <div className="donut-legend">
              <div className="donut-legend-item">
                <span className="dot dot-credit" />
                <span className="muted small">Credited</span>
                <strong>{formatCurrency(credited)}</strong>
              </div>
              <div className="donut-legend-item">
                <span className="dot dot-debit" />
                <span className="muted small">Debited</span>
                <strong>{formatCurrency(debited)}</strong>
              </div>
            </div>
          </div>
        </div>

      </div>

      {/* ── Ledger summary + Recent transactions ──────────────────── */}
      <div className="dash-row" style={{ gridTemplateColumns: '1fr 1.8fr' }}>

        {/* Ledger summary */}
        <div className="panel">
          <h3 style={{ marginBottom: '16px' }}>Ledger Summary</h3>
          <div className="ledger-summary-list">
            <div className="ledger-row">
              <span className="muted">Total credited</span>
              <span className="amount-credit">+{formatCurrency(credited)}</span>
            </div>
            <div className="ledger-row">
              <span className="muted">Total debited</span>
              <span className="amount-debit">−{formatCurrency(debited)}</span>
            </div>
            <div className="ledger-divider" />
            <div className="ledger-row ledger-row-total">
              <span>Net flow</span>
              <strong className={Number(netFlow) >= 0 ? 'amount-credit' : 'amount-debit'}>
                {Number(netFlow) >= 0 ? '+' : '−'}{formatCurrency(Math.abs(Number(netFlow)))}
              </strong>
            </div>
            <div className="ledger-row">
              <span className="muted">Transactions</span>
              <span>{txCount}</span>
            </div>
          </div>
        </div>

        {/* Recent activity */}
        <div className="panel">
          <div className="panel-head">
            <h3>Recent activity</h3>
            <Link to="/transactions" className="link">View all →</Link>
          </div>
          {loading && <p className="muted">Loading…</p>}
          {!loading && recent.length === 0 && (
            <p className="muted small">No transactions yet. <Link to="/pay" className="link">Add credits</Link> to get started.</p>
          )}
          {!loading && recent.length > 0 && (
            <ul className="tx-list">
              {recent.map((t) => (
                <li key={t.id} className="tx-item">
                  <div className={`tx-icon ${t.direction}`}>
                    {t.direction === 'credit' ? '↓' : '↑'}
                  </div>
                  <div className="tx-main">
                    <strong>
                      {t.direction === 'credit' ? `From ${t.from_email}` : `To ${t.to_email}`}
                    </strong>
                    <span className="muted small">{formatDate(t.created_at)}</span>
                  </div>
                  <div className={`tx-amount ${t.direction}`}>
                    {t.direction === 'credit' ? '+' : '−'}{formatCurrency(t.amount)}
                  </div>
                </li>
              ))}
            </ul>
          )}
        </div>

      </div>
    </section>
  )
}
