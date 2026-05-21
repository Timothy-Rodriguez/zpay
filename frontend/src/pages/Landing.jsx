import { Link } from 'react-router-dom'

export default function Landing() {
  return (
    <>
      <section className="hero-section">
        <div className="hero-content">
          <span className="pill">Payment Simulation · Distributed Systems · Kubernetes</span>
          <h1>
            A real-world payment system <span className="gradient">built to scale.</span>
          </h1>
          <p className="lead">
            ZPay is a payment simulation and ledger platform developed with Go Gin and React framework that implements
            production-grade idempotency, deterministic locking, event-driven
            architecture, and distributed system patterns — deployed on
            Kubernetes cluster with full observability.
          </p>
          <div className="cta-row">
            <Link to="/create-user" className="btn btn-primary btn-lg">
              Try it out
            </Link>
            <a href="#what-is-zpay" className="btn btn-ghost btn-lg">
              Learn more
            </a>
          </div>
          <div className="trust">
            <div>
              <strong>Idempotent</strong>
              <span>Transactions</span>
            </div>
            <div>
              <strong>Kubernetes</strong>
              <span>Deployed</span>
            </div>
            <div>
              <strong>Event-Driven</strong>
              <span>Architecture</span>
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

      <section id="what-is-zpay" className="features">
        <h2>What is ZPay?</h2>
        <div className="feature-grid">
          <div className="feature">
            <div className="feature-icon">💳</div>
            <h3>Payment Simulation & Ledger</h3>
            <p>
              A fully functional payment platform simulating real-world
              money movement with a double-entry ledger, balance tracking,
              and transaction history.
            </p>
          </div>
          <div className="feature">
            <div className="feature-icon">🔒</div>
            <h3>Idempotency & Deterministic Locks</h3>
            <p>
              Every transaction is protected by idempotency keys and
              deterministic row-level locking to prevent double spends
              and race conditions under concurrent load.
            </p>
          </div>
          <div className="feature">
            <div className="feature-icon">☸️</div>
            <h3>Kubernetes Deployment</h3>
            <p>
              The entire stack — API, Kafka, Postgres, Redis, and
              observability — runs on Kubernetes with namespaced
              resources, persistent volumes, and ingress routing.
            </p>
          </div>
          <div className="feature">
            <div className="feature-icon">📨</div>
            <h3>Event-Driven Architecture</h3>
            <p>
              Payments publish events to Kafka via a transactional outbox
              pattern. Consumers process receipts asynchronously and
              dispatch email notifications, mimicking real fintech pipelines.
            </p>
          </div>
          <div className="feature">
            <div className="feature-icon">📊</div>
            <h3>Full Observability</h3>
            <p>
              Distributed tracing with Jaeger, metrics with Prometheus &amp;
              Grafana, and structured log aggregation with Loki &amp; Promtail
              give end-to-end visibility into every request.
            </p>
          </div>
          <div className="feature">
            <div className="feature-icon">🚀</div>
            <h3>CI/CD with GitLab</h3>
            <p>
              Automated build, test, and deploy pipelines via GitLab CI/CD
              keep the platform continuously integrated and shipped to the
              cluster with zero manual steps.
            </p>
          </div>
        </div>
      </section>

      <section className="features about-section">
        <h2>About Me</h2>
        <div className="about-body">
          <p>
            I'm <strong>Timothy Rodriguez</strong>, a Backend Developer with
            4 years of experience designing, developing, and scaling
            production-grade systems using <strong>Golang</strong>,{' '}
            <strong>Python</strong>, <strong>Java</strong>, REST APIs, and
            SQL/NoSQL databases. ZPay is one of my projects that brings
            together distributed systems, cloud-native deployment, and
            real-world engineering patterns in a single platform.
          </p>
          <p>Want to connect?</p>
          <div className="about-links">
            <a
              href="https://www.linkedin.com/in/timothy-rodriguez-86a8b1200/"
              target="_blank"
              rel="noopener noreferrer"
              className="btn btn-ghost"
            >
              LinkedIn
            </a>
            <a
              href="https://leetcode.com/u/mtimothyrodriguez2000/"
              target="_blank"
              rel="noopener noreferrer"
              className="btn btn-ghost"
            >
              LeetCode
            </a>
            <a
              href="https://github.com/Timothy-Rodriguez"
              target="_blank"
              rel="noopener noreferrer"
              className="btn btn-ghost"
            >
              GitHub
            </a>
          </div>
        </div>
      </section>
    </>
  )
}
