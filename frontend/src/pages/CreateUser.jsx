import { useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { useAuth } from '../context/AuthContext.jsx'
import api from '../utils/api.js'

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

  // email existence check
  const [emailStatus, setEmailStatus] = useState(null) // null | 'checking' | 'exists' | 'available'

  // only show password-match feedback once confirm has been touched
  const [confirmTouched, setConfirmTouched] = useState(false)

  const passwordIssues = password ? validatePassword(password) : []
  const passwordsMatch = password === confirm

  const handleEmailBlur = async () => {
    if (!email || !/^\S+@\S+\.\S+$/.test(email)) return
    setEmailStatus('checking')
    try {
      const data = await api.checkUserExists(email)
      setEmailStatus(data.exists ? 'exists' : 'available')
    } catch {
      setEmailStatus(null)
    }
  }

  const submit = async (e) => {
    e.preventDefault()
    setError('')

    if (!/^\S+@\S+\.\S+$/.test(email)) {
      setError('Please enter a valid email address')
      return
    }
    if (emailStatus === 'exists') {
      setError('This email is already registered')
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

  const emailBorderColor =
    emailStatus === 'exists'
      ? 'var(--color-error, #e53e3e)'
      : emailStatus === 'available'
        ? 'var(--color-success, #38a169)'
        : undefined

  return (
    <section className="auth-wrap">
      <div className="auth-card">
        <h1>Create your ZPay account</h1>
        <p className="muted">
          Simulated payment receipts will be sent to this email for demonstration purposes.
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
            onChange={(e) => { setEmail(e.target.value); setEmailStatus(null) }}
            onBlur={handleEmailBlur}
            style={emailBorderColor ? { borderColor: emailBorderColor, outlineColor: emailBorderColor } : undefined}
          />
          {emailStatus === 'exists' && (
            <p style={{ color: 'var(--color-error, #e53e3e)', margin: '0.25rem 0 0', fontSize: '0.85rem' }}>
              User Exists! <Link to="/login-user">Login?</Link>
            </p>
          )}

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
            onChange={(e) => { setConfirm(e.target.value); setConfirmTouched(true) }}
            style={
              confirmTouched && confirm
                ? { borderColor: passwordsMatch ? 'var(--color-success, #38a169)' : 'var(--color-error, #e53e3e)',
                    outlineColor: passwordsMatch ? 'var(--color-success, #38a169)' : 'var(--color-error, #e53e3e)' }
                : undefined
            }
          />
          {confirmTouched && confirm && !passwordsMatch && (
            <p style={{ color: 'var(--color-error, #e53e3e)', margin: '0.25rem 0 0', fontSize: '0.85rem' }}>
              Passwords do not match
            </p>
          )}
          {confirmTouched && confirm && passwordsMatch && (
            <p style={{ color: 'var(--color-success, #38a169)', margin: '0.25rem 0 0', fontSize: '0.85rem' }}>
              Passwords match ✓
            </p>
          )}

          {error && <div className="alert error">{error}</div>}

          <button
            type="submit"
            className="btn btn-primary btn-lg"
            disabled={loading || passwordIssues.length > 0 || !email || !confirm || emailStatus === 'exists'}
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
