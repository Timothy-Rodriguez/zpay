import { useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { useAuth } from '../context/AuthContext.jsx'

const PASSWORD_RULES = [
  { test: (p) => p.length >= 8, label: 'At least 8 characters' },
  { test: (p) => /[A-Z]/.test(p), label: 'One uppercase letter' },
  { test: (p) => /[a-z]/.test(p), label: 'One lowercase letter' },
]

function validatePassword(password) {
  const failed = PASSWORD_RULES.filter((r) => !r.test(password))
  return failed.map((r) => r.label)
}

export default function CreateUser() {
  const { signUp } = useAuth()
  const navigate = useNavigate()

  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [confirm, setConfirm] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)

  const passwordIssues = password ? validatePassword(password) : []

  const submit = async (e) => {
    e.preventDefault()
    setError('')

    if (!/^\S+@\S+\.\S+$/.test(email)) {
      setError('Please enter a valid email address')
      return
    }
    const issues = validatePassword(password)
    if (issues.length) {
      setError(`Password must have: ${issues.join(', ')}`)
      return
    }
    if (password !== confirm) {
      setError('Passwords do not match')
      return
    }

    setLoading(true)
    try {
      await signUp(email, password)
      navigate('/login-user', { replace: true })
    } catch (err) {
      setError(err.message || 'Failed to create account')
    } finally {
      setLoading(false)
    }
  }

  return (
    <section className="auth-wrap">
      <div className="auth-card">
        <h1>Create your ZPay account</h1>
        <p className="muted">
          Sign up with your email and a strong password.
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

          <label htmlFor="password">Password</label>
          <input
            id="password"
            type="password"
            required
            placeholder="Create a password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
          />

          <ul className="muted small" style={{ paddingLeft: '1.1rem', margin: '0.25rem 0 0.5rem' }}>
            {PASSWORD_RULES.map((r) => {
              const ok = r.test(password)
              return (
                <li key={r.label} style={{ color: ok ? 'green' : undefined }}>
                  {ok ? '✓' : '•'} {r.label}
                </li>
              )
            })}
          </ul>

          <label htmlFor="confirm">Re-type password</label>
          <input
            id="confirm"
            type="password"
            required
            placeholder="Confirm password"
            value={confirm}
            onChange={(e) => setConfirm(e.target.value)}
          />

          {error && <div className="alert error">{error}</div>}

          <button
            type="submit"
            className="btn btn-primary btn-lg"
            disabled={loading || passwordIssues.length > 0 || !email || !confirm}
          >
            {loading ? 'Creating account…' : 'Create account'}
          </button>
        </form>
        <p className="muted small">
          Already have an account? <Link to="/login-user">Log in</Link>
        </p>
      </div>
    </section>
  )
}
