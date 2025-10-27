import { useEffect, useState } from 'react'
import api from '../services/api'

export default function FileViewer({ file, onClose, diskPath, partitionName }) {
  const [meta, setMeta] = useState(null)
  const [content, setContent] = useState('')
  const [loadingMeta, setLoadingMeta] = useState(false)
  const [loadingContent, setLoadingContent] = useState(false)
  const [error, setError] = useState(null)

  useEffect(() => {
    if (!file) return
    const load = async () => {
      setError(null)
      setLoadingMeta(true)
      try {
        const m = await api.statPath(diskPath, partitionName, file.path || file.name)
        setMeta(m)
      } catch (err) {
        console.error('statPath error', err)
        setError('No se pudo obtener metadata')
      } finally {
        setLoadingMeta(false)
      }
    }
    load()
    // eslint-disable-next-line
  }, [file])

  const loadContent = async () => {
    setLoadingContent(true)
    setError(null)
    try {
      // fallback to analyzer read if no dedicated endpoint
      const data = await api.readFileByCat(file.path || file.name)
      // limit preview size to avoid huge renders
      setContent(typeof data === 'string' ? data.slice(0, 200000) : String(data))
    } catch (err) {
      console.error('readFile error', err)
      setError('No se pudo leer el archivo')
    } finally {
      setLoadingContent(false)
    }
  }

  return (
    <div className="flex flex-col h-full">
      <div className="flex justify-between items-center mb-2">
        <h3 className="font-bold">{file.name}</h3>
        <div className="flex gap-2">
          <button onClick={onClose} className="px-3 py-1 bg-gray-200 rounded">Cerrar</button>
          <button onClick={loadContent} className="px-3 py-1 bg-blue-500 text-white rounded">Cargar contenido</button>
        </div>
      </div>

      <div className="mb-3 text-sm">
        {loadingMeta && <div>Cargando metadata...</div>}
        {error && <div className="text-red-600">{error}</div>}
        {meta && (
          <div className="grid grid-cols-2 gap-2">
            <div><strong>Ruta:</strong> {meta.path || file.path}</div>
            <div><strong>Tipo:</strong> {meta.type}</div>
            <div><strong>Tamaño:</strong> {meta.size ?? '—'}</div>
            <div><strong>Modificado:</strong> {meta.modified ?? '—'}</div>
            <div><strong>Creado:</strong> {meta.created ?? '—'}</div>
            <div><strong>Permisos:</strong> {meta.permissions ?? '—'}</div>
          </div>
        )}
      </div>

      <div className="flex-1 overflow-auto border rounded p-2 bg-gray-50">
        {loadingContent && <div>Cargando contenido...</div>}
        {!loadingContent && content && (
          <pre className="whitespace-pre-wrap text-sm">{content}</pre>
        )}
        {!loadingContent && !content && <div className="text-sm text-gray-500">No hay contenido cargado. Presiona "Cargar contenido" para ver una vista previa (si es texto).</div>}
      </div>
    </div>
  )
}
