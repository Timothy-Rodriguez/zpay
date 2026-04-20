import { createContext, useContext, useEffect, useState } from 'react'

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

  // Passwordless sign-in: simulates sending a magic link and immediately
  // authenticates the user. In production replace with real magic-link flow.
  const signIn = async (email) => {
    if (!email || !/^\S+@\S+\.\S+$/.test(email)) {
      throw new Error('Please enter a valid email address')
    }
    await new Promise((r) => setTimeout(r, 600))
    const name = email.split('@')[0].replace(/[._-]+/g, ' ')
    setUser({
      email,
      name: name.charAt(0).toUpperCase() + name.slice(1),
      joinedAt: new Date().toISOString(),
    })
    return true
  }

  const signOut = () => setUser(null)

  return (
    <AuthContext.Provider value={{ user, signIn, signOut }}>
      {children}
    </AuthContext.Provider>
  )
}

export function useAuth() {
  const ctx = useContext(AuthContext)
  if (!ctx) throw new Error('useAuth must be used inside AuthProvider')
  return ctx
}
