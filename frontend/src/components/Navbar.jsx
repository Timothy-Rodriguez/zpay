import { Link, NavLink, useNavigate } from 'react-router-dom'
import { useAuth } from '../context/AuthContext.jsx'

export default function Navbar() {
  const { user, signOut } = useAuth()
  const navigate = useNavigate()

  const handleSignOut = () => {
    signOut()
    navigate('/')
  }

  return (
    <header className="nav">
      <Link to="/" className="brand">
        <span className="brand-logo">Z</span>
        <span>ZPay</span>
      </Link>
      <nav className="nav-links">
        {user ? (
          <>
            <NavLink to="/dashboard">Dashboard</NavLink>
            <NavLink to="/add-credits">Add Credits</NavLink>
            <NavLink to="/pay">Pay</NavLink>
            <NavLink to="/transactions">Transactions</NavLink>
            <NavLink to="/profile">Profile</NavLink>
            <button className="btn btn-ghost" onClick={handleSignOut}>
              Sign out
            </button>
          </>
        ) : (
          <>
            <NavLink to="/">Home</NavLink>
            <NavLink to="/signin" className="btn btn-primary">
              Sign in
            </NavLink>
          </>
        )}
      </nav>
    </header>
  )
}
