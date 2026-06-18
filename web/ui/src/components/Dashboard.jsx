import { useState, useEffect } from 'react'
import { vhostAPI, logAPI, certAPI } from '../api'

function Dashboard() {
  const [stats, setStats] = useState({
    totalVHosts: 0,
    enabledVHosts: 0,
    totalCerts: 0,
    totalLogs: 0,
  })
  const [recentLogs, setRecentLogs] = useState([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(null)

  useEffect(() => {
    loadDashboardData()
    const interval = setInterval(loadDashboardData, 5000)
    return () => clearInterval(interval)
  }, [])

  const loadDashboardData = async () => {
    try {
      const [vhostsRes, certsRes, logsRes] = await Promise.all([
        vhostAPI.list(),
        certAPI.list(),
        logAPI.list({ limit: 10, offset: 0 }),
      ])

      const vhosts = vhostsRes.data
      setStats({
        totalVHosts: vhosts.length,
        enabledVHosts: vhosts.filter(v => v.enabled).length,
        totalCerts: certsRes.data.length,
        totalLogs: logsRes.data.total || 0,
      })

      setRecentLogs(logsRes.data.logs || [])
      setLoading(false)
      setError(null)
    } catch (err) {
      setError(err.message)
      setLoading(false)
    }
  }

  if (loading) {
    return <div className="loading">Loading dashboard...</div>
  }

  if (error) {
    return <div className="error">Error loading dashboard: {error}</div>
  }

  return (
    <div className="container">
      <h2 style={{ marginBottom: '30px', color: '#2c3e50' }}>Dashboard</h2>

      <div className="stats-grid">
        <div className="stat-card">
          <div className="stat-label">Virtual Hosts</div>
          <div className="stat-value">{stats.totalVHosts}</div>
        </div>
        <div className="stat-card">
          <div className="stat-label">Active Hosts</div>
          <div className="stat-value">{stats.enabledVHosts}</div>
        </div>
        <div className="stat-card">
          <div className="stat-label">Certificates</div>
          <div className="stat-value">{stats.totalCerts}</div>
        </div>
        <div className="stat-card">
          <div className="stat-label">Request Logs</div>
          <div className="stat-value">{stats.totalLogs}</div>
        </div>
      </div>

      {stats.totalVHosts === 0 && (
        <div className="card" style={{ backgroundColor: '#fff3cd', border: '2px solid #ffc107', marginBottom: '20px' }}>
          <h3 style={{ color: '#856404', marginBottom: '15px' }}>⚠️ No Virtual Hosts Configured</h3>
          <p style={{ color: '#856404', marginBottom: '15px' }}>
            SPECTRE requires at least one virtual host to be configured before it can intercept traffic.
            A virtual host maps an incoming hostname to a backend target URL.
          </p>

          <h4 style={{ color: '#856404', marginBottom: '10px', fontSize: '1.1rem' }}>📝 Option 1: Use the Web UI</h4>
          <p style={{ color: '#856404', marginBottom: '15px' }}>
            Navigate to the <strong>"Virtual Hosts"</strong> tab and click <strong>"+ Add Virtual Host"</strong> to create your first virtual host.
          </p>

          <h4 style={{ color: '#856404', marginBottom: '10px', fontSize: '1.1rem' }}>🔧 Option 2: Use the API</h4>
          <p style={{ color: '#856404', marginBottom: '10px' }}>Create a virtual host using curl:</p>
          <pre style={{ backgroundColor: '#fff', border: '1px solid #ddd', padding: '15px', borderRadius: '5px', color: '#333' }}>
{`curl -X POST http://localhost:9000/api/v1/vhosts \\
  -H "Content-Type: application/json" \\
  -d '{
    "hostname": "api.example.com",
    "target_url": "https://real-backend.example.com"
  }'`}
          </pre>

          <h4 style={{ color: '#856404', marginTop: '15px', marginBottom: '10px', fontSize: '1.1rem' }}>📖 Example Use Case</h4>
          <p style={{ color: '#856404' }}>
            To intercept traffic to <code>api.github.com</code> and forward it to the real GitHub API:
          </p>
          <ol style={{ marginLeft: '20px', marginTop: '10px', color: '#856404' }}>
            <li>Create a virtual host with hostname <code>api.github.com</code> and target <code>https://api.github.com</code></li>
            <li>Add <code>127.0.0.1 api.github.com</code> to your <code>/etc/hosts</code> file</li>
            <li>Install the SPECTRE Root CA certificate (see Quick Start below)</li>
            <li>Make requests to <code>https://api.github.com:8443</code> and watch them appear in the Request Logs</li>
          </ol>
        </div>
      )}

      <div className="card">
        <div className="card-header">
          <h3 className="card-title">Recent Requests</h3>
          <button className="btn-primary" onClick={loadDashboardData}>
            Refresh
          </button>
        </div>

        {recentLogs.length === 0 ? (
          <p style={{ textAlign: 'center', color: '#95a5a6', padding: '20px' }}>
            No requests logged yet. Start intercepting traffic!
          </p>
        ) : (
          <table>
            <thead>
              <tr>
                <th>Time</th>
                <th>Method</th>
                <th>URL</th>
                <th>Status</th>
                <th>Duration</th>
                <th>Client IP</th>
              </tr>
            </thead>
            <tbody>
              {recentLogs.map((log) => (
                <tr key={log.id}>
                  <td>{new Date(log.timestamp).toLocaleString()}</td>
                  <td>
                    <span className="badge badge-info">{log.method}</span>
                  </td>
                  <td style={{ maxWidth: '400px', overflow: 'hidden', textOverflow: 'ellipsis' }}>
                    {log.url}
                  </td>
                  <td>
                    <span className={`badge ${log.status_code >= 200 && log.status_code < 300 ? 'badge-success' : 'badge-danger'}`}>
                      {log.status_code}
                    </span>
                  </td>
                  <td>{log.duration_ms}ms</td>
                  <td>{log.client_ip}</td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>

      <div className="card">
        <h3 className="card-title" style={{ marginBottom: '15px' }}>Quick Start</h3>
        <div style={{ lineHeight: '1.8' }}>
          <p><strong>1. Install Root CA:</strong></p>
          <pre>curl http://localhost:9000/api/v1/certificates/ca -o spectre-ca.crt
sudo security add-trusted-cert -d -r trustRoot -k /Library/Keychains/System.keychain spectre-ca.crt</pre>

          <p style={{ marginTop: '15px' }}><strong>2. Add a virtual host via the "Virtual Hosts" tab</strong></p>

          <p style={{ marginTop: '15px' }}><strong>3. Route traffic to proxy:</strong></p>
          <pre>echo "127.0.0.1 api.example.com" | sudo tee -a /etc/hosts</pre>

          <p style={{ marginTop: '15px' }}><strong>4. Test interception:</strong></p>
          <pre>curl https://api.example.com:8443/endpoint</pre>
        </div>
      </div>
    </div>
  )
}

export default Dashboard
