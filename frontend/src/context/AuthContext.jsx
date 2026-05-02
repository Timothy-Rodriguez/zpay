import { createContext, useContext, useEffect, useState } from 'react'
import { api } from '../utils/api.js'

const AuthContext = createContext(null)

export function AuthProvider({ children }) {
  const [user, setUser] = useState(() => {
    try {
      const raw = localStorage.getItem('zpay_user')
      return raw ? JSON.parse(raw) : null
    } catch {
      return null
    }
  })

  useEffect(() => {
    if (user) localStorage.setItem('zpay_user', JSON.stringify(user))
    else localStorage.removeItem('zpay_user')
  }, [user])

  const buildUser = (email) => {
    const name = email.split('@')[0].replace(/[._-]+/g, ' ')
    return {
      email,
      name: name.charAt(0).toUpperCase() + name.slice(1),
      joinedAt: new Date().toISOString(),
    }
  }

  // Sign up via backend /signup. Does not auto-login.
  const signUp = async (email, password) => {
    if (!email || !/^\S+@\S+\.\S+$/.test(email)) {
      throw new Error('Please enter a valid email address')
    }
    await api.signup(email, password)
    return true
  }

  // Login via backend /login. Backend sets HttpOnly cookies; we mirror the
  // user identity locally so the React app knows who is signed in.
  const login = async (email, password) => {
    if (!email || !/^\S+@\S+\.\S+$/.test(email)) {
      throw new Error('Please enter a valid email address')
    }
    await api.login(email, password)
    setUser(buildUser(email))
    return true
  }

  // Legacy passwordless sign-in kept so the existing /signin page works.
  const signIn = async (email) => {
    if (!email || !/^\S+@\S+\.\S+$/.test(email)) {
      throw new Error('Please enter a valid email address')
    }
    await new Promise((r) => setTimeout(r, 600))
    setUser(buildUser(email))
    return true
  }

  const signOut = () => {
    api.logout()
    setUser(null)
  }

  return (
    <AuthContext.Provider value={{ user, signUp, login, signIn, signOut }}>
      {children}
    </AuthContext.Provider>
  )
}

export function useAuth() {
  const ctx = useContext(AuthContext)
  if (!ctx) throw new Error('useAuth must be used inside AuthProvider')
  return ctx
}
