import { useState, useEffect } from 'react'
import { logAPI } from '../api'

function RequestLog() {
  const [logs, setLogs] = useState([])
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(null)
  const [selectedLog, setSelectedLog] = useState(null)
  const [showDetail, setShowDetail] = useState(false)
  const [limit, setLimit] = useState(50)
  const [offset, setOffset] = useState(0)
  const [autoRefresh, setAutoRefresh] = useState(true)

  useEffect(() => {
    loadLogs()
  }, [limit, offset])

  useEffect(() => {
    if (!autoRefresh) return
    const interval = setInterval(loadLogs, 3000)
    return () => clearInterval(interval)
  }, [autoRefresh, limit, offset])

  const loadLogs = async () => {
    try {
      const response = await logAPI.list({ limit, offset })
      setLogs(response.data.logs || [])
      setTotal(response.data.total || 0)
      setLoading(false)
      setError(null)
    } catch (err) {
      setError(err.message)
      setLoading(false)
    }
  }

  const handleViewDetails = async (log) => {
    try {
      const response = await logAPI.get(log.id)
      setSelectedLog(response.data)
      setShowDetail(true)
    } catch (err) {
      alert(`Error: ${err.response?.data?.error || err.message}`)
    }
  }

  const handleNextPage = () => {
    setOffset(offset + limit)
  }

  const handlePrevPage = () => {
    setOffset(Math.max(0, offset - limit))
  }

  const formatHeaders = (headersJson) => {
    try {
      const headers = JSON.parse(headersJson)
      return Object.entries(headers).map(([key, values]) =>
        `${key}: ${Array.isArray(values) ? values.join(', ') : values}`
      ).join('\n')
    } catch {
      return headersJson
    }
  }

  const formatBody = (body) => {
    if (!body || body.length === 0) return '(empty)'
    try {
      const text = atob(body)
      try {
        const json = JSON.parse(text)
        return JSON.stringify(json, null, 2)
      } catch {
        return text
      }
    } catch {
      return '(binary data)'
    }
  }

  if (loading && logs.length === 0) {
    return <div className="loading">Loading request logs...</div>
  }

  return (
    <div className="container">
      <div className="card">
        <div className="card-header">
          <h2 className="card-title">Request Logs ({total})</h2>
          <div style={{ display: 'flex', gap: '10px', alignItems: 'center' }}>
            <label style={{ marginBottom: 0 }}>
              <input
                type="checkbox"
                checked={autoRefresh}
                onChange={(e) => setAutoRefresh(e.target.checked)}
                style={{ marginRight: '5px' }}
              />
              Auto-refresh
            </label>
            <button className="btn-primary" onClick={loadLogs}>
              Refresh
            </button>
          </div>
        </div>

        {error && <div className="error">Error: {error}</div>}

        {logs.length === 0 ? (
          <p style={{ textAlign: 'center', color: '#95a5a6', padding: '40px' }}>
            No requests logged yet. Traffic will appear here as it's intercepted.
          </p>
        ) : (
          <>
            <div style={{ overflowX: 'auto' }}>
              <table>
                <thead>
                  <tr>
                    <th>ID</th>
                    <th>Time</th>
                    <th>Method</th>
                    <th>URL</th>
                    <th>Status</th>
                    <th>Duration</th>
                    <th>Client IP</th>
                    <th>Actions</th>
                  </tr>
                </thead>
                <tbody>
                  {logs.map((log) => (
                    <tr key={log.id}>
                      <td>{log.id}</td>
                      <td style={{ whiteSpace: 'nowrap' }}>
                        {new Date(log.timestamp).toLocaleString()}
                      </td>
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
                      <td>
                        <button
                          className="btn-secondary"
                          onClick={() => handleViewDetails(log)}
                        >
                          View
                        </button>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>

            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginTop: '20px', padding: '15px 0', borderTop: '1px solid #e0e0e0' }}>
              <div>
                Showing {offset + 1} - {Math.min(offset + limit, total)} of {total}
              </div>
              <div style={{ display: 'flex', gap: '10px' }}>
                <button
                  className="btn-secondary"
                  onClick={handlePrevPage}
                  disabled={offset === 0}
                >
                  Previous
                </button>
                <button
                  className="btn-secondary"
                  onClick={handleNextPage}
                  disabled={offset + limit >= total}
                >
                  Next
                </button>
              </div>
            </div>
          </>
        )}
      </div>

      {showDetail && selectedLog && (
        <div className="modal-overlay" onClick={() => setShowDetail(false)}>
          <div className="modal" style={{ maxWidth: '900px' }} onClick={(e) => e.stopPropagation()}>
            <div className="modal-header">
              <h3 className="modal-title">Request Details - ID: {selectedLog.id}</h3>
            </div>

            <div style={{ marginBottom: '20px' }}>
              <h4 style={{ marginBottom: '10px', color: '#2c3e50' }}>Request</h4>
              <div style={{ marginBottom: '10px' }}>
                <strong>{selectedLog.method}</strong> {selectedLog.url}
              </div>
              <div style={{ marginBottom: '10px', fontSize: '0.875rem', color: '#7f8c8d' }}>
                <strong>Time:</strong> {new Date(selectedLog.timestamp).toLocaleString()} |
                <strong> Duration:</strong> {selectedLog.duration_ms}ms |
                <strong> Client:</strong> {selectedLog.client_ip}
              </div>

              <div style={{ marginTop: '15px' }}>
                <strong>Request Headers:</strong>
                <pre>{formatHeaders(selectedLog.request_headers)}</pre>
              </div>

              <div style={{ marginTop: '15px' }}>
                <strong>Request Body:</strong>
                <pre>{formatBody(selectedLog.request_body)}</pre>
              </div>
            </div>

            <div style={{ marginBottom: '20px', paddingTop: '20px', borderTop: '2px solid #e0e0e0' }}>
              <h4 style={{ marginBottom: '10px', color: '#2c3e50' }}>Response</h4>
              <div style={{ marginBottom: '10px' }}>
                <strong>Status:</strong>{' '}
                <span className={`badge ${selectedLog.status_code >= 200 && selectedLog.status_code < 300 ? 'badge-success' : 'badge-danger'}`}>
                  {selectedLog.status_code}
                </span>
              </div>

              <div style={{ marginTop: '15px' }}>
                <strong>Response Headers:</strong>
                <pre>{formatHeaders(selectedLog.response_headers)}</pre>
              </div>

              <div style={{ marginTop: '15px' }}>
                <strong>Response Body:</strong>
                <pre>{formatBody(selectedLog.response_body)}</pre>
              </div>
            </div>

            <div className="modal-footer">
              <button className="btn-secondary" onClick={() => setShowDetail(false)}>
                Close
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}

export default RequestLog
