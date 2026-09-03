import { useState, useEffect } from 'react'
import { vhostAPI } from '../api'

function VHostList() {
  const [vhosts, setVHosts] = useState([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(null)
  const [showModal, setShowModal] = useState(false)
  const [editingVHost, setEditingVHost] = useState(null)
  const [formData, setFormData] = useState({
    hostname: '',
    target_url: '',
    enabled: true,
  })

  useEffect(() => {
    loadVHosts()
  }, [])

  const loadVHosts = async () => {
    try {
      const response = await vhostAPI.list()
      setVHosts(response.data || [])
      setLoading(false)
      setError(null)
    } catch (err) {
      setError(err.message)
      setLoading(false)
    }
  }

  const handleCreate = () => {
    setEditingVHost(null)
    setFormData({ hostname: '', target_url: '', enabled: true })
    setShowModal(true)
  }

  const handleEdit = (vhost) => {
    setEditingVHost(vhost)
    setFormData({
      hostname: vhost.hostname,
      target_url: vhost.target_url,
      enabled: vhost.enabled,
    })
    setShowModal(true)
  }

  const handleSubmit = async (e) => {
    e.preventDefault()
    try {
      if (editingVHost) {
        await vhostAPI.update(editingVHost.id, formData)
      } else {
        await vhostAPI.create(formData)
      }
      setShowModal(false)
      loadVHosts()
    } catch (err) {
      alert(`Error: ${err.response?.data?.error || err.message}`)
    }
  }

  const handleDelete = async (id) => {
    if (!confirm('Are you sure you want to delete this virtual host?')) {
      return
    }
    try {
      await vhostAPI.delete(id)
      loadVHosts()
    } catch (err) {
      alert(`Error: ${err.response?.data?.error || err.message}`)
    }
  }

  const handleExportHAR = async (vhost) => {
    try {
      const response = await vhostAPI.exportHAR(vhost.id)
      const disposition = response.headers['content-disposition']
      let filename = `${vhost.hostname}.har`
      if (disposition) {
        const match = disposition.match(/filename="?(.+?)"?$/)
        if (match) filename = match[1]
      }
      const url = URL.createObjectURL(response.data)
      const a = document.createElement('a')
      a.href = url
      a.download = filename
      a.click()
      URL.revokeObjectURL(url)
    } catch (err) {
      alert(`Error: ${err.response?.data?.error || err.message}`)
    }
  }

  const handleToggleEnabled = async (vhost) => {
    try {
      await vhostAPI.update(vhost.id, {
        hostname: vhost.hostname,
        target_url: vhost.target_url,
        enabled: !vhost.enabled,
      })
      loadVHosts()
    } catch (err) {
      alert(`Error: ${err.response?.data?.error || err.message}`)
    }
  }

  if (loading) {
    return <div className="loading">Loading virtual hosts...</div>
  }

  return (
    <div className="container">
      <div className="card">
        <div className="card-header">
          <h2 className="card-title">Virtual Hosts</h2>
          <button className="btn-primary" onClick={handleCreate}>
            + Add Virtual Host
          </button>
        </div>

        {error && <div className="error">Error: {error}</div>}

        {vhosts.length === 0 ? (
          <p style={{ textAlign: 'center', color: '#95a5a6', padding: '40px' }}>
            No virtual hosts configured. Click "Add Virtual Host" to get started.
          </p>
        ) : (
          <table>
            <thead>
              <tr>
                <th>Hostname</th>
                <th>Target URL</th>
                <th>Status</th>
                <th>Created</th>
                <th>Actions</th>
              </tr>
            </thead>
            <tbody>
              {vhosts.map((vhost) => (
                <tr key={vhost.id}>
                  <td>
                    <strong>{vhost.hostname}</strong>
                  </td>
                  <td>{vhost.target_url}</td>
                  <td>
                    <span
                      className={`badge ${vhost.enabled ? 'badge-success' : 'badge-danger'}`}
                      style={{ cursor: 'pointer' }}
                      onClick={() => handleToggleEnabled(vhost)}
                      title="Click to toggle"
                    >
                      {vhost.enabled ? 'Enabled' : 'Disabled'}
                    </span>
                  </td>
                  <td>{new Date(vhost.created_at).toLocaleDateString()}</td>
                  <td>
                    <button
                      className="btn-secondary"
                      onClick={() => handleEdit(vhost)}
                      style={{ marginRight: '10px' }}
                    >
                      Edit
                    </button>
                    <button
                      className="btn-secondary"
                      onClick={() => handleExportHAR(vhost)}
                      style={{ marginRight: '10px' }}
                    >
                      Export HAR
                    </button>
                    <button
                      className="btn-danger"
                      onClick={() => handleDelete(vhost.id)}
                    >
                      Delete
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>

      {showModal && (
        <div className="modal-overlay" onClick={() => setShowModal(false)}>
          <div className="modal" onClick={(e) => e.stopPropagation()}>
            <div className="modal-header">
              <h3 className="modal-title">
                {editingVHost ? 'Edit Virtual Host' : 'Add Virtual Host'}
              </h3>
            </div>

            <form onSubmit={handleSubmit}>
              <div className="form-group">
                <label>Hostname *</label>
                <input
                  type="text"
                  value={formData.hostname}
                  onChange={(e) => setFormData({ ...formData, hostname: e.target.value })}
                  placeholder="api.example.com or *.example.com"
                  required
                />
                <small style={{ color: '#7f8c8d', fontSize: '0.875rem' }}>
                  Exact hostname or wildcard pattern (e.g., *.example.com)
                </small>
              </div>

              <div className="form-group">
                <label>Target URL *</label>
                <input
                  type="url"
                  value={formData.target_url}
                  onChange={(e) => setFormData({ ...formData, target_url: e.target.value })}
                  placeholder="https://real-backend.example.com"
                  required
                />
                <small style={{ color: '#7f8c8d', fontSize: '0.875rem' }}>
                  Backend server to forward requests to
                </small>
              </div>

              <div className="checkbox-container">
                <input
                  type="checkbox"
                  id="enabled"
                  checked={formData.enabled}
                  onChange={(e) => setFormData({ ...formData, enabled: e.target.checked })}
                />
                <label htmlFor="enabled" style={{ marginBottom: 0 }}>
                  Enabled
                </label>
              </div>

              <div className="modal-footer">
                <button
                  type="button"
                  className="btn-secondary"
                  onClick={() => setShowModal(false)}
                >
                  Cancel
                </button>
                <button type="submit" className="btn-primary">
                  {editingVHost ? 'Update' : 'Create'}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  )
}

export default VHostList
