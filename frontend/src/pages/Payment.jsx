import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { api } from '../utils/api.js'
import { formatCurrency } from '../utils/format.js'

export default function Payment() {
  const navigate = useNavigate()
  const [balance, setBalance] = useState(null)
  const [toEmail, setToEmail] = useState('')
  const [amount, setAmount] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(true)

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

  const submit = (e) => {
    e.preventDefault()
    setError('')

    if (!/^\S+@\S+\.\S+$/.test(toEmail)) {
      setError('Please enter a valid recipient email')
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

  return (
    <section className="page narrow">
      <h1>Send payment</h1>
      <p className="muted">
        Available: <strong>{loading ? '...' : formatCurrency(balance || 0)}</strong>
      </p>

      <form onSubmit={submit} className="form panel">
        <label htmlFor="toEmail">Recipient email</label>
        <input
          id="toEmail"
          type="email"
          required
          placeholder="recipient@example.com"
          value={toEmail}
          onChange={(e) => setToEmail(e.target.value)}
        />

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

        <button type="submit" className="btn btn-primary btn-lg" disabled={loading}>
          Continue
        </button>
      </form>
    </section>
  )
}
