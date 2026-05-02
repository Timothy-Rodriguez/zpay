import { useState } from 'react'
import { Link, useLocation, useNavigate } from 'react-router-dom'
import { useAuth } from '../context/AuthContext.jsx'

export default function LoginUser() {
  const { login } = useAuth()
  const navigate = useNavigate()
  const location = useLocation()
  const redirectTo = location.state?.from?.pathname || '/dashboard'

  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)

  const submit = async (e) => {
    e.preventDefault()
    setError('')

    if (!/^\S+@\S+\.\S+$/.test(email)) {
      setError('Please enter a valid email address')
      return
    }
    if (!password) {
      setError('Please enter your password')
      return
    }

    setLoading(true)
    try {
      await login(email, password)
      navigate(redirectTo, { replace: true })
    } catch (err) {
      setError(err.message || 'Login failed')
    } finally {
      setLoading(false)
    }
  }

  return (
    <section className="auth-wrap">
      <div className="auth-card">
        <h1>Log in to ZPay</h1>
        <p className="muted">Use your email and password to access your account.</p>
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
            placeholder="Your password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
          />

          {error && <div className="alert error">{error}</div>}

          <button
            type="submit"
            className="btn btn-primary btn-lg"
            disabled={loading}
          >
            {loading ? 'Logging in…' : 'Log in'}
          </button>
        </form>
        <p className="muted small">
          New to ZPay? <Link to="/create-user">Create an account</Link>
        </p>
      </div>
    </section>
  )
}
