import { useState } from 'react'
import { Navigate, useLocation, useNavigate } from 'react-router-dom'
import { useAuth } from '../context/AuthContext.jsx'
import { api, newIdempotencyKey } from '../utils/api.js'
import { formatCurrency } from '../utils/format.js'

export default function PaymentProcess() {
  const { user, login } = useAuth()
  const location = useLocation()
  const navigate = useNavigate()

  const payload = location.state
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)
  const [success, setSuccess] = useState(null)

  // Guard against direct navigation without payload
  if (!payload || !payload.toEmail || !payload.amount) {
    return <Navigate to="/payment" replace />
  }

  const { toEmail, amount } = payload

  const submit = async (e) => {
    e.preventDefault()
    setError('')

    if (!password) {
      setError('Please enter your password to authorize the payment')
      return
    }

    setLoading(true)
    try {
      // Re-authenticate to confirm the user is who they claim to be.
      // This also refreshes auth cookies if the previous session expired.
      if (user?.email) {
        await login(user.email, password)
      }

      const result = await api.pay({
        toEmail,
        amount,
        idempotencyKey: newIdempotencyKey(),
      })
      setSuccess(result)
    } catch (err) {
      setError(err.message || 'Payment failed')
    } finally {
      setLoading(false)
    }
  }

  if (success) {
    return (
      <section className="page narrow">
        <h1>Payment successful</h1>
        <div className="panel">
          <p className="muted">Your payment has been processed.</p>
          <ul className="form">
            <li><strong>To:</strong> {success.to || toEmail}</li>
            <li><strong>Amount:</strong> {formatCurrency(success.amount || amount)}</li>
            {success.timestamp && <li><strong>When:</strong> {success.timestamp}</li>}
          </ul>
          <button
            className="btn btn-primary"
            onClick={() => navigate('/dashboard')}
          >
            Back to dashboard
          </button>
        </div>
      </section>
    )
  }

  return (
    <section className="page narrow">
      <h1>Confirm payment</h1>
      <p className="muted">Review the details below and enter your password to authorize.</p>

      <div className="panel" style={{ marginBottom: '1rem' }}>
        <ul className="form" style={{ margin: 0 }}>
          {user?.email && (
            <li><strong>From:</strong> {user.email}</li>
          )}
          <li><strong>To:</strong> {toEmail}</li>
          <li><strong>Amount:</strong> {formatCurrency(amount)}</li>
        </ul>
      </div>

      <form onSubmit={submit} className="form panel">
        <label htmlFor="password">Password</label>
        <input
          id="password"
          type="password"
          autoFocus
          required
          placeholder="Enter your password to confirm"
          value={password}
          onChange={(e) => setPassword(e.target.value)}
        />

        {error && <div className="alert error">{error}</div>}

        <div style={{ display: 'flex', gap: '0.5rem' }}>
          <button
            type="button"
            className="btn btn-ghost"
            onClick={() => navigate('/payment')}
            disabled={loading}
          >
            Back
          </button>
          <button
            type="submit"
            className="btn btn-primary btn-lg"
            disabled={loading}
          >
            {loading ? 'Processing…' : `Pay ${formatCurrency(amount)}`}
          </button>
        </div>
      </form>
    </section>
  )
}
