import axios from 'axios'

const api = axios.create({
  baseURL: '/api/v1',
  headers: {
    'Content-Type': 'application/json',
  },
})

export const healthAPI = {
  check: () => api.get('/health'),
}

export const vhostAPI = {
  list: () => api.get('/vhosts'),
  get: (id) => api.get(`/vhosts/${id}`),
  create: (data) => api.post('/vhosts', data),
  update: (id, data) => api.put(`/vhosts/${id}`, data),
  delete: (id) => api.delete(`/vhosts/${id}`),
}

export const certAPI = {
  list: () => api.get('/certificates'),
  delete: (id) => api.delete(`/certificates/${id}`),
  downloadCA: () => {
    window.location.href = '/api/v1/certificates/ca'
  },
  replaceCA: (data) => api.post('/certificates/ca/replace', data),
}

export const logAPI = {
  list: (params) => api.get('/logs', { params }),
  get: (id) => api.get(`/logs/${id}`),
}

export default api
