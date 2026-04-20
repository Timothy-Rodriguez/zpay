import { createContext, useContext, useEffect, useState } from 'react'
import { useAuth } from './AuthContext.jsx'

const WalletContext = createContext(null)

const DEFAULT_CREDITS = 1000

const defaultState = {
  credits: DEFAULT_CREDITS,
  transactions: [
    {
      id: 'tx-welcome',
      type: 'credit',
      amount: DEFAULT_CREDITS,
      description: 'Welcome bonus',
      date: new Date().toISOString(),
      status: 'completed',
    },
  ],
  payees: [
    { id: 'p1', name: 'Aarav Shah', email: 'aarav@example.com' },
    { id: 'p2', name: 'Netflix', email: 'billing@netflix.com' },
    { id: 'p3', name: 'Spotify', email: 'pay@spotify.com' },
  ],
}

export function WalletProvider({ children }) {
  const { user } = useAuth()
  const storageKey = user ? `zpay_wallet_${user.email}` : null

  const [state, setState] = useState(defaultState)

  // Load per-user wallet
  useEffect(() => {
    if (!storageKey) {
      setState(defaultState)
      return
    }
    try {
      const raw = localStorage.getItem(storageKey)
      setState(raw ? JSON.parse(raw) : defaultState)
    } catch {
      setState(defaultState)
    }
  }, [storageKey])

  // Persist
  useEffect(() => {
    if (storageKey) localStorage.setItem(storageKey, JSON.stringify(state))
  }, [state, storageKey])

  const addCredits = (amount, method = 'Card') => {
    const value = Number(amount)
    if (!value || value <= 0) throw new Error('Enter a valid amount')
    const tx = {
      id: `tx-${Date.now()}`,
      type: 'credit',
      amount: value,
      description: `Top-up via ${method}`,
      date: new Date().toISOString(),
      status: 'completed',
    }
    setState((s) => ({
      ...s,
      credits: s.credits + value,
      transactions: [tx, ...s.transactions],
    }))
    return tx
  }

  const makePayment = ({ amount, to, note }) => {
    const value = Number(amount)
    if (!value || value <= 0) throw new Error('Enter a valid amount')
    if (!to) throw new Error('Select a payee')
    if (value > state.credits) throw new Error('Insufficient credits')
    const tx = {
      id: `tx-${Date.now()}`,
      type: 'debit',
      amount: value,
      description: note ? `Payment to ${to} — ${note}` : `Payment to ${to}`,
      date: new Date().toISOString(),
      status: 'completed',
    }
    setState((s) => ({
      ...s,
      credits: s.credits - value,
      transactions: [tx, ...s.transactions],
    }))
    return tx
  }

  const addPayee = (payee) => {
    const p = { id: `p-${Date.now()}`, ...payee }
    setState((s) => ({ ...s, payees: [p, ...s.payees] }))
    return p
  }

  return (
    <WalletContext.Provider
      value={{ ...state, addCredits, makePayment, addPayee }}
    >
      {children}
    </WalletContext.Provider>
  )
}

export function useWallet() {
  const ctx = useContext(WalletContext)
  if (!ctx) throw new Error('useWallet must be used inside WalletProvider')
  return ctx
}
