import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { api } from '../utils/api.js'
import { formatCurrency } from '../utils/format.js'
import { useAuth } from '../context/AuthContext.jsx'

// ── Inline styles for the modal ──────────────────────────────────────────────
const overlayStyle = {
  position: 'fixed', inset: 0,
  background: 'rgba(0,0,0,0.45)',
  display: 'flex', alignItems: 'center', justifyContent: 'center',
  zIndex: 1000,
}
const modalStyle = {
  background: '#1e2433',
  color: '#e2e8f0',
  borderRadius: '0.75rem',
  padding: '1.5rem',
  width: 'min(480px, 92vw)',
  maxHeight: '80vh',
  overflowY: 'auto',
  boxShadow: '0 20px 60px rgba(0,0,0,0.5)',
}

function AccountsModal({ accounts, onClose, onUse, currentEmail }) {
  const visible = accounts.filter(
    (acc) => acc.email.toLowerCase() !== (currentEmail || '').toLowerCase()
  )
  return (
    <div style={overlayStyle} onClick={onClose}>
      <div style={modalStyle} onClick={(e) => e.stopPropagation()}>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '1rem' }}>
          <h3 style={{ margin: 0, color: '#f7fafc' }}>Available test accounts</h3>
          <button type="button" onClick={onClose}
            style={{ background: 'none', border: 'none', fontSize: '1.25rem', cursor: 'pointer', lineHeight: 1, color: '#a0aec0' }}>
            ×
          </button>
        </div>

        {visible.length === 0 ? (
          <p style={{ color: '#a0aec0', fontSize: '0.875rem' }}>No accounts found.</p>
        ) : (
          <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: '0.875rem' }}>
            <thead>
              <tr style={{ borderBottom: '1px solid #2d3748', textAlign: 'left', color: '#a0aec0' }}>
                <th style={{ padding: '0.4rem 0.5rem', fontWeight: 500 }}>Email</th>
                <th style={{ padding: '0.4rem 0.5rem', fontWeight: 500 }}>Balance</th>
                <th style={{ padding: '0.4rem 0.5rem' }}></th>
              </tr>
            </thead>
            <tbody>
              {visible.map((acc) => (
                <tr key={acc.email} style={{ borderBottom: '1px solid #2d3748' }}>
                  <td style={{ padding: '0.5rem 0.5rem', color: '#e2e8f0' }}>{acc.email}</td>
                  <td style={{ padding: '0.5rem 0.5rem', color: '#e2e8f0' }}>{formatCurrency(acc.balance)}</td>
                  <td style={{ padding: '0.5rem 0.5rem' }}>
                    <button type="button" className="btn btn-sm"
                      style={{ fontSize: '0.8rem', padding: '0.2rem 0.6rem' }}
                      onClick={() => onUse(acc.email)}>
                      Use
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>
    </div>
  )
}

export default function Payment() {
  const navigate = useNavigate()
  const { user } = useAuth()
  const [balance, setBalance] = useState(null)
  const [toEmail, setToEmail] = useState('')
  const [amount, setAmount] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(true)

  // recipient existence check — triggered when focus moves to amount field
  const [recipientStatus, setRecipientStatus] = useState(null) // null | 'checking' | 'found' | 'not-found' | 'invalid'

  // test accounts modal
  const [testAccounts, setTestAccounts] = useState(null) // null | 'loading' | []

  const openTestAccounts = async (e) => {
    e.preventDefault()
    setTestAccounts('loading')
    try {
      const data = await api.getAccounts()
      setTestAccounts(data.accounts || [])
    } catch {
      setTestAccounts([])
    }
  }

  const closeTestAccounts = () => setTestAccounts(null)

  const useAccount = (email) => {
    setToEmail(email)
    setRecipientStatus(null)
    setTestAccounts(null)
  }

  useEffect(() => {
    const fetchBalance = async () => {
      try {
        const data = await api.getBalance()
        if (data && data.balance) {
          setBalance(data.balance)
        }
      } catch (err) {
        console.error('Failed to fetch balance:', err)
      } finally {
        setLoading(false)
      }
    }
    fetchBalance()
  }, [])

  const handleEmailBlur = async () => {
    if (!toEmail) return
    if (!/^\S+@\S+\.\S+$/.test(toEmail)) {
      setRecipientStatus('invalid')
      return
    }
    if (user && toEmail.trim().toLowerCase() === user.email.toLowerCase()) {
      setRecipientStatus('self')
      return
    }
    setRecipientStatus('checking')
    try {
      const data = await api.checkUserExists(toEmail.trim())
      setRecipientStatus(data.exists ? 'found' : 'not-found')
    } catch {
      setRecipientStatus(null)
    }
  }

  const submit = (e) => {
    e.preventDefault()
    setError('')

    if (!/^\S+@\S+\.\S+$/.test(toEmail)) {
      setError('Please enter a valid recipient email')
      return
    }
    if (recipientStatus === 'not-found') {
      setError('Recipient account not found')
      return
    }
    if (user && toEmail.trim().toLowerCase() === user.email.toLowerCase()) {
      setError('You cannot send a payment to yourself')
      return
    }
    const value = Number(amount)
    if (!value || value <= 0) {
      setError('Please enter a valid amount greater than zero')
      return
    }
    if (balance !== null && value > Number(balance)) {
      setError('Insufficient balance')
      return
    }

    navigate('/payment-process', {
      state: { toEmail: toEmail.trim(), amount: value },
    })
  }

  const emailBorder =
    recipientStatus === 'found'
      ? { borderColor: 'var(--color-success, #38a169)', outlineColor: 'var(--color-success, #38a169)' }
      : recipientStatus === 'not-found' || recipientStatus === 'invalid' || recipientStatus === 'self'
        ? { borderColor: 'var(--color-error, #e53e3e)', outlineColor: 'var(--color-error, #e53e3e)' }
        : undefined

  return (
    <section className="page narrow">
      {Array.isArray(testAccounts) && (
        <AccountsModal accounts={testAccounts} onClose={closeTestAccounts} onUse={useAccount} currentEmail={user?.email} />
      )}
      {testAccounts === 'loading' && (
        <div style={overlayStyle}>
          <p style={{ color: '#fff' }}>Loading…</p>
        </div>
      )}

      <h1>Make a payment</h1>
      <p className="muted" style={{ marginBottom: '0.25rem' }}>
        Available: <strong>{loading ? '...' : formatCurrency(balance || 0)}</strong>
      </p>
      <p className="muted small" style={{ marginBottom: '1rem' }}>
        Here for testing?{' '}
        <a href="#" onClick={openTestAccounts} style={{ textDecoration: 'underline', cursor: 'pointer' }}>
          Simulate a payment to available accounts in ZPay!
        </a>
      </p>

      <form onSubmit={submit} className="form panel">
        <label htmlFor="toEmail">Recipient email</label>
        <input
          id="toEmail"
          type="email"
          required
          placeholder="recipient@example.com"
          value={toEmail}
          onChange={(e) => { setToEmail(e.target.value); setRecipientStatus(null) }}
          onBlur={handleEmailBlur}
          style={emailBorder}
        />
        {recipientStatus === 'invalid' && (
          <p style={{ color: 'var(--color-error, #e53e3e)', margin: '0.25rem 0 0', fontSize: '0.85rem' }}>
            Please enter a valid email address
          </p>
        )}
        {recipientStatus === 'self' && (
          <p style={{ color: 'var(--color-error, #e53e3e)', margin: '0.25rem 0 0', fontSize: '0.85rem' }}>
            You cannot send a payment to yourself
          </p>
        )}
        {recipientStatus === 'not-found' && (
          <p style={{ color: 'var(--color-error, #e53e3e)', margin: '0.25rem 0 0', fontSize: '0.85rem' }}>
            No ZPay account found for this email
          </p>
        )}
        {recipientStatus === 'found' && (
          <p style={{ color: 'var(--color-success, #38a169)', margin: '0.25rem 0 0', fontSize: '0.85rem' }}>
            Recipient account found ✓
          </p>
        )}
        {recipientStatus === 'checking' && (
          <p style={{ color: 'var(--color-muted, #718096)', margin: '0.25rem 0 0', fontSize: '0.85rem' }}>
            Checking…
          </p>
        )}

        <label htmlFor="amount">Amount</label>
        <input
          id="amount"
          type="number"
          min="1"
          step="0.01"
          required
          placeholder="0.00"
          value={amount}
          onChange={(e) => setAmount(e.target.value)}
        />

        {error && <div className="alert error">{error}</div>}

        <button
          type="submit"
          className="btn btn-primary btn-lg"
          disabled={loading || recipientStatus === 'not-found' || recipientStatus === 'invalid' || recipientStatus === 'self' || recipientStatus === 'checking'}
        >
          Continue
        </button>

        <div style={{ marginTop: '1.25rem', padding: '0.875rem 1rem', background: 'rgba(99,179,237,0.07)', border: '1px solid rgba(99,179,237,0.2)', borderRadius: '0.5rem', fontSize: '0.85rem' }}>
          <p style={{ margin: '0 0 0.6rem', color: '#a0aec0' }}>
            Want to receive payments from a test account? Use these credentials to access Test account:
          </p>
          {[
            { label: 'Email ID', value: 'test@gmail.com' },
            { label: 'Password', value: 'Test1234' },
          ].map(({ label, value }) => (
            <div key={label} style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: '0.35rem' }}>
              <span style={{ color: '#cbd5e0' }}>
                <span style={{ color: '#a0aec0' }}>{label}:&nbsp;</span>{value}
              </span>
              <button
                type="button"
                title={`Copy ${label}`}
                onClick={() => navigator.clipboard.writeText(value)}
                style={{ background: 'none', border: 'none', cursor: 'pointer', padding: '0.15rem 0.3rem', color: '#63b3ed', lineHeight: 1 }}
              >
                <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                  <rect x="9" y="9" width="13" height="13" rx="2" ry="2"/>
                  <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/>
                </svg>
              </button>
            </div>
          ))}
        </div>
      </form>
    </section>
  )
}
