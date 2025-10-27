import { useEffect, useState } from 'react'
import api from '../services/api'

export default function GraphView() {
  const [dot, setDot] = useState('')

  useEffect(() => {
    api.getGraphDot().then(setDot).catch(() => setDot(''))
  }, [])

  return (
    <div className="p-4">
      <h3 className="text-lg font-bold mb-2">Grafo del sistema de archivos</h3>
      {dot ? (
        <pre className="bg-gray-100 p-2 rounded">{dot}</pre>
      ) : (
        <div>No hay grafo disponible</div>
      )}
    </div>
  )
}
