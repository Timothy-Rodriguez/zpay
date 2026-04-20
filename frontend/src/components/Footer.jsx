export default function Footer() {
  return (
    <footer className="footer">
      <span>© {new Date().getFullYear()} ZPay. All rights reserved.</span>
      <span className="muted">Built with React · Secured by design</span>
    </footer>
  )
}
