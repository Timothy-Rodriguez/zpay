import { useState } from 'react'
import { useLocation, useNavigate } from 'react-router-dom'
import { useAuth } from '../context/AuthContext.jsx'

export default function SignIn() {
  const { signIn } = useAuth()
  const navigate = useNavigate()
  const location = useLocation()
  const redirectTo = location.state?.from?.pathname || '/dashboard'

  const [email, setEmail] = useState('')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const [sent, setSent] = useState(false)

  const submit = async (e) => {
    e.preventDefault()
    setError('')
    setLoading(true)
    try {
      setSent(true)
      await signIn(email)
      navigate(redirectTo, { replace: true })
    } catch (err) {
      setError(err.message)
      setSent(false)
    } finally {
      setLoading(false)
    }
  }

  return (
    <section className="auth-wrap">
      <div className="auth-card">
        <h1>Welcome to ZPay</h1>
        <p className="muted">
          Enter your email — we'll sign you in with a secure magic link. No
          password required.
        </p>
        <form onSubmit={submit} className="form">
          <label htmlFor="email">Email address</label>
          <input
            id="email"
            type="email"
            autoFocus
            required
            placeholder="you@company.com"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
          />
          {error && <div className="alert error">{error}</div>}
          {sent && !error && (
            <div className="alert success">
              Magic link sent! Signing you in…
            </div>
          )}
          <button
            type="submit"
            className="btn btn-primary btn-lg"
            disabled={loading}
          >
            {loading ? 'Sending link…' : 'Continue with email'}
          </button>
        </form>
        <p className="muted small">
          By continuing you agree to ZPay's Terms and Privacy Policy.
        </p>
      </div>
    </section>
  )
}
