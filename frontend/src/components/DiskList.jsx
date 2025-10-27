import { useEffect, useState } from 'react'
import api from '../services/api'

export default function DiskList({ onSelect }) {
  const [disks, setDisks] = useState([])
  const [path, setPath] = useState('')
  const [parts, setParts] = useState(null)

  const fetchPartitions = async () => {
    try {
      const res = await api.listPartitions(path)
      setParts(res.partitions || [])
    } catch (err) {
      setParts([])
    }
  }

  return (
    <div className="p-4">
      <h3 className="text-lg font-bold">Seleccionar disco</h3>
      <div className="flex gap-2 my-2">
        <input className="border p-2 flex-1" placeholder="Ruta al disco (ej. /home/...)" value={path} onChange={e => setPath(e.target.value)} />
        <button className="bg-btn text-white p-2 rounded" onClick={fetchPartitions}>Cargar particiones</button>
      </div>

      {parts && (
        <div className="grid gap-2">
          {parts.map((p, i) => (
            <div key={i} className="border p-2 rounded flex justify-between items-center">
              <div>
                <div className="font-semibold">{p.name || p.id}</div>
                <div className="text-sm text-gray-600">size: {p.size} start: {p.start} mounted: {String(p.isMounted)}</div>
              </div>
              <div>
                <button className="bg-green-500 text-white px-3 py-1 rounded" onClick={() => onSelect(path, p.name || p.id)}>Abrir</button>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
