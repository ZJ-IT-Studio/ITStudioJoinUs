export class ApiError extends Error {
  code: string
  fields?: Record<string, string>
  constructor(message: string, code = 'UNKNOWN', fields?: Record<string, string>) { super(message); this.code = code; this.fields = fields }
}

export async function api<T>(path: string, init: RequestInit = {}): Promise<T> {
  const headers = new Headers(init.headers)
  if (init.body && !(init.body instanceof FormData)) headers.set('Content-Type', 'application/json')
  const method = (init.method || 'GET').toUpperCase()
  if (!['GET','HEAD','OPTIONS'].includes(method)) {
    const cookieName = path.startsWith('/admin/') ? 'itstudio_admin_csrf' : 'itstudio_student_csrf'
    const token = document.cookie.split('; ').find(v => v.startsWith(`${cookieName}=`))?.split('=').slice(1).join('=')
    if (token) headers.set('X-CSRF-Token', decodeURIComponent(token))
  }
  const response = await fetch(`/api/v1${path}`, { ...init, headers, credentials: 'same-origin' })
  if (!response.ok) {
    const payload = await response.json().catch(() => ({}))
    throw new ApiError(payload?.error?.message || `请求失败 (${response.status})`, payload?.error?.code, payload?.error?.fields)
  }
  const contentType = response.headers.get('content-type') || ''
  if (!contentType.includes('application/json')) return response as T
  return response.json()
}

export const post = <T>(path: string, body?: unknown) => api<T>(path, { method: 'POST', body: body instanceof FormData ? body : JSON.stringify(body ?? {}) })
export const put = <T>(path: string, body: unknown) => api<T>(path, { method: 'PUT', body: JSON.stringify(body) })
export const del = <T>(path: string) => api<T>(path, { method: 'DELETE' })
