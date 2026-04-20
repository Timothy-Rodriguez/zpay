import { useAuth } from '../context/AuthContext.jsx'
import { useWallet } from '../context/WalletContext.jsx'
import { formatCurrency, formatDate } from '../utils/format.js'

export default function Profile() {
  const { user, signOut } = useAuth()
  const { credits, transactions } = useWallet()

  return (
    <section className="page narrow">
      <h1>Profile</h1>

      <div className="panel profile">
        <div className="avatar">{user?.name?.[0]?.toUpperCase()}</div>
        <div>
          <h2>{user?.name}</h2>
          <p className="muted">{user?.email}</p>
          <p className="muted small">
            Member since {formatDate(user?.joinedAt)}
          </p>
        </div>
      </div>

      <div className="panel">
        <h3>Wallet summary</h3>
        <div className="kv">
          <div>
            <span className="muted">Available credits</span>
            <strong>{formatCurrency(credits)}</strong>
          </div>
          <div>
            <span className="muted">Transactions</span>
            <strong>{transactions.length}</strong>
          </div>
        </div>
      </div>

      <button className="btn btn-outline" onClick={signOut}>
        Sign out
      </button>
    </section>
  )
}
