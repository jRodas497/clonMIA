import { useState, useEffect } from 'react'
import api from '../services/api'

// Modal-style Login: overlay with backdrop so it doesn't push the page content
export default function Login({ onLogin, onClose, onLogout, session }) {
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [id, setId] = useState('')
  const [error, setError] = useState(null)

  useEffect(() => {
    // prefill from localStorage if available
    const su = localStorage.getItem('mia_username') || ''
    const sp = localStorage.getItem('mia_password') || ''
    const sid = localStorage.getItem('mia_id') || ''
    setUsername(su)
    setPassword(sp)
    setId(sid)
  }, [])

  const submit = async (e) => {
    e.preventDefault()
    try {
      const res = await api.login(username, password, id)
      // propagate session/result to parent
      // Expect backend to return an object with status: 'success' on success
      if (res && (res.status === 'success' || res.status === 'ok')) {
        // save credentials locally so UI can remember them
        localStorage.setItem('mia_username', username)
        localStorage.setItem('mia_password', password)
        localStorage.setItem('mia_id', id)
        // propagate normalized session object to parent
        if (typeof onLogin === 'function') onLogin({ username, id, raw: res })
        if (typeof onClose === 'function') onClose()
      } else {
        // backend returned a non-successful payload (but HTTP 200). Show message and do not close the modal.
        const msg = (res && res.message) || 'Credenciales inválidas'
        setError(msg)
      }
    } catch (err) {
      console.error('Login error:', err)
      // Network or HTTP error: show message and do NOT call onLogin/onClose
      const msg = err && err.message ? err.message : 'Error conectando al servidor'
      setError(msg)
    }
  }

  // Cerrar el modal al hacer clic fuera del panel
  const onBackdropClick = (e) => {
    if (e.target === e.currentTarget) {
      if (typeof onClose === 'function') onClose()
    }
  }

  return (
    // Backdrop
    <div onClick={onBackdropClick} className="fixed inset-0 z-50 flex items-center justify-center bg-black bg-opacity-50">
      {/* Modal panel */}
      <div className="bg-white rounded-lg shadow-lg w-full max-w-md mx-4" role="dialog" aria-modal="true">
        <div className="p-6">
          <h2 className="text-xl font-bold mb-4">Iniciar sesión</h2>
          {error && <div className="text-red-600 mb-2">{error}</div>}
          <form onSubmit={submit} className="flex flex-col gap-3">
            <input className="border p-2 rounded" placeholder="Usuario" value={username} onChange={e => setUsername(e.target.value)} />
            <input className="border p-2 rounded" placeholder="Contraseña" type="password" value={password} onChange={e => setPassword(e.target.value)} />
            <input className="border p-2 rounded" placeholder="ID Partición" value={id} onChange={e => setId(e.target.value)} />
            <div className="flex gap-2 justify-end mt-4">
              <button type="submit" className="px-4 py-2 rounded-md bg-indigo-600 text-white hover:bg-indigo-700">Entrar</button>
              {onClose && <button type="button" onClick={onClose} className="px-4 py-2 rounded-md bg-gray-500 text-white hover:bg-gray-600">Cancelar</button>}
              {onLogout && <button type="button" onClick={() => { if (typeof onLogout === 'function') onLogout(); if (typeof onClose === 'function') onClose() }} className="px-4 py-2 rounded-md bg-red-600 text-white hover:bg-red-700">Cerrar sesión</button>}
            </div>
          </form>
        </div>
      </div>
    </div>
  )
}
