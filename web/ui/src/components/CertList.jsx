import { useState, useEffect } from 'react'
import { certAPI } from '../api'

function CertList() {
  const [certs, setCerts] = useState([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(null)
  const [showCAForm, setShowCAForm] = useState(false)
  const [caForm, setCAForm] = useState({
    common_name: '',
    organization: '',
    organizational_unit: '',
    country: '',
    province: '',
    locality: '',
    validity_years: 10
  })
  const [caReplacing, setCAReplacing] = useState(false)

  useEffect(() => {
    loadCerts()
  }, [])

  const loadCerts = async () => {
    try {
      const response = await certAPI.list()
      setCerts(response.data || [])
      setLoading(false)
      setError(null)
    } catch (err) {
      setError(err.message)
      setLoading(false)
    }
  }

  const handleDelete = async (id) => {
    if (!confirm('Are you sure you want to delete this certificate?')) {
      return
    }
    try {
      await certAPI.delete(id)
      loadCerts()
    } catch (err) {
      alert(`Error: ${err.response?.data?.error || err.message}`)
    }
  }

  const handleDownloadCA = () => {
    certAPI.downloadCA()
  }

  const handleReplaceCA = async (e) => {
    e.preventDefault()

    if (!caForm.common_name.trim()) {
      alert('Common Name is required')
      return
    }

    if (!confirm('Are you sure you want to replace the Root CA? This will invalidate all existing certificates and they will be regenerated on-demand.')) {
      return
    }

    setCAReplacing(true)
    try {
      await certAPI.replaceCA(caForm)
      alert('CA certificate replaced successfully! Download the new CA and install it in your trust store.')
      setShowCAForm(false)
      setCAForm({
        common_name: '',
        organization: '',
        organizational_unit: '',
        country: '',
        province: '',
        locality: '',
        validity_years: 10
      })
      loadCerts()
    } catch (err) {
      alert(`Error: ${err.response?.data?.error || err.message}`)
    } finally {
      setCAReplacing(false)
    }
  }

  const handleCAFormChange = (field, value) => {
    setCAForm(prev => ({ ...prev, [field]: value }))
  }

  if (loading) {
    return <div className="loading">Loading certificates...</div>
  }

  return (
    <div className="container">
      <div className="card">
        <div className="card-header">
          <h2 className="card-title">SSL Certificates</h2>
          <div style={{ display: 'flex', gap: '10px' }}>
            <button className="btn-success" onClick={handleDownloadCA}>
              ⬇ Download Root CA
            </button>
            <button
              className="btn-primary"
              onClick={() => setShowCAForm(!showCAForm)}
              style={{ backgroundColor: '#3498db' }}
            >
              {showCAForm ? '✖ Cancel' : '🔄 Replace Root CA'}
            </button>
          </div>
        </div>

        {error && <div className="error">Error: {error}</div>}

        {showCAForm && (
          <div style={{ padding: '20px', backgroundColor: '#f8f9fa', border: '2px solid #3498db', borderRadius: '8px', marginBottom: '20px' }}>
            <h3 style={{ marginTop: 0, color: '#2c3e50' }}>Replace Root CA Certificate</h3>
            <p style={{ color: '#7f8c8d', marginBottom: '20px' }}>
              Create a new Root CA with custom certificate fields. All existing virtual host certificates will be invalidated and regenerated on-demand.
            </p>

            <form onSubmit={handleReplaceCA}>
              <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '15px' }}>
                <div>
                  <label style={{ display: 'block', marginBottom: '5px', fontWeight: 'bold', color: '#2c3e50' }}>
                    Common Name (CN) *
                  </label>
                  <input
                    type="text"
                    value={caForm.common_name}
                    onChange={(e) => handleCAFormChange('common_name', e.target.value)}
                    placeholder="e.g., My Company Root CA"
                    required
                    style={{ width: '100%', padding: '10px', border: '1px solid #ddd', borderRadius: '5px' }}
                  />
                </div>

                <div>
                  <label style={{ display: 'block', marginBottom: '5px', fontWeight: 'bold', color: '#2c3e50' }}>
                    Organization (O)
                  </label>
                  <input
                    type="text"
                    value={caForm.organization}
                    onChange={(e) => handleCAFormChange('organization', e.target.value)}
                    placeholder="e.g., My Company Inc"
                    style={{ width: '100%', padding: '10px', border: '1px solid #ddd', borderRadius: '5px' }}
                  />
                </div>

                <div>
                  <label style={{ display: 'block', marginBottom: '5px', fontWeight: 'bold', color: '#2c3e50' }}>
                    Organizational Unit (OU)
                  </label>
                  <input
                    type="text"
                    value={caForm.organizational_unit}
                    onChange={(e) => handleCAFormChange('organizational_unit', e.target.value)}
                    placeholder="e.g., IT Security"
                    style={{ width: '100%', padding: '10px', border: '1px solid #ddd', borderRadius: '5px' }}
                  />
                </div>

                <div>
                  <label style={{ display: 'block', marginBottom: '5px', fontWeight: 'bold', color: '#2c3e50' }}>
                    Country (C)
                  </label>
                  <input
                    type="text"
                    value={caForm.country}
                    onChange={(e) => handleCAFormChange('country', e.target.value)}
                    placeholder="e.g., US"
                    maxLength={2}
                    style={{ width: '100%', padding: '10px', border: '1px solid #ddd', borderRadius: '5px' }}
                  />
                </div>

                <div>
                  <label style={{ display: 'block', marginBottom: '5px', fontWeight: 'bold', color: '#2c3e50' }}>
                    State/Province (ST)
                  </label>
                  <input
                    type="text"
                    value={caForm.province}
                    onChange={(e) => handleCAFormChange('province', e.target.value)}
                    placeholder="e.g., California"
                    style={{ width: '100%', padding: '10px', border: '1px solid #ddd', borderRadius: '5px' }}
                  />
                </div>

                <div>
                  <label style={{ display: 'block', marginBottom: '5px', fontWeight: 'bold', color: '#2c3e50' }}>
                    Locality/City (L)
                  </label>
                  <input
                    type="text"
                    value={caForm.locality}
                    onChange={(e) => handleCAFormChange('locality', e.target.value)}
                    placeholder="e.g., San Francisco"
                    style={{ width: '100%', padding: '10px', border: '1px solid #ddd', borderRadius: '5px' }}
                  />
                </div>

                <div>
                  <label style={{ display: 'block', marginBottom: '5px', fontWeight: 'bold', color: '#2c3e50' }}>
                    Validity (Years)
                  </label>
                  <input
                    type="number"
                    value={caForm.validity_years}
                    onChange={(e) => handleCAFormChange('validity_years', parseInt(e.target.value))}
                    min="1"
                    max="30"
                    style={{ width: '100%', padding: '10px', border: '1px solid #ddd', borderRadius: '5px' }}
                  />
                </div>
              </div>

              <div style={{ marginTop: '20px', display: 'flex', gap: '10px' }}>
                <button
                  type="submit"
                  className="btn-success"
                  disabled={caReplacing}
                  style={{ padding: '12px 24px' }}
                >
                  {caReplacing ? '⏳ Replacing CA...' : '✅ Replace CA Certificate'}
                </button>
                <button
                  type="button"
                  className="btn-secondary"
                  onClick={() => setShowCAForm(false)}
                  style={{ padding: '12px 24px', backgroundColor: '#95a5a6' }}
                >
                  Cancel
                </button>
              </div>
            </form>
          </div>
        )}

        <div style={{ padding: '15px', backgroundColor: '#fff3cd', border: '1px solid #ffc107', borderRadius: '5px', marginBottom: '20px' }}>
          <strong>⚠️ Installation Required:</strong> To intercept HTTPS traffic, you must install the Root CA certificate in your system's trust store.
          <br /><br />
          <strong>macOS:</strong>
          <pre style={{ marginTop: '10px' }}>curl http://localhost:9000/api/v1/certificates/ca -o spectre-ca.crt
sudo security add-trusted-cert -d -r trustRoot -k /Library/Keychains/System.keychain spectre-ca.crt</pre>
          <strong>Linux:</strong>
          <pre style={{ marginTop: '10px' }}>curl http://localhost:9000/api/v1/certificates/ca -o spectre-ca.crt
sudo cp spectre-ca.crt /usr/local/share/ca-certificates/
sudo update-ca-certificates</pre>
          <strong>Windows:</strong>
          <pre style={{ marginTop: '10px' }}>curl http://localhost:9000/api/v1/certificates/ca -o spectre-ca.crt
certutil -addstore -f "ROOT" spectre-ca.crt</pre>
        </div>

        <h3 style={{ marginTop: '30px', marginBottom: '15px', color: '#2c3e50' }}>
          Generated Certificates ({certs.length})
        </h3>

        {certs.length === 0 ? (
          <p style={{ textAlign: 'center', color: '#95a5a6', padding: '40px' }}>
            No certificates generated yet. Certificates are created on-demand when intercepting HTTPS traffic.
          </p>
        ) : (
          <table>
            <thead>
              <tr>
                <th>Hostname</th>
                <th>Created</th>
                <th>Expires</th>
                <th>Valid For</th>
                <th>Actions</th>
              </tr>
            </thead>
            <tbody>
              {certs.map((cert) => {
                const created = new Date(cert.created_at)
                const expires = new Date(cert.expires_at)
                const now = new Date()
                const daysLeft = Math.ceil((expires - now) / (1000 * 60 * 60 * 24))
                const isExpired = daysLeft < 0
                const isExpiringSoon = daysLeft < 30 && daysLeft >= 0

                return (
                  <tr key={cert.id}>
                    <td>
                      <strong>{cert.hostname}</strong>
                    </td>
                    <td>{created.toLocaleDateString()}</td>
                    <td>{expires.toLocaleDateString()}</td>
                    <td>
                      <span
                        className={`badge ${isExpired ? 'badge-danger' : isExpiringSoon ? 'badge-warning' : 'badge-success'}`}
                      >
                        {isExpired ? 'Expired' : `${daysLeft} days`}
                      </span>
                    </td>
                    <td>
                      <button
                        className="btn-danger"
                        onClick={() => handleDelete(cert.id)}
                      >
                        Delete
                      </button>
                    </td>
                  </tr>
                )
              })}
            </tbody>
          </table>
        )}
      </div>

      <div className="card">
        <h3 className="card-title" style={{ marginBottom: '15px' }}>How It Works</h3>
        <div style={{ lineHeight: '1.8' }}>
          <p><strong>1. Root CA Generation:</strong> On first startup, SPECTRE generates a self-signed Root CA certificate.</p>

          <p style={{ marginTop: '15px' }}><strong>2. On-Demand Certificate Generation:</strong> When a client connects via HTTPS, SPECTRE:</p>
          <ul style={{ marginLeft: '20px', marginTop: '10px' }}>
            <li>Extracts the hostname from SNI (Server Name Indication)</li>
            <li>Generates a new SSL certificate signed by the Root CA</li>
            <li>Caches the certificate in memory and database</li>
            <li>Presents it to the client</li>
          </ul>

          <p style={{ marginTop: '15px' }}><strong>3. Certificate Caching:</strong> Generated certificates are cached to improve performance on subsequent connections.</p>

          <p style={{ marginTop: '15px' }}><strong>4. Certificate Properties:</strong></p>
          <ul style={{ marginLeft: '20px', marginTop: '10px' }}>
            <li>Algorithm: RSA 2048-bit</li>
            <li>Validity: 365 days (1 year)</li>
            <li>Subject Alternative Name (SAN): Includes the hostname</li>
            <li>Issuer: SPECTRE Root CA</li>
          </ul>
        </div>
      </div>
    </div>
  )
}

export default CertList
