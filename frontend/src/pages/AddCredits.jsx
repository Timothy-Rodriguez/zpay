import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useWallet } from '../context/WalletContext.jsx'
import { formatCurrency } from '../utils/format.js'

const PRESETS = [100, 500, 1000, 5000]
const METHODS = [
  { id: 'card', label: 'Credit / Debit Card' },
  { id: 'upi', label: 'UPI' },
  { id: 'netbanking', label: 'Net Banking' },
]

export default function AddCredits() {
  const { credits, addCredits } = useWallet()
  const navigate = useNavigate()
  const [amount, setAmount] = useState(500)
  const [method, setMethod] = useState('card')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')

  const submit = async (e) => {
    e.preventDefault()
    setError('')
    setLoading(true)
    try {
      await new Promise((r) => setTimeout(r, 700))
      const m = METHODS.find((x) => x.id === method)?.label || 'Card'
      addCredits(amount, m)
      navigate('/dashboard')
    } catch (err) {
      setError(err.message)
    } finally {
      setLoading(false)
    }
  }

  return (
    <section className="page narrow">
      <h1>Add credits</h1>
      <p className="muted">
        Current balance: <strong>{formatCurrency(credits)}</strong>
      </p>

      <form onSubmit={submit} className="form panel">
        <label>Choose amount</label>
        <div className="preset-row">
          {PRESETS.map((p) => (
            <button
              type="button"
              key={p}
              className={`chip-btn ${amount === p ? 'active' : ''}`}
              onClick={() => setAmount(p)}
            >
              {formatCurrency(p)}
            </button>
          ))}
        </div>

        <label htmlFor="amt">Custom amount</label>
        <input
          id="amt"
          type="number"
          min="1"
          value={amount}
          onChange={(e) => setAmount(Number(e.target.value))}
        />

        <label>Payment method</label>
        <div className="method-list">
          {METHODS.map((m) => (
            <label key={m.id} className={`method ${method === m.id ? 'active' : ''}`}>
              <input
                type="radio"
                name="method"
                value={m.id}
                checked={method === m.id}
                onChange={() => setMethod(m.id)}
              />
              <span>{m.label}</span>
            </label>
          ))}
        </div>

        {error && <div className="alert error">{error}</div>}

        <button className="btn btn-primary btn-lg" disabled={loading}>
          {loading ? 'Processing…' : `Add ${formatCurrency(amount)}`}
        </button>
      </form>
    </section>
  )
}
