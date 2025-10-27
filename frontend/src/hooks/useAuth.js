import { useState } from 'react'
import api from '../services/api'

export function useAuth() {
  const [user, setUser] = useState(null)

  const login = async (username, password, id) => {
    const res = await api.login(username, password, id)
    // Ajusta según respuesta real del backend
    setUser(res.user || res)
    return res
  }

  const logout = () => setUser(null)

  return { user, login, logout }
}
