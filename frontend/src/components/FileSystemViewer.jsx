import { useEffect, useState } from 'react'
import api from '../services/api'
import Modal from './Modal'
import FileViewer from './FileViewer'

export default function FileSystemViewer({ diskPath, partitionName, onClose }) {
  const [currentPath, setCurrentPath] = useState('/')      // current folder inside partition
  const [entries, setEntries] = useState([])               // children of currentPath
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState(null)
  const [selectedFile, setSelectedFile] = useState(null)   // { name, path }
  const [showFileViewer, setShowFileViewer] = useState(false)

  useEffect(() => {
    if (!diskPath || !partitionName) return
    setCurrentPath('/')
  }, [diskPath, partitionName])

  useEffect(() => {
    // whenever currentPath changes, load entries
    loadEntries(currentPath)
  }, [currentPath])

  const loadEntries = async (path) => {
    setLoading(true)
    setError(null)
    try {
      const res = await api.listPath(diskPath, partitionName, path) // implement in services/api.js
      // If backend suggests autoHome (server user's home) and we're at root, navigate there automatically
      if (res && res.autoHome && (path === '/' || path === '' || path === undefined)) {
        const auto = res.autoHome
        // request the entries for autoHome
        const res2 = await api.listPath(diskPath, partitionName, auto)
        setCurrentPath(auto)
        setEntries(res2.entries || [])
      } else {
        setEntries(res.entries || [])
      }
    } catch (err) {
      setError(err.message || 'No se pudo listar la ruta')
      setEntries([])
    } finally {
      setLoading(false)
    }
  }

  const enterFolder = (name) => {
    // normalize to avoid double slashes
    const next = (currentPath === '/' ? '' : currentPath) + '/' + name
    setCurrentPath(next)
  }

  const upTo = (segmentIndex) => {
    if (segmentIndex < 0) {
      setCurrentPath('/')
      return
    }
    const parts = currentPath.split('/').filter(Boolean)
    const newParts = parts.slice(0, segmentIndex + 1)
    setCurrentPath('/' + newParts.join('/'))
  }

  const openFile = (entry) => {
    setSelectedFile({ name: entry.name, path: (currentPath === '/' ? '' : currentPath) + '/' + entry.name })
    setShowFileViewer(true)
  }

  return (
    <div className="p-4 h-full flex flex-col">
      <div className="mb-3 flex items-center justify-between">
        <div className="text-lg font-semibold">Explorador: {partitionName}</div>
        <div className="text-sm text-gray-600">{currentPath}</div>
      </div>

      <div className="mb-3">
        {/* Breadcrumbs */}
        <nav className="text-sm text-blue-600">
          <button onClick={() => setCurrentPath('/')} className="mr-2">/</button>
          {currentPath !== '/' && currentPath.split('/').filter(Boolean).map((seg, idx, arr) => (
            <span key={idx}>
              <button className="mr-2" onClick={() => upTo(idx)}>{seg}</button>
              {idx < arr.length - 1 && <span>/</span>}
            </span>
          ))}
        </nav>
      </div>

      <div className="flex-1 overflow-auto border rounded p-2">
        {loading && <div>Cargando...</div>}
        {error && <div className="text-red-600">{error}</div>}
        {!loading && !error && (
          <div className="grid gap-2">
            {entries.length === 0 && <div className="text-sm text-gray-500">No hay elementos</div>}
            {entries.map((e, i) => (
              <div key={i} className="flex items-center justify-between p-2 border rounded">
                <div className="flex items-center gap-3">
                  <div>{e.type === 'dir' ? '📁' : '📄'}</div>
                  <div className="font-medium">{e.name}</div>
                </div>
                <div className="flex gap-2">
                  {e.type === 'dir' ? (
                    <button onClick={() => enterFolder(e.name)} className="px-3 py-1 bg-green-500 text-white rounded">Entrar</button>
                  ) : (
                    <button onClick={() => openFile(e)} className="px-3 py-1 bg-blue-500 text-white rounded">Ver</button>
                  )}
                </div>
              </div>
            ))}
          </div>
        )}
      </div>

      {/* File viewer modal */}
      {showFileViewer && selectedFile && (
        <Modal title={selectedFile.name} onClose={() => setShowFileViewer(false)}>
          <FileViewer diskPath={diskPath} partitionName={partitionName} file={selectedFile} />
        </Modal>
      )}
    </div>
  )
}