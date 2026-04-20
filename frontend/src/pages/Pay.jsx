import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useWallet } from '../context/WalletContext.jsx'
import { formatCurrency } from '../utils/format.js'

export default function Pay() {
  const { credits, payees, makePayment, addPayee } = useWallet()
  const navigate = useNavigate()

  const [payeeId, setPayeeId] = useState(payees[0]?.id || '')
  const [amount, setAmount] = useState('')
  const [note, setNote] = useState('')
  const [newPayee, setNewPayee] = useState({ name: '', email: '' })
  const [showNew, setShowNew] = useState(false)
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)

  const submit = async (e) => {
    e.preventDefault()
    setError('')
    setLoading(true)
    try {
      let target = payees.find((p) => p.id === payeeId)
      if (showNew) {
        if (!newPayee.name || !newPayee.email)
          throw new Error('Enter payee name and email')
        target = addPayee(newPayee)
      }
      if (!target) throw new Error('Select a payee')
      await new Promise((r) => setTimeout(r, 600))
      makePayment({ amount, to: target.name, note })
      navigate('/transactions')
    } catch (err) {
      setError(err.message)
    } finally {
      setLoading(false)
    }
  }

  return (
    <section className="page narrow">
      <h1>Send payment</h1>
      <p className="muted">
        Available: <strong>{formatCurrency(credits)}</strong>
      </p>

      <form onSubmit={submit} className="form panel">
        <div className="row-split">
          <label>Payee</label>
          <button
            type="button"
            className="link"
            onClick={() => setShowNew((v) => !v)}
          >
            {showNew ? 'Choose existing' : '+ New payee'}
          </button>
        </div>

        {showNew ? (
          <div className="grid-2">
            <input
              placeholder="Full name"
              value={newPayee.name}
              onChange={(e) =>
                setNewPayee({ ...newPayee, name: e.target.value })
              }
            />
            <input
              type="email"
              placeholder="Email"
              value={newPayee.email}
              onChange={(e) =>
                setNewPayee({ ...newPayee, email: e.target.value })
              }
            />
          </div>
        ) : (
          <select
            value={payeeId}
            onChange={(e) => setPayeeId(e.target.value)}
          >
            {payees.map((p) => (
              <option key={p.id} value={p.id}>
                {p.name} — {p.email}
              </option>
            ))}
          </select>
        )}

        <label htmlFor="amount">Amount</label>
        <input
          id="amount"
          type="number"
          min="1"
          placeholder="0.00"
          value={amount}
          onChange={(e) => setAmount(e.target.value)}
          required
        />

        <label htmlFor="note">Note (optional)</label>
        <input
          id="note"
          type="text"
          placeholder="What's this for?"
          value={note}
          onChange={(e) => setNote(e.target.value)}
        />

        {error && <div className="alert error">{error}</div>}

        <button className="btn btn-primary btn-lg" disabled={loading}>
          {loading
            ? 'Sending…'
            : `Pay ${amount ? formatCurrency(Number(amount)) : ''}`}
        </button>
      </form>
    </section>
  )
}
