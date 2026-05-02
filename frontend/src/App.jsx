import { Route, Routes } from 'react-router-dom'
import Navbar from './components/Navbar.jsx'
import Footer from './components/Footer.jsx'
import ProtectedRoute from './components/ProtectedRoute.jsx'
import Landing from './pages/Landing.jsx'
import SignIn from './pages/SignIn.jsx'
import CreateUser from './pages/CreateUser.jsx'
import LoginUser from './pages/LoginUser.jsx'
import Payment from './pages/Payment.jsx'
import PaymentProcess from './pages/PaymentProcess.jsx'
import Dashboard from './pages/Dashboard.jsx'
import Pay from './pages/Pay.jsx'
import Transactions from './pages/Transactions.jsx'
import Profile from './pages/Profile.jsx'
import NotFound from './pages/NotFound.jsx'
import './App.css'

function App() {
  return (
    <div className="app">
      <Navbar />
      <main className="main">
        <Routes>
          <Route path="/" element={<Landing />} />
          <Route path="/signin" element={<SignIn />} />
          <Route path="/create-user" element={<CreateUser />} />
          <Route path="/login-user" element={<LoginUser />} />
          <Route path="/payment" element={<ProtectedRoute><Payment /></ProtectedRoute>} />
          <Route path="/payment-process" element={<ProtectedRoute><PaymentProcess /></ProtectedRoute>} />
          <Route path="/dashboard" element={<ProtectedRoute><Dashboard /></ProtectedRoute>} />
          <Route path="/pay" element={<ProtectedRoute><Pay /></ProtectedRoute>} />
          <Route path="/transactions" element={<ProtectedRoute><Transactions /></ProtectedRoute>} />
          <Route path="/profile" element={<ProtectedRoute><Profile /></ProtectedRoute>} />
          <Route path="*" element={<NotFound />} />
        </Routes>
      </main>
      <Footer />
    </div>
  )
}

export default App
