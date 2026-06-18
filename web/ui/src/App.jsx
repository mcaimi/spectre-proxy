import { useState } from 'react'
import { BrowserRouter as Router, Routes, Route, Link, NavLink } from 'react-router-dom'
import Dashboard from './components/Dashboard'
import VHostList from './components/VHostList'
import CertList from './components/CertList'
import RequestLog from './components/RequestLog'
import './App.css'

function App() {
  return (
    <Router>
      <div className="app">
        <nav className="navbar">
          <div className="navbar-brand">
            <h1>🔍 SPECTRE</h1>
            <span className="navbar-subtitle">HTTP/HTTPS Interception Proxy</span>
          </div>
          <div className="navbar-menu">
            <NavLink to="/" className={({ isActive }) => isActive ? 'nav-link active' : 'nav-link'}>
              Dashboard
            </NavLink>
            <NavLink to="/vhosts" className={({ isActive }) => isActive ? 'nav-link active' : 'nav-link'}>
              Virtual Hosts
            </NavLink>
            <NavLink to="/certificates" className={({ isActive }) => isActive ? 'nav-link active' : 'nav-link'}>
              Certificates
            </NavLink>
            <NavLink to="/logs" className={({ isActive }) => isActive ? 'nav-link active' : 'nav-link'}>
              Request Logs
            </NavLink>
          </div>
        </nav>

        <main className="main-content">
          <Routes>
            <Route path="/" element={<Dashboard />} />
            <Route path="/vhosts" element={<VHostList />} />
            <Route path="/certificates" element={<CertList />} />
            <Route path="/logs" element={<RequestLog />} />
          </Routes>
        </main>

        <footer className="footer">
          <p>SPECTRE - Secure Protocol Examination and Capture Tool for Reverse Engineering</p>
        </footer>
      </div>
    </Router>
  )
}

export default App
