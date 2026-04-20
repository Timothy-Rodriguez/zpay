import { Link } from 'react-router-dom'

export default function Landing() {
  return (
    <>
      <section className="hero-section">
        <div className="hero-content">
          <span className="pill">New · Instant passwordless sign-in</span>
          <h1>
            Money that moves <span className="gradient">at your speed.</span>
          </h1>
          <p className="lead">
            ZPay is a modern fintech wallet to store credits, top up in seconds,
            and send payments to anyone — all with bank-grade security and zero
            passwords.
          </p>
          <div className="cta-row">
            <Link to="/signin" className="btn btn-primary btn-lg">
              Get started free
            </Link>
            <a href="#features" className="btn btn-ghost btn-lg">
              Learn more
            </a>
          </div>
          <div className="trust">
            <div>
              <strong>₹2.4B+</strong>
              <span>Processed</span>
            </div>
            <div>
              <strong>180k</strong>
              <span>Active users</span>
            </div>
            <div>
              <strong>99.99%</strong>
              <span>Uptime</span>
            </div>
          </div>
        </div>
        <div className="hero-card">
          <div className="card-mock">
            <div className="card-mock-top">
              <span>ZPay Wallet</span>
              <span className="chip">VISA</span>
            </div>
            <div className="card-mock-balance">
              <small>Available credits</small>
              <h2>1,000.00</h2>
            </div>
            <div className="card-mock-bottom">
              <span>•••• 4242</span>
              <span>12/29</span>
            </div>
          </div>
        </div>
      </section>

      <section id="features" className="features">
        <h2>Why ZPay?</h2>
        <div className="feature-grid">
          <div className="feature">
            <div className="feature-icon">⚡</div>
            <h3>Instant transfers</h3>
            <p>Send credits to anyone in seconds with zero fees.</p>
          </div>
          <div className="feature">
            <div className="feature-icon">🔐</div>
            <h3>Passwordless security</h3>
            <p>Sign in with just your email — no passwords to forget or leak.</p>
          </div>
          <div className="feature">
            <div className="feature-icon">💳</div>
            <h3>Top-up anytime</h3>
            <p>Add credits via card, UPI, or net banking in one tap.</p>
          </div>
          <div className="feature">
            <div className="feature-icon">📊</div>
            <h3>Live insights</h3>
            <p>Track every transaction with a clean, unified dashboard.</p>
          </div>
        </div>
      </section>
    </>
  )
}
